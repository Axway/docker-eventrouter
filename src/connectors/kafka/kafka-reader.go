package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type TopicPartition struct {
	Topic     string
	Partition int
	Offset    int64
}

type KafkaReaderConf struct {
	Servers           string
	Topic             string
	Group             string
	Cert, CertKey, Ca string
	User, Password    string
	SaslType          string
}

type KafkaReader struct {
	CtxS   string
	Conf   *KafkaReaderConf
	Reader *kafka.Reader
	Dialer *kafka.Dialer
}

func logf(msg string, a ...interface{}) {
	fmt.Print(time.Now().Format("2006-01-02T15:04:05.000000-07:00"))
	fmt.Print(" ERR [KAFKA] ")
	fmt.Printf(msg, a...)
	fmt.Println()
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

	var mechanism sasl.Mechanism
	if q.Conf.User != "" && q.Conf.Password != "" {
		log.Infoc(q.CtxS, "User and password set. Using SASL.", "type", q.Conf.SaslType)

		if strings.EqualFold(q.Conf.SaslType, "SCRAM-SHA-512") {
			mechanism, err = scram.Mechanism(scram.SHA512, q.Conf.User, q.Conf.Password)
			if err != nil {
				log.Fatalc(q.CtxS, "mechanism", "err", err)
			}
		} else if strings.EqualFold(q.Conf.SaslType, "SCRAM-SHA-256") {
			mechanism, err = scram.Mechanism(scram.SHA256, q.Conf.User, q.Conf.Password)
			if err != nil {
				log.Fatalc(q.CtxS, "mechanism", "err", err)
			}
		} else if strings.EqualFold(q.Conf.SaslType, "plain") || q.Conf.SaslType == "" {
			mechanism = plain.Mechanism{Username: q.Conf.User, Password: q.Conf.Password}
		} else {
			log.Fatalc(q.CtxS, "Unknonw value for SaslType (plain, scram-sha-256, scram-sha-512)", "value", q.Conf.SaslType)
		}
	}

	var tlsConfig *tls.Config
	if q.Conf.Ca != "" {
		log.Infoc(q.CtxS, "SSL configured", "CA", q.Conf.Ca)
		tlsConfig = tools.TlsClientConfig(q.Conf.Ca, q.Conf.Cert, q.Conf.CertKey, "kafka-reader")
	}
	q.Dialer = &kafka.Dialer{
		// Timeout:   10 * time.Second,
		DualStack:     true,
		ClientID:      p.Instance_id + "-" + hostname + "-" + q.Conf.Group + "-reader",
		TLS:           tlsConfig,
		SASLMechanism: mechanism,
	}

	addrs := strings.Split(q.Conf.Servers, ",")
	q.Reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        addrs,
		GroupID:        q.Conf.Group,
		Topic:          q.Conf.Topic,
		MaxBytes:       10e6, // 10MB
		Dialer:         q.Dialer,
		CommitInterval: time.Second, // flushes commits to Kafka every second
		ErrorLogger:    kafka.LoggerFunc(logf),
	})

	log.Infoc(q.CtxS, "connected to kafka servers as consumer", "servers", q.Conf.Servers, "topic", q.Conf.Topic)
	return err
}

func (q *KafkaReader) AckMsg(ack processor.EventAck) {
	ack2 := ack.(TopicPartition)
	log.Debugc(q.CtxS, "commiting offsets", "ack", ack2)

	var m kafka.Message
	m.Offset = ack2.Offset
	m.Topic = ack2.Topic
	m.Partition = ack2.Partition

	if err := q.Reader.CommitMessages(context.Background(), m); err != nil {
		log.Errorc(q.CtxS, "error commiting offsets", "err", err)
	}
	log.Tracec(q.CtxS, "committed offsets", "ack", ack2, "partitions", m.Partition)
}

func (q *KafkaReader) Ctx() string {
	return q.CtxS
}

func (q *KafkaReader) Read() ([]processor.AckableEvent, error) {
	msg, err := q.Reader.FetchMessage(context.Background())
	if err != nil {
		// The client will automatically try to recover from all errors.
		log.Errorc(q.CtxS, "reader error", "err", err, "msg", fmt.Sprintf("%+v", msg))
		return nil, err
	}

	var c TopicPartition
	c.Topic = msg.Topic
	c.Partition = msg.Partition
	c.Offset = msg.Offset

	log.Tracec(q.CtxS, "Message", "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, string(msg.Key), string(msg.Value))
	out := processor.AckableEvent{q, c, string(msg.Value), nil}
	return []processor.AckableEvent{out}, nil
}

func (q *KafkaReader) Close() error {
	log.Infoc(q.CtxS, "closing kafka-reader")
	err := q.Reader.Close()
	if err != nil {
		log.Errorc(q.CtxS, "error closing kafka-reader", "err", err)
	} else {
		log.Infoc(q.CtxS, "closed kafka-reader")
	}
	return err
}
