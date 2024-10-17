package kafka

import (
	"context"
	"crypto/tls"
	"strings"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

var (
/*FIXME: disable asynchronous mode*/
// kafkaAckQueueSize = config.DeclareInt("connectors.kafka-writer.ackQueueSize", 1000, "Kafka ack Queue Size")
)

type KafkaWriterConf struct {
	Addresses         string
	Topic             string
	Cert, CertKey, Ca string
	User, Password    string
	SaslType          string
}

type KafkaWriter struct {
	CtxS   string
	Conf   *KafkaWriterConf
	Writer *kafka.Writer
	
	/*FIXME: disable asynchronous mode*/
	//sentMess *processor.Channel
	//acksCh   chan kafka.Message
	//errorCh  chan error
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
	
	/*FIXME: disable asynchronous mode*/
	//q.sentMess = p.Chans.Create("kafka-sent", kafkaAckQueueSize)
	//q.acksCh = make(chan kafka.Message, kafkaAckQueueSize)
	//q.errorCh = make(chan error, kafkaAckQueueSize)

	return nil
}

/*FIXME: disable asynchronous mode*/
/*func (q *KafkaWriter) KafkaCompletion(messages []kafka.Message, err error) {
	for _, message := range messages {
		if err != nil {
			q.errorCh <- err
		} else {
			q.acksCh <- message
		}
	}
}*/

func (q *KafkaWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	// Delivery report handler for produced messages
	// events := q.k.Events()
	log.Fatalc(q.CtxS, "Not supported")
	/*FIXME: disable asynchronous mode*/
/*
	done := ctx.Done()
	defer log.Infoc(q.CtxS, "Stopped processing acks")
loop:
	for {
		// log.Debugc(q.CtxS, "Starting processing ack")
		var event processor.AckableEvent
		var ok bool
		select {
		case event, ok = <-q.sentMess.C:
			// log.Debugc(q.CtxS, "Event waiting for Ack")
			if !ok {
				log.Infoc(q.CtxS, "close ack loop")
				return
			}
		case <-done:
			break loop
		}

		if event.Msg != nil {
			select {
			case err := <-q.errorCh:
				log.Infoc(q.CtxS, "err returned by kafka", "err", err)
			case <-q.acksCh:
				// log.Debugc(q.CtxS, "Received Ack")
				acks <- event
			case <-done:
				break loop
			}
		} else { // if empty message, sends ack back
			acks <- event
		}
	}
*/
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
	return false
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
	q.Writer = &kafka.Writer{
		Addr:                   kafka.TCP(addrs...),
		Topic:                  q.Conf.Topic,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		Async:                  false,
		MaxAttempts:            1,
		/*FIXME: disable asynchronous mode*/
		//Completion:             q.KafkaCompletion,
		Transport: &kafka.Transport{
			TLS:  tlsConfig,
			SASL: mechanism,
		},
		ErrorLogger: kafka.LoggerFunc(logf),
		BatchSize:   1,
	}
	log.Infoc(q.CtxS, "connected to kafka servers as producer", "servers", q.Conf.Addresses, "topic", q.Conf.Topic)
}

func (q *KafkaWriter) Write(events []processor.AckableEvent) (int, error) {

	if q.Writer == nil {
		q.InitializeKafka()
	}

	n := 0
	for _, event := range events {
		if event.Msg == nil {
			log.Tracec(q.CtxS, "Event filtered", "topic", q.Conf.Topic)
			/*FIXME: disable asynchronous mode*/
			//q.sentMess.C <- event
			n++
			continue
		}
		data := []byte(event.Msg.(string))

		if q.Writer == nil {
			log.Warnc(q.CtxS, "")
			return n, nil
		}
		if err := q.Writer.WriteMessages(context.Background(), kafka.Message{Value: data}); err != nil {
			log.Errorc(q.CtxS, "error writing event", "err", err)
			return n, err
		}
		log.Tracec(q.CtxS, "Wrote Event", "topic", q.Conf.Topic, "msg", event.Msg.(string))
		/*FIXME: disable asynchronous mode*/
		//q.sentMess.C <- event
		n++
	}
	return n, nil
}
