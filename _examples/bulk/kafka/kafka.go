// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// +build ignore

package main

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/dustin/go-humanize"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmelasticsearch"

	"github.com/elastic/go-elasticsearch/v8/_examples/bulk/kafka/consumer"
	"github.com/elastic/go-elasticsearch/v8/_examples/bulk/kafka/producer"
)

var (
	brokerURL string

	topicName  = "stocks"
	topicParts = 4
	msgRate    int

	indexName    = "stocks"
	numConsumers = 4
	flushBytes   = 0 // Default
	numWorkers   = 0 // Default
	indexerError error
)

func init() {
	if v := os.Getenv("KAFKA_URL"); v != "" {
		brokerURL = v
	} else {
		brokerURL = "localhost:9092"
	}
	flag.IntVar(&msgRate, "rate", 1000, "Producer rate (msg/sec)")
	flag.Parse()
}

func main() {
	log.SetFlags(0)

	go func() { log.Println(http.ListenAndServe("localhost:6060", nil)) }()

	var (
		wg  sync.WaitGroup
		ctx = context.Background()

		producers []*producer.Producer
		consumers []*consumer.Consumer
		indexers  []esutil.BulkIndexer
	)

	done := make(chan os.Signal)
	signal.Notify(done, os.Interrupt)
	go func() { <-done; log.Println("\n"); os.Exit(0) }()

	producers = append(producers,
		&producer.Producer{
			BrokerURL:   brokerURL,
			TopicName:   topicName,
			TopicParts:  topicParts,
			MessageRate: msgRate})

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		RetryOnStatus: []int{502, 503, 504, 429}, // Add 429 to the list of retryable statuses
		RetryBackoff:  func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
		MaxRetries:    5,
		EnableMetrics: true,
		Transport:     apmelasticsearch.WrapRoundTripper(http.DefaultTransport),
	})
	if err != nil {
		log.Fatalf("Error: NewClient(): %s", err)
	}
	expvar.Publish("go-elasticsearch", expvar.Func(func() interface{} { m, _ := es.Metrics(); return m }))

	idx, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:      indexName,
		Client:     es,
		NumWorkers: numWorkers,
		FlushBytes: int(flushBytes),
		Flusher:    &InstrumentedFlusher{flusher: &esutil.BulkIndexerDefaultFlusher{Client: es}},
		OnError:    func(err error) { indexerError = err; apm.DefaultTracer.NewError(err).Send() },
	})
	if err != nil {
		log.Fatalf("ERROR: NewBulkIndexer(): %s", err)
	}
	indexers = append(indexers, idx)

	for i := 1; i <= numConsumers; i++ {
		consumers = append(consumers,
			&consumer.Consumer{
				BrokerURL: brokerURL,
				TopicName: topicName,
				Indexer:   indexers[0]})
	}

	reporter := time.NewTicker(500 * time.Millisecond)
	defer reporter.Stop()
	go func() {
		fmt.Println("Initializing...")
		for {
			select {
			case <-reporter.C:
				fmt.Print(report(producers, consumers, indexers))
			}
		}
	}()

	errcleaner := time.NewTicker(10 * time.Second)
	defer errcleaner.Stop()
	go func() {
		for {
			select {
			case <-errcleaner.C:
				indexerError = nil
			}
		}
	}()

	if err := producers[0].CreateTopic(ctx); err != nil {
		log.Fatalf("ERROR: Producer: %s", err)
	}

	for _, c := range consumers {
		wg.Add(1)
		go func(c *consumer.Consumer) {
			defer wg.Done()
			if err := c.Run(ctx); err != nil {
				log.Fatalf("ERROR: Consumer: %s", err)
			}
		}(c)
	}

	time.Sleep(5 * time.Second) // Leave some room for consumers to connect
	for _, p := range producers {
		wg.Add(1)
		go func(p *producer.Producer) {
			defer wg.Done()
			if err := p.Run(ctx); err != nil {
				log.Fatalf("ERROR: Producer: %s", err)
			}
		}(p)
	}

	wg.Wait()

	fmt.Print(report(producers, consumers, indexers))
}

