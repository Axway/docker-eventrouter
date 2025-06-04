package awssqs

import (
	"context"
	"os"

	"axway.com/qlt-router/src/config"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/aws/aws-sdk-go/aws/session"
)

type AwsSQSReader struct {
	Conf *AwsSQSReaderConf
	CtxS string

	Svc      SQS
	QueueURL string
	ctx      context.Context
}

type AwsSQSReaderConf struct {
	Region          string
	QueueName       string `required:"true"`
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Profile         string
}

var (
	sqsReadTimeout = config.DeclareDuration("connectors.sqs-writer.sqsReadTimeout", "10s", "Sqs read timeout")
)

func (c *AwsSQSReaderConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := AwsSQSReader{
		Conf: c,
		CtxS: p.Name,
		Svc:  SQS{},
		ctx:  context,
	}

	conn, err := processor.GenProcessorHelperReader(context, &q, p, ctl, inc, out)
	return conn, err
}

func (c *AwsSQSReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *AwsSQSReader) Ctx() string {
	return q.CtxS
}

func (q *AwsSQSReader) Init(p *processor.Processor) error {
	// Create a new AWS session
	var sess *session.Session
	var err error
	if q.Conf.Region == "" || q.Conf.AccessKeyID == "" || q.Conf.SecretAccessKey == "" {
		sess, err = New(SQSConfig{})
	} else {
		sess, err = New(SQSConfig{
			Region:          q.Conf.Region,
			AccessKeyID:     q.Conf.AccessKeyID,
			SecretAccessKey: q.Conf.SecretAccessKey,
			Endpoint:        q.Conf.Endpoint,
			Profile:         q.Conf.Profile,
		})
	}

	if err != nil {
		log.Errorc(q.CtxS, "Error creating session:", "err", err)
		return err
	}

	// Create an SQS client with the session
	q.Svc = NewSQS(sess, sqsReadTimeout)

	q.QueueURL, err = q.Svc.CreateQueue(q.ctx, &q.Conf.QueueName)
	if err != nil {
		log.Errorc(q.CtxS, "Error creating queue:", "err", err)
		return err
	}
	return nil
}

func (q *AwsSQSReader) Read() ([]processor.AckableEvent, error) {
	msgs, err := q.Svc.Receive(q.ctx, q.QueueURL, 10)
	if err != nil {
		log.Errorc(q.CtxS, "amazon SQS reader error", "err", err)
		err = os.ErrDeadlineExceeded
		return nil, err
	}

	if len(msgs) == 0 {
		return nil, nil
	}

	var outputs []processor.AckableEvent
	for _, msg := range msgs {
		out := processor.AckableEvent{
			Src:   q,
			Msgid: *msg.ReceiptHandle,
			Msg:   *msg.Body,
			Orig:  nil,
		}

		outputs = append(outputs, out)
	}
	return outputs, nil
}

func (q *AwsSQSReader) IsServer() bool {
	return false
}

func (q *AwsSQSReader) AckMsg(ack processor.EventAck) {
	receiptHandle := ack.(string)
	err := q.Svc.DeleteMessage(context.Background(), q.QueueURL, receiptHandle)
	if err != nil {
		log.Errorc(q.CtxS, "Error deleting message:", "err", err)
	}
	log.Tracec(q.CtxS, "committed offsets", "ack", receiptHandle)
}

func (q *AwsSQSReader) Close() error {
	return nil
}
