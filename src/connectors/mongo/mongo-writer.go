package mongoConnector

import (
	"context"
	"encoding/json"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoWriterConf struct {
	Url        string
	Db         string
	Collection string
}

type MongoWriter struct {
	Conf       *MongoWriterConf
	CtxS       string
	client     *mongo.Client
	collection *mongo.Collection
}

func (conf *MongoWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q MongoWriter
	q.Conf = conf
	return processor.GenProcessorHelperWriter(context, processor.ConnectorRuntimeWriter(&q), p, ctl, inc, out)
}

func (c *MongoWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *MongoWriter) Init(p *processor.Processor) error {
	client, err := mongo.NewClient(options.Client().ApplyURI(q.Conf.Url))
	if err != nil {
		log.Errorc(q.CtxS, "create ", "url", q.Conf.Url, "err", err)
		return err
	}
	q.client = client

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Errorc(q.CtxS, "connect ", "url", q.Conf.Url, "err", err)
		return err
	}

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Errorc(q.CtxS, "listDatabases ", "url", q.Conf.Url, "err", err)
	}
	log.Infoc(q.CtxS, "available databases", "databases", databases)

	collection := client.Database(q.Conf.Db).Collection(q.Conf.Collection)
	q.collection = collection
	return nil
}

func (q *MongoWriter) Close() error {
	log.Infoc(q.CtxS, "Closing...")
	err := q.client.Disconnect(context.Background())
	if err != nil {
		log.Errorc(q.CtxS, "Failed to close writer", "err", err)
		return err
	}
	log.Infoc(q.CtxS, "Closed")
	return nil
}

func (q *MongoWriter) Ctx() string {
	return q.CtxS
}

func (q *MongoWriter) IsAckAsync() bool {
	return false
}

func (q *MongoWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	log.Fatalc(q.CtxS, "Not supported")
}

func (q *MongoWriter) Write(events []processor.AckableEvent) error {
	var docs []interface{}
	for _, event := range events {
		var m interface{}
		// FIXME: this is inefficient at best
		s := event.Msg.(string)
		err := json.Unmarshal([]byte(s), &m)
		if err != nil {
			return err
		}
		msg := map[string]interface{}{"msg": m}
		docs = append(docs, msg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := q.collection.InsertMany(ctx, docs, &options.InsertManyOptions{})
	if err != nil {
		log.Errorc(q.CtxS, "insertMany", "n", len(docs), "err", err)
		return err
	}
	return nil
}
