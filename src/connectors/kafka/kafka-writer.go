package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"

	"axway.com/qlt-router/src/config"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	kafkaCloseFlushTimeout      = config.DeclareDuration("connectors.kafka-writer.closeFlushTimeout", "15s", "Kafka close flush timeout")
	kafkaAckQueueSize           = config.DeclareInt("connectors.kafka-writer.ackQueueSize", 1000, "Kafka ack Queue Size")
	kafkaWriteDeliveryQueueSize = config.DeclareInt("connectors.kafka-writer.writeDeliveryQuerySize", 1000, "Kafka Write Delivery Queue Size")
)

type KafkaWriterConf struct {
	Servers string
	Topic   string
	Group   string
}

type KafkaWriter struct {
	CtxS string
	Conf *KafkaWriterConf
	k    *kafka.Producer
	acks *processor.Channel
	// delivery_chan chan kafka.Event
}

func (conf *KafkaWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q KafkaWriter
	q.Conf = conf
	return processor.GenProcessorHelperWriter(context, processor.ConnectorRuntimeWriter(&q), p, ctl, inc, out)
}

func (c *KafkaWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *KafkaWriter) Init(p *processor.Processor) error {
	q.CtxS = p.Name //"[KAFKA-WRITER] " //+ p.Flow.Name
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalc(q.CtxS, "hostname", "err", err)
	}
	conf := kafka.ConfigMap{
		"bootstrap.servers": q.Conf.Servers,
		"client.id":         hostname + "-writer",
		//"group.id":                           q.Conf.Group,
		"acks":                  "all",
		"broker.address.family": "v4",
		//"topic.metadata.refresh.interval.ms": "100",
	}
	log.Infoc(q.CtxS, "New Producer", "conf", fmt.Sprintf("%+v", conf))
	k, err := kafka.NewProducer(&conf)
	if err != nil {
		log.Errorc(q.CtxS, "kafka new producer error", "err", err)
	}
	q.k = k
	// q.delivery_chan = make(chan kafka.Event, kafkaWriteDeliveryQueueSize)
	q.acks = p.Chans.Create("kafka-acks", kafkaAckQueueSize)

	meta, err := k.GetMetadata(nil, true, 100)
	if err != nil {
		log.Errorc(q.CtxS, "fetch metadata", "err", err)
		return err
	}
	log.Infoc(q.CtxS, "metadata", "meta", meta)
	return nil
}

func (q *KafkaWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	// Delivery report handler for produced messages
	// events := q.k.Events()
	done := ctx.Done()
	defer log.Infoc(q.CtxS, "Stopped processing acks")
loop:
	for {
		// log.Debugln(q.CtxS, "Starting processing ack")
		var event processor.AckableEvent
		var ok bool
		select {
		case event, ok = <-q.acks.C:
			// log.Debugln(q.CtxS, "Received Ack", event)
			if !ok {
				log.Infoc(q.CtxS, "close ack loop")
				return
			}
		case <-done:
			break loop
		}
		// log.Debugln(q.CtxS, "Wait Events")
		select {
		case e := <-q.k.Events():
			// log.Debugln(q.CtxS, "kafka event", e.String())
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					// FIXME: failure: should resend !!!
					log.Fatalc(q.CtxS, "Delivery failed", "partition", fmt.Sprintf("%+v", ev.TopicPartition))
				} else {
					// log.Printf(q.Ctx+"Delivered message to %v\n", ev.TopicPartition)
					/*fmt.Printf("Delivered message to topic %s [%d] at offset %v\n",
					*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)*/
				}
				log.Debugc(q.CtxS, "Received Ack", "event", event)
				acks <- event
			}
		case <-done:
			break loop
		}
	}
}

func (q *KafkaWriter) Close() error {
	log.Infoc(q.CtxS, "Closing...")
	n := q.k.Flush(int(kafkaCloseFlushTimeout.Milliseconds()))
	q.k.Close()
	if n != 0 {
		log.Errorc(q.CtxS, "Failed to close writer", "n", n)
		return errors.New("Unfinished work")
	}
	log.Infoc(q.CtxS, "Closed")
	return nil
}

/*func (q *KafkaConsumer) AckMsg(msgid int64) {
	return
}*/

func (q *KafkaWriter) Ctx() string {
	return q.CtxS
}

func (q *KafkaWriter) IsAckAsync() bool {
	return true
}

func (q *KafkaWriter) IsActive() bool {
	return true
}

/*func (q *KafkaWriter) PrepareEvent(event *processor.AckableEvent) (string, error) {
	msg := event.Msg.([]byte)
	return string(msg), nil
}*/

func (q *KafkaWriter) Write(events []processor.AckableEvent) (int, error) {
	// datas := processor.PrepareEvents(q, events)
	// log.Debugln(q.CtxS, "Writing Events")
	n := 0
	for _, event := range events {
		// log.Debugln(q.CtxS, "Writing Event", "msg", event.Msg)
		data := []byte(event.Msg.(string))
		err := q.k.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &q.Conf.Topic, Partition: kafka.PartitionAny},
			Value:          data,
		}, nil)
		if err != nil {
			log.Errorc(q.CtxS, "error writing event", "err", err)
			return n, err
		}
		log.Tracec(q.CtxS, "Wrote Event", "topic", q.Conf.Topic, "msg", event.Msg.(string))
		q.acks.C <- event
		n++
	}
	// log.Debugln(q.CtxS, "Flush")
	// count := q.k.Flush(500)
	// log.Debugc(q.CtxS, "Flush", "count", len(events), "nonFlushed", count)
	return n, nil
}
