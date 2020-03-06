# Example: Bulk Indexing from Kafka

-----

### NOTES

```
curl -X POST -H 'Content-Type: application/json' -H 'kbn-xsrf: true' 'http://localhost:5601/api/saved_objects/_export' -d '
{
  "objects": [
    { "type": "index-pattern", "id": "stocks-index-pattern" },
    { "type": "dashboard",     "id": "140b5490-5fce-11ea-a238-bf5970186390" }
  ],
  "includeReferencesDeep": true
}' > ./etc/kibana-objects.ndjson

curl -X POST -H 'kbn-xsrf: true' 'http://localhost:5601/api/saved_objects/_import?overwrite=true' --form file=@etc/kibana-objects.ndjson

open http://localhost:5601/app/kibana#/dashboard/140b5490-5fce-11ea-a238-bf5970186390
```
