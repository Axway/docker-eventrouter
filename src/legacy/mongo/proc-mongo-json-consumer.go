package mongo

import (
	"context"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConsumerConf struct {
	url        string
	db         string
	collection string
	batchsize  int
}

type MongoConsumer struct {
	ctx string
}

func (conf *MongoConsumerConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) {
	var q MongoConsumer
	q.ctx = "[MONGODB] " + p.Flow.Name
	client, err := mongo.NewClient(options.Client().ApplyURI(conf.url))
	if err != nil {
		log.Fatal(q.ctx, "create "+conf.url, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(q.ctx, "connect "+conf.url, err)
	}
	defer client.Disconnect(ctx)

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Fatal(q.ctx, "listDatabases "+conf.url, err)
	}
	log.Println(q.ctx, "available databases", databases)

	collection := client.Database(conf.db).Collection(conf.collection)

	var events []processor.AckableEvent
	var docs []interface{}
	ctx = context.Background()
	done := ctx.Done()
	for {
		flush := false
		if len(events) > 0 {
			select {
			case event := <-inc:
				events = append(events, event)
				docs = append(docs, event.Msg.(map[string]string))
			default:
				flush = true
			}
		} else {
			select {
			case event := <-inc:
				events = append(events, event)
				docs = append(docs, event.Msg.(map[string]string))
			case <-done:
				log.Infoln(q.ctx, "done")
				return
			}
		}

		if conf.batchsize > 0 && len(events) > conf.batchsize {
			flush = true
		}

		if flush {
			// log.Debug(q.ctx, "insertMany ", len(docs))
			_, err := collection.InsertMany(ctx, docs)
			if err != nil {
				log.Fatal(q.ctx, "insertMany ", len(docs), " ", err)
			}

			for _, event := range events {
				event.Src.AckMsg(event.Orig.Msgid)
			}
			docs = nil
			events = nil
		}
	}
}
