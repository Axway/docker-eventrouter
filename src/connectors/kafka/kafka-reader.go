package kafka

import (
	"context"
	"fmt"
	"os"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaReaderConf struct {
	Servers string
	Topic   string
	Group   string
}

type KafkaReader struct {
	CtxS string
	Conf *KafkaReaderConf
	k    *kafka.Consumer
}

func (conf *KafkaReaderConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q KafkaReader

	q.Conf = conf

	return processor.GenProcessorHelperReader(ctx, &q, p, ctl, inc, out)
}

func (c *KafkaReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *KafkaReader) Init(p *processor.Processor) error {
	q.CtxS = p.Name
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalc(q.CtxS, "hostname", "err", err)
	}

	k, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":     q.Conf.Servers,
		"client.id":             hostname + "-reader",
		"group.id":              q.Conf.Group,
		"auto.offset.reset":     "earliest",
		"enable.auto.commit":    false,
		"broker.address.family": "v4",
		//"session.timeout.ms":    "6000",
		//"heartbeat.interval.ms": 100,
	})
	if err != nil {
		log.Errorc(q.CtxS, "error creation kafka consumer", "err", err)
		return err
	}
	q.k = k

	err = q.k.SubscribeTopics([]string{q.Conf.Topic}, nil)
	if err != nil {
		log.Errorc(q.CtxS, "error subscribing topics", "err", err)
		return err
	}
	log.Infoc(q.CtxS, "connected to kafka servers as consumer", "servers", q.Conf.Servers, "topic", q.Conf.Topic)
	return err
}

func (q *KafkaReader) AckMsg(ack processor.EventAck) {
	ack2 := ack.(kafka.TopicPartition)
	log.Debugc(q.CtxS, "commiting offsets", "ack", ack2)
	committed, err := q.k.CommitOffsets([]kafka.TopicPartition{ack2})
	if err != nil {
		log.Errorc(q.CtxS, "error commiting offsets", "err", err)
	} else {
		log.Tracec(q.CtxS, "committed offsets", "ack", ack2, "partitions", committed)
	}
}

func (q *KafkaReader) Ctx() string {
	return q.CtxS
}

func (q *KafkaReader) Read() ([]processor.AckableEvent, error) {
	msg, err := q.k.ReadMessage(-1)
	if err != nil {
		// The client will automatically try to recover from all errors.
		log.Errorc(q.CtxS, "reader error", "err", err, "msg", fmt.Sprintf("%+v", msg))
		return nil, err
	}
	c := msg.TopicPartition
	log.Tracec(q.CtxS, "Message", "partition", msg.TopicPartition, "msg", string(msg.Value))
	out := processor.AckableEvent{q, c, string(msg.Value), nil}
	return []processor.AckableEvent{out}, nil
}

func (q *KafkaReader) Close() error {
	log.Infoc(q.CtxS, "closing kafka-reader")
	err := q.k.Close()
	if err != nil {
		log.Errorc(q.CtxS, "error closing kafka-reader", "err", err)
	} else {
		log.Infoc(q.CtxS, "closed kafka-reader")
	}
	return err
}
