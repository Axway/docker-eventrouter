package elasticsearch

import (
	"context"
	"fmt"
	"time"

	"axway.com/qlt-router/src/processor"
	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

/*
const mapping = `

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
	}`
*/
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

type EsConsumerConf struct {
	url       string
	indexName string
}

type EsConsumer struct {
	ctx string
}

func (conf *EsConsumerConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, ESQueue, out chan processor.AckableEvent) {
	p.Conf = conf
	var q EsConsumer

	q.ctx = "[ES]"

	log.Println(q.ctx, "Initializing Elasticsearch", conf.url)
	// Starting with elastic.v5, you must pass a context to execute each service

	client, err := elastic.NewClient(elastic.SetURL(conf.url))
	if err != nil {
		panic(err)
	}

	info, code, err := client.Ping(conf.url).Do(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf(q.ctx, "Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

	esversion, err := client.ElasticsearchVersion(conf.url)
	if err != nil {
		panic(err)
	}
	log.Printf(q.ctx, "Elasticsearch version %s\n", esversion)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(conf.indexName).Do(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(conf.indexName).Do(ctx)
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
	log.Println(q.ctx, "Starting es loop")
	count := 1
	done := ctx.Done()
	for {
		log.Println(q.ctx, "Waiting MessageMessage on ESQueue...")
		select {
		case event := <-ESQueue:
			log.Println(q.ctx, "Message for es")

			// t := time.Now().Format(time.RFC3339Nano)
			// event["created"] = t
			msg := processor.ConvertToJSON(event.Msg.(map[string]string))

			count++
			log.Println(q.ctx, "msg", string(msg))
			put2, err := client.Index().
				Index(conf.indexName).
				Type("event").
				Id("3-" + fmt.Sprintf("%d %d", time.Now().Unix(), count)).
				BodyString(string(msg)).
				Do(ctx)
			if err != nil {
				// Handle error
				panic(err)
			}
			log.Printf(q.ctx, "Indexed %s to index %s, type %s", put2.Id, put2.Index, put2.Type)
		case <-done:
			log.Infoln(q.ctx, "done")
			return
		}
	}
}
