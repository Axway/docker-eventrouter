package kafka

import (
	"context"

	"axway.com/qlt-router/src/processor"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/sirupsen/logrus"
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
	k, err := kafka.NewConsumer(&kafka.ConfigMap{"bootstrap.servers": q.Conf.Servers, "group.id": q.Conf.Group, "auto.offset.reset": "earliest"})
	if err != nil {
		log.Errorln(q.CtxS, "error creation kafka consumer", "err", err)
		return err
	}
	q.k = k

	err = q.k.SubscribeTopics([]string{q.Conf.Topic}, nil)
	if err != nil {
		log.Errorln(q.CtxS, "error subscribing topics", "err", err)
		return err
	}
	return err
}

func (q *KafkaReader) AckMsg(ack processor.EventAck) {
	ack2 := ack.(kafka.TopicPartition)
	q.k.CommitOffsets([]kafka.TopicPartition{ack2})
	return
}

func (q *KafkaReader) Ctx() string {
	return q.CtxS
}

func (q *KafkaReader) Read() ([]processor.AckableEvent, error) {
	msg, err := q.k.ReadMessage(-1)
	if err != nil {
		// The client will automatically try to recover from all errors.
		log.Printf(q.CtxS+"Consumer error: %v (%v)\n", err, msg)
		return nil, err
	}
	c := msg.TopicPartition
	// log.Printf(q.Ctx+"Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
	out := processor.AckableEvent{q, c, string(msg.Value), nil}
	return []processor.AckableEvent{out}, nil
}

func (q *KafkaReader) Close() error {
	log.Infoln(q.CtxS, "closing kafka-reader")
	err := q.k.Close()
	if err != nil {
		log.Errorln(q.CtxS, "error closing kafka-reader", "err", err)
	} else {
		log.Infoln(q.CtxS, "closed kafka-reader")
	}
	return err
}
