package awssqs

import (
	"context"

	"axway.com/qlt-router/src/config"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/aws/aws-sdk-go/aws/session"
)

type AwsSQSWriter struct {
	Conf *AwsSQSWriterConf
	CtxS string

	Svc      SQS
	ctx      context.Context
	cancel   context.CancelFunc
	QueueURL string
	initDone bool

	waitingAcks *processor.Channel
	msgCh       chan string
	acksCh      chan string
	errorCh     chan error
}

type AwsSQSWriterConf struct {
	Region          string
	QueueName       string `required:"true"`
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Profile         string
}

var (
	sqsAckQueueSize = config.DeclareInt("connectors.sqs-writer.sqsAckQueueSize", 1000, "Sqs ack Queue Size")
	sqsWriteTimeout = config.DeclareDuration("connectors.sqs-writer.sqsWriteTimeout", "10s", "Sqs write timeout")
)

func (c *AwsSQSWriterConf) Start(context2 context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := AwsSQSWriter{
		Conf:     c,
		CtxS:     p.Name,
		Svc:      SQS{},
		initDone: false,
	}

	q.ctx, q.cancel = context.WithCancel(context.Background())
	conn, err := processor.GenProcessorHelperWriter(context2, &q, p, ctl, inc, out)
	return conn, err
}

func (c *AwsSQSWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *AwsSQSWriter) Ctx() string {
	return q.CtxS
}

func (q *AwsSQSWriter) LaunchSqsWriteLoop() {
	if !q.initDone {
		go func() {
		loop:
			for {
				select {
				case msg := <-q.msgCh:
					_, err := q.Svc.Send(q.ctx, &SendRequest{
						QueueURL: q.QueueURL,
						Body:     msg,
					})
					if err != nil {
						log.Infoc(q.CtxS, "err returned by AWS Sqs send", "err", err)
						q.initDone = false
						q.errorCh <- err
					} else {
						q.acksCh <- msg
					}
				case <-q.ctx.Done():
					q.initDone = false
					break loop
				}
			}
		}()
		q.initDone = true
	}
}

func (q *AwsSQSWriter) Init(p *processor.Processor) error {
	if !q.initDone {
		q.waitingAcks = p.Chans.Create(q.CtxS+"-EventWaitingAck", sqsAckQueueSize)
		q.acksCh = make(chan string, sqsAckQueueSize)
		q.errorCh = make(chan error, sqsAckQueueSize)
		q.msgCh = make(chan string, sqsAckQueueSize)

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
		q.Svc = NewSQS(sess, sqsWriteTimeout)

		q.QueueURL, err = q.Svc.CreateQueue(q.ctx, &q.Conf.QueueName)
		if err != nil {
			log.Errorc(q.CtxS, "Error creating queue:", "err", err)
			return err
		}

		q.LaunchSqsWriteLoop()
	}

	return nil
}

func (q *AwsSQSWriter) DrainAcks() {
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

func (q *AwsSQSWriter) DrainErrs() {
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

func (q *AwsSQSWriter) Write(events []processor.AckableEvent) (int, error) {
	if !q.IsActive() {
		q.initDone = true
	}
	for _, e := range events {
		if e.Msg == nil {
			log.Tracec(q.CtxS, "Event filtered", "QueueURL", q.Conf.QueueName)
			q.waitingAcks.C <- e
			continue
		}
		q.msgCh <- e.Msg.(string)
		q.waitingAcks.C <- e
	}
	return len(events), nil
}

func (q *AwsSQSWriter) IsAckAsync() bool {
	return true
}

func (q *AwsSQSWriter) IsActive() bool {
	return q.initDone
}

func (q *AwsSQSWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	done := ctx.Done()
	defer log.Infoc(q.CtxS, "Stopped processing acks")

	log.Debugc(q.CtxS, "Starting processing acks")
loop:
	for {

		select {
		case event, ok := <-q.waitingAcks.C:
			if !ok {
				log.Infoc(q.CtxS, "close ack loop")
				return
			}

			if event.Msg != nil {
				select {
				case err := <-q.errorCh:
					log.Infoc(q.CtxS, "err returned by AWS Sqs", "err", err)
					q.DrainAcks()
					q.DrainErrs()
					errs <- err
					continue
				case <-q.acksCh:
					acks <- event
				case <-done:
					break loop
				}
			} else { // if empty message, sends ack back
				acks <- event
			}

		case <-done:
			log.Infoc(q.CtxS, "close ack loop")
			break loop
		}
	}
}

func (q *AwsSQSWriter) Close() error {
	if q.IsActive() {
		q.cancel()
	}
	q.initDone = false
	return nil
}
