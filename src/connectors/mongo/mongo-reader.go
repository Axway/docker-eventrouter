package mongoConnector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoMessage struct {
	ID  primitive.ObjectID `bson:"_id" json:"id"`
	Msg map[string]string  `bson:"msg" json:"msg"`
}

type MongoMessageReader struct {
	ID   primitive.ObjectID `bson:"_id" json:"id"`
	Name string             `bson:"name" json:"name"`
	Pos  primitive.ObjectID `bson:"pos" json:"pos"`
}

var zero primitive.ObjectID

// Replace MongoReader* by your Connector Name
type MongoReader struct {
	Conf              *MongoReaderConf
	CtxS              string
	client            *mongo.Client
	collection        *mongo.Collection
	readersCollection *mongo.Collection

	ReaderId primitive.ObjectID
	Current  primitive.ObjectID
	AckPos   primitive.ObjectID
}

type MongoReaderConf struct {
	Url               string `required:"true"`
	Db                string `required:"true"`
	Collection        string `required:"true"`
	ReadersCollection string `required:"true"`
	ReaderName        string `required:"true"`
}

func (c *MongoReaderConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := MongoReader{Conf: c, CtxS: p.Name}

	if c.ReadersCollection == c.Collection {
		return nil, errors.New("ReadersCollection cannot be the same as Collection")
	}
	r, err := processor.GenProcessorHelperReader(context, &q, p, ctl, inc, out)
	return r, err
}

func (c *MongoReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *MongoReader) Init(p *processor.Processor) error {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(q.Conf.Url))
	if err != nil {
		log.Errorc(q.CtxS, "connect ", "url", q.Conf.Url, "err", err)
		return err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Errorc(q.CtxS, "ping ", "url", q.Conf.Url, "err", err)
		return err
	}

	q.client = client
	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Errorc(q.CtxS, "listDatabases ", "url", q.Conf.Url, "err", err)
		return err
	}
	log.Infoc(q.CtxS, "available databases", "databases", databases)

	collection := client.Database(q.Conf.Db).Collection(q.Conf.Collection)
	q.collection = collection

	rcoll := client.Database(q.Conf.Db).Collection(q.Conf.ReadersCollection)
	q.readersCollection = rcoll

	m := MongoMessageReader{}
	err = q.readersCollection.FindOne(ctx, bson.M{"name": q.Conf.ReaderName}).Decode(&m)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Infoc(q.CtxS, "init unknown reader, creating new one", "name", q.Conf.ReaderName, "readersCollection", q.readersCollection)
			result, err := q.readersCollection.InsertOne(ctx, MongoMessageReader{primitive.NewObjectID(), q.Conf.ReaderName, zero})
			if err != nil {
				log.Errorc(q.CtxS, "insert reader failed", "err", err)
				return err
			}
			q.ReaderId = result.InsertedID.(primitive.ObjectID)
			// q.Current = zero
		} else {
			log.Errorc(q.CtxS, "find reader failed", "name", q.Conf.ReaderName, "readersCollection", q.readersCollection, "err", err)
			return err
		}
	} else {
		q.ReaderId = m.ID
		q.Current = m.Pos
	}
	log.Infoc(q.CtxS, "reader", "name", q.Conf.ReaderName, "readerId", fmt.Sprint(m.ID), "readerPos", q.Current, "readersCollection", q.readersCollection)
	return nil
}

func (q *MongoReader) AckMsg(ack processor.EventAck) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := q.readersCollection.UpdateByID(ctx, q.ReaderId, bson.M{"$set": bson.M{"pos": ack}})
	if err != nil {
		log.Errorc(q.CtxS, "update reader ackindex failed", "err", err)
		// return err
	}
	q.AckPos = ack.(primitive.ObjectID)
}

func (m *MongoReader) Ctx() string {
	return m.CtxS
}

func (q *MongoReader) Read() ([]processor.AckableEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"_id": bson.M{"$gt": q.Current}}
	if q.Current == zero {
		filter = bson.M{}
	}

	cursor, err := q.collection.Find(ctx,
		filter,
		(&options.FindOptions{}).SetSort(bson.M{"_id": 1}).SetLimit(100))
	if err != nil {
		log.Errorc(q.CtxS, "read error", "err", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	msgs := []processor.AckableEvent{}
	log.Debugc(q.CtxS, "read", "batchLength", cursor.RemainingBatchLength())

	for item := cursor.Next(ctx); item; item = cursor.Next(ctx) {
		id := cursor.Current.Lookup("_id").ObjectID()
		msg := cursor.Current.Lookup("msg").String()
		event := processor.AckableEvent{q, id, msg, nil}
		msgs = append(msgs, event)
		q.Current = id
		log.Debugc(q.CtxS, "read messages", "_id", id, "msg", msg)
	}

	return msgs, nil
}

func (q *MongoReader) Close() error {
	log.Infoc(q.CtxS, "Closing...")
	err := q.client.Disconnect(context.Background())
	if err != nil {
		log.Errorc(q.CtxS, "Failed to close writer", "err", err)
		return err
	}
	log.Infoc(q.CtxS, "Closed")
	return nil
}