func report(
	producers []*producer.Producer,
	consumers []*consumer.Consumer,
	indexers []esutil.BulkIndexer,
) string {
	var (
		b strings.Builder

		value    string
		currRow  = 1
		numCols  = 6
		colWidth = 20

		divider = func(last bool) {
			fmt.Fprintf(&b, "\033[%d;0H", currRow)
			fmt.Fprint(&b, "┣")
			for i := 1; i <= numCols; i++ {
				fmt.Fprint(&b, strings.Repeat("━", colWidth))
				if last && i == 5 {
					fmt.Fprint(&b, "┷")
					continue
				}
				if i < numCols {
					fmt.Fprint(&b, "┿")
				}
			}
			fmt.Fprint(&b, "┫")
			currRow++
		}
	)

	fmt.Print("\033[2J\033[K")
	fmt.Printf("\033[%d;0H", currRow)

	fmt.Fprint(&b, "┏")
	for i := 1; i <= numCols; i++ {
		fmt.Fprint(&b, strings.Repeat("━", colWidth))
		if i < numCols {
			fmt.Fprint(&b, "┯")
		}
	}
	fmt.Fprint(&b, "┓")
	currRow++

	for i, p := range producers {
		fmt.Fprintf(&b, "\033[%d;0H", currRow)
		value = fmt.Sprintf("Producer %d", i+1)
		fmt.Fprintf(&b, "┃ %-*s│", colWidth-1, value)
		s := p.Stats()
		value = fmt.Sprintf("duration=%s", s.Duration.Truncate(time.Second))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("msg/sec=%s", humanize.FtoaWithDigits(s.Throughput, 2))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("sent=%s", humanize.Comma(int64(s.TotalMessages)))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("bytes=%s", humanize.Bytes(uint64(s.TotalBytes)))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("errors=%s", humanize.Comma(int64(s.TotalErrors)))
		fmt.Fprintf(&b, " %-*s┃", colWidth-1, value)
		currRow++
	}

	divider(false)

	for i, c := range consumers {
		fmt.Fprintf(&b, "\033[%d;0H", currRow)
		value = fmt.Sprintf("Consumer %d", i+1)
		fmt.Fprintf(&b, "┃ %-*s│", colWidth-1, value)
		s := c.Stats()
		value = fmt.Sprintf("lagging=%s", humanize.Comma(s.TotalLag))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("msg/sec=%s", humanize.FtoaWithDigits(s.Throughput, 2))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("received=%s", humanize.Comma(s.TotalMessages))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("bytes=%s", humanize.Bytes(uint64(s.TotalBytes)))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("errors=%s", humanize.Comma(s.TotalErrors))
		fmt.Fprintf(&b, " %-*s┃", colWidth-1, value)
		currRow++
		divider(i == len(consumers)-1)
	}

	for i, x := range indexers {
		fmt.Fprintf(&b, "\033[%d;0H", currRow)
		value = fmt.Sprintf("Indexer %d", i+1)
		fmt.Fprintf(&b, "┃ %-*s│", colWidth-1, value)
		s := x.Stats()
		value = fmt.Sprintf("added=%s", humanize.Comma(int64(s.NumAdded)))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("flushed=%s", humanize.Comma(int64(s.NumFlushed)))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		value = fmt.Sprintf("failed=%s", humanize.Comma(int64(s.NumFailed)))
		fmt.Fprintf(&b, " %-*s│", colWidth-1, value)
		if indexerError != nil {
			value = "err=" + indexerError.Error()
			if len(value) > 2*colWidth {
				value = value[:2*colWidth]
			}
		} else {
			value = ""
		}
		fmt.Fprintf(&b, " %-*s┃", 2*colWidth, value)
		currRow++
		if i < len(indexers)-1 {
			divider(true)
		}
	}

	fmt.Fprintf(&b, "\033[%d;0H", currRow)
	fmt.Fprint(&b, "┗")
	for i := 1; i <= numCols; i++ {
		fmt.Fprint(&b, strings.Repeat("━", colWidth))
		if i == 5 {
			fmt.Fprint(&b, "━")
			continue
		}
		if i < numCols {
			fmt.Fprint(&b, "┷")
		}
	}
	fmt.Fprint(&b, "┛")
	currRow++

	return b.String()
}

type InstrumentedFlusher struct {
	flusher esutil.BulkIndexerFlusher
}

func (f *InstrumentedFlusher) Flush(ctx context.Context, req esapi.BulkRequest) (*esapi.Response, error) {
	txn := apm.DefaultTracer.StartTransaction("Bulk", "indexing")
	defer txn.End()

	ctx = apm.ContextWithTransaction(ctx, txn)
	res, err := f.flusher.Flush(ctx, req)
	if err != nil {
		apm.CaptureError(ctx, err).Send()
	}
	txn.Result = res.Status()
	return res, err
}
