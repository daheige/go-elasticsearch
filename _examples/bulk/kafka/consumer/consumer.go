// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package consumer

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/segmentio/kafka-go"

	"github.com/elastic/go-elasticsearch/v8/esutil"
)

type Consumer struct {
	BrokerURL string
	TopicName string

	Indexer esutil.BulkIndexer
	reader  *kafka.Reader

	totalMessages int64
	totalErrors   int64
	totalBytes    int64
}

func (c *Consumer) Run(ctx context.Context) (err error) {
	if c.Indexer == nil {
		panic(fmt.Sprintf("%T.Indexer is nil", c))
	}

	c.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{c.BrokerURL},
		GroupID: "go-elasticsearch-demo",
		Topic:   c.TopicName,
		// MinBytes: 1e+6, // 1MB
		// MaxBytes: 5e+6, // 5MB

		ReadLagInterval: 1 * time.Second,
	})

	for {
		msg, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			return fmt.Errorf("reader: %s", err)
		}
		// log.Printf("%v/%v/%v:%s\n", msg.Topic, msg.Partition, msg.Offset, string(msg.Value))

		if err := c.Indexer.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action: "index",
				Body:   bytes.NewReader(msg.Value),
				OnSuccess: func(item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
					// log.Printf("Indexed %s/%s", res.Index, res.DocumentID)
				},
				OnFailure: func(item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						// log.Printf("ERROR: %s", err)
					} else {
						// log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			}); err != nil {
			return fmt.Errorf("indexer: %s", err)
		}
	}
	c.reader.Close()
	c.Indexer.Close(context.Background())

	return nil
}

func (c *Consumer) Report() {
	if c.reader == nil || c.Indexer == nil {
		return
	}
	readerStats := c.reader.Stats()
	indexerStats := c.Indexer.Stats()

	c.totalMessages += readerStats.Messages
	c.totalErrors += readerStats.Errors

	fmt.Printf(
		"lagging=%-*s  |   received=%-*s |   errors=%-*s  |   added=%-*s |   flushed=%-*s |   failed=%-*s",
		10, humanize.Comma(readerStats.Lag),
		10, humanize.Comma(c.totalMessages),
		10, humanize.Comma(c.totalErrors),
		10, humanize.Comma(int64(indexerStats.NumAdded)),
		10, humanize.Comma(int64(indexerStats.NumFlushed)),
		0, humanize.Comma(int64(indexerStats.NumFailed)))
}
