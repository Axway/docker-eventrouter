package main

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

/* const mapping = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"tweet":{
			"properties":{
				"user":{
					"type":"keyword"
				},
				"message":{
					"type":"text",
					"store": true,
					"fielddata": true
				},
				"obj": {
					"type":"object"
				},
				"image":{
					"type":"keyword"
				},
				"created":{
					"type":"date"
				},
				"tags":{
					"type":"keyword"
				},
				"location":{
					"type":"geo_point"
				},
				"suggest_field":{
					"type":"completion"
				}
			}
		}
	}
}` */
const mapping6 = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"event":{
			"properties":{
				"created":{
					"type":"date"
				}
			}
		}
	}
}`

const mapping7 = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
}`

func esInit(url string, ESQueue chan map[string]string) {
	log.Println("[ES] Initializing Elasticsearch", url)
	// Starting with elastic.v5, you must pass a context to execute each service
	ctx := context.Background()

	client, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		panic(err)
	}

	info, code, err := client.Ping(url).Do(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("[ES] Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

	esversion, err := client.ElasticsearchVersion(url)
	if err != nil {
		panic(err)
	}
	log.Printf("[ES] Elasticsearch version %s\n", esversion)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(indexName).Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}

	// Flush to make sure the documents got written.
	_, err = client.Flush().Index("xfbtransfer").Do(ctx)
	if err != nil {
		panic(err)
	}

	// ...

	// Delete an index.
	/*deleteIndex, err := client.DeleteIndex("twitter").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !deleteIndex.Acknowledged {
		// Not acknowledged
	}*/
	log.Println("[ES] Starting es loop")
	count := 1
	for {
		log.Println("[ES] Waiting MessageMessage on ESQueue...")
		event := <-ESQueue
		log.Println("[ES] Message for es")

		//t := time.Now().Format(time.RFC3339Nano)
		//event["created"] = t
		msg := convertToJSON(event)

		count++
		log.Println("[ES] msg", string(msg))
		put2, err := client.Index().
			Index(indexName).
			Type("event").
			Id("3-" + fmt.Sprintf("%d %d", time.Now().Unix(), count)).
			BodyString(string(msg)).
			Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		log.Printf("[ES] Indexed %s to index %s, type %s", put2.Id, put2.Index, put2.Type)
	}
}
