package awssqs

import (
	"context"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// replace AwsSQSWriter

type AwsSQSWriter struct {
	Conf *AwsSQSWriterConf
	CtxS string

	Svc *sqs.SQS
}

type AwsSQSWriterConf struct {
	Region   string `required:"true" default:"us-west-2"`
	QueueURL string `required:"true"`
}

func (c *AwsSQSWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := AwsSQSWriter{c, p.Name, nil}

	// Specify the AWS Region and SQS queue URL
	// c.Region = "us-west-2"
	// c.QueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/my-queue"

	conn, err := processor.GenProcessorHelperWriter(context, &q, p, ctl, inc, out)
	return conn, err
}

func (c *AwsSQSWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *AwsSQSWriter) Ctx() string {
	return q.CtxS
}

func (q *AwsSQSWriter) Init(p *processor.Processor) error {
	// Create a new AWS session using the default credentials and the specified Region
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(q.Conf.Region),
	})
	if err != nil {
		log.Errorc(q.CtxS, "Error creating session:", "err", err)
		return err
	}

	// Create an SQS client with the session
	q.Svc = sqs.New(sess)
	return nil
}

func (q *AwsSQSWriter) Write(events []processor.AckableEvent) (int, error) {
	entries := make([]*sqs.SendMessageBatchRequestEntry, len(events))
	for i, e := range events {
		msg, _ := e.Msg.(string)
		entries[i] = &sqs.SendMessageBatchRequestEntry{MessageBody: aws.String(msg)}
	}

	// Create the message input object
	inputs := &sqs.SendMessageBatchInput{
		Entries:  entries,
		QueueUrl: aws.String(q.Conf.QueueURL),
	}

	// Send the message to the SQS queue
	_, err := q.Svc.SendMessageBatch(inputs)
	if err != nil {
		log.Errorc("Error sending messages:", "err", err)
		return 0, err
	}

	return len(events), nil
}

func (q *AwsSQSWriter) IsAckAsync() bool {
	return false
}

func (q *AwsSQSWriter) IsActive() bool {
	return true
}

func (q *AwsSQSWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	log.Fatalc(q.CtxS, "Not supported")
}

func (q *AwsSQSWriter) Close() error {
	return nil
}
