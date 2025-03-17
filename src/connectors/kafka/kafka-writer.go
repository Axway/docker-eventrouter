package kafka

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"axway.com/qlt-router/src/config"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

var (
	kafkaAckQueueSize      = config.DeclareInt("connectors.kafka-writer.ackQueueSize", 1000, "Kafka ack Queue Size")
	kafkaBatchSize         = config.DeclareInt("connectors.kafka-writer.batchSize", 50, "Maximum number of messages in a batch send to Kafka")
	kafkaBatchTimeout      = config.DeclareDuration("connectors.kafka-writer.batchTimeout", "200ms", "Time limit on how often incomplete message batches will be flushed to kafka.")
	kafkaConnectionTimeout = config.DeclareDuration("connectors.kafka-writer.connectionTimeout", "60s", "How long the kafka client waits before closing the connection")
)

type KafkaWriterConf struct {
	Addresses         string
	Topic             string
	Cert, CertKey, Ca string
	User, Password    string
	SaslType          string
	Synchronous       bool
}

type KafkaWriter struct {
	CtxS   string
	Conf   *KafkaWriterConf
	Writer *kafka.Writer

	waitingAcks *processor.Channel
	acksCh      chan kafka.Message
	errorCh     chan error
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
	q.CtxS = p.Name

	if !q.Conf.Synchronous {
		q.waitingAcks = p.Chans.Create(q.CtxS+"-EventWaitingAck", kafkaAckQueueSize)
		q.acksCh = make(chan kafka.Message, kafkaAckQueueSize)
		q.errorCh = make(chan error, kafkaAckQueueSize)
	}

	return nil
}

func (q *KafkaWriter) KafkaCompletion(messages []kafka.Message, err error) {
	for _, message := range messages {
		if err != nil {
			q.errorCh <- err
		} else {
			q.acksCh <- message
		}
	}
}
func (q *KafkaWriter) DrainAcks() {
	for i := len(q.waitingAcks.C); i > 0; i-- {
		select {
		case _, ok := <-q.waitingAcks.C:
			if !ok {
				log.Debugc(q.CtxS, "acks channel closed while draining")
				return
			}
		default:
			log.Debugc(q.CtxS, "no event in acks channel while draining")
			return
		}
	}
}

func (q *KafkaWriter) DrainErrs() {
	for i := len(q.errorCh); i > 0; i-- {
		select {
		case _, ok := <-q.errorCh:
			if !ok {
				log.Debugc(q.CtxS, "error channel closed while draining")
				return
			}
		default:
			log.Debugc(q.CtxS, "no event in error channel while draining")
			return
		}
	}
}

func (q *KafkaWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	if q.Conf.Synchronous {
		log.Fatalc(q.CtxS, "Not supported. Kafka writer set as Synchronous")
	}

	done := ctx.Done()
	defer log.Infoc(q.CtxS, "Stopped processing acks")

	t := time.NewTimer(kafkaConnectionTimeout)
	timer_set := true
	// log.Debugc(q.CtxS, "Starting processing acks")
loop:
	for {
		if !timer_set {
			t.Reset(kafkaConnectionTimeout)
			timer_set = true
		}

		var event processor.AckableEvent
		var ok bool
		select {
		case event, ok = <-q.waitingAcks.C:
			// log.Debugc(q.CtxS, "Event waiting for Ack")
			if !ok {
				log.Infoc(q.CtxS, "close ack loop")
				return
			}

			if event.Msg != nil {
				select {
				case err := <-q.errorCh:
					log.Infoc(q.CtxS, "err returned by kafka", "err", err)
					q.Close()
					q.DrainAcks()
					q.DrainErrs()
					continue
				case <-q.acksCh:
					// log.Debugc(q.CtxS, "Received Ack")
					acks <- event
				case <-done:
					break loop
				}
			} else { // if empty message, sends ack back
				acks <- event
			}
		case <-t.C:
			if q.Writer != nil {
				log.Infoc(q.CtxS, "closing connection due to inactivity")
				q.Close()
				q.DrainAcks()
				q.DrainErrs()
				continue
			}
		case <-done:
			log.Infoc(q.CtxS, "close ack loop")
			break loop
		}

		timer_set = false
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
	}
}

func (q *KafkaWriter) Close() error {
	log.Infoc(q.CtxS, "Closing...")
	if q.Writer != nil {
		q.Writer.Close()
		q.Writer = nil
	}
	log.Infoc(q.CtxS, "Closed")
	return nil
}

func (q *KafkaWriter) Ctx() string {
	return q.CtxS
}

func (q *KafkaWriter) IsAckAsync() bool {
	return !q.Conf.Synchronous
}

func (q *KafkaWriter) IsActive() bool {
	return q.Writer != nil
}

func (q *KafkaWriter) InitializeKafka() {
	var mechanism sasl.Mechanism
	var err error
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

		tlsConfig = tools.TlsClientConfig(q.Conf.Ca, q.Conf.Cert, q.Conf.CertKey, "kafka-writer")
	}
	addrs := strings.Split(q.Conf.Addresses, ",")
	if q.Conf.Synchronous {
		q.Writer = &kafka.Writer{
			Addr:                   kafka.TCP(addrs...),
			Topic:                  q.Conf.Topic,
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
			Async:                  false,
			MaxAttempts:            1,
			Transport: &kafka.Transport{
				TLS:  tlsConfig,
				SASL: mechanism,
			},
			ErrorLogger: kafka.LoggerFunc(logf),
			BatchSize:   1,
		}
		log.Infoc(q.CtxS, "connected to kafka servers as producer (synchronous mode)", "servers", q.Conf.Addresses, "topic", q.Conf.Topic)
	} else {
		q.Writer = &kafka.Writer{
			Addr:                   kafka.TCP(addrs...),
			Topic:                  q.Conf.Topic,
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
			Async:                  true,
			MaxAttempts:            1,
			Completion:             q.KafkaCompletion,
			BatchTimeout:           kafkaBatchTimeout,
			BatchSize:              kafkaBatchSize,
			ErrorLogger:            kafka.LoggerFunc(logf),
			Transport: &kafka.Transport{
				TLS:  tlsConfig,
				SASL: mechanism,
			},
		}
		log.Infoc(q.CtxS, "connected to kafka servers as producer (asynchronous mode)", "servers", q.Conf.Addresses, "topic", q.Conf.Topic)
	}
}

func (q *KafkaWriter) Write(events []processor.AckableEvent) (int, error) {
	if q.Writer == nil {
		q.InitializeKafka()
	}

	n := 0
	for _, event := range events {
		if q.Writer == nil {
			log.Warnc(q.CtxS, "")
			return n, nil
		}

		if event.Msg == nil {
			log.Tracec(q.CtxS, "Event filtered", "topic", q.Conf.Topic)
			if !q.Conf.Synchronous {
				q.waitingAcks.C <- event
			}
			n++
			continue
		}
		data := []byte(event.Msg.(string))

		if err := q.Writer.WriteMessages(context.Background(), kafka.Message{Value: data}); err != nil {
			log.Errorc(q.CtxS, "error writing event", "err", err)
			q.Close()
			return n, err
		}
		log.Tracec(q.CtxS, "Wrote Event", "topic", q.Conf.Topic, "msg", event.Msg.(string))
		if !q.Conf.Synchronous {
			q.waitingAcks.C <- event
		}
		n++
	}
	return n, nil
}
