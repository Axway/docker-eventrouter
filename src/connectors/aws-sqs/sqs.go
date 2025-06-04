package awssqs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSConfig struct {
	Endpoint        string
	Region          string
	Profile         string
	AccessKeyID     string
	SecretAccessKey string
}

type Attribute struct {
	Key   string
	Value string
	Type  string
}

type SendRequest struct {
	QueueURL   string
	Body       string
	Attributes []Attribute
}

func New(config SQSConfig) (*session.Session, error) {
	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Config: aws.Config{
				Endpoint: aws.String(config.Endpoint),
			},
			Profile: config.Profile,
		})
	}

	return session.NewSessionWithOptions(
		session.Options{
			Config: aws.Config{
				Credentials: credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""),
				Region:      aws.String(config.Region),
				Endpoint:    aws.String(config.Endpoint),
				//S3ForcePathStyle: aws.Bool(true),
			},
			Profile: config.Profile,
		},
	)
}

type SQS struct {
	timeout time.Duration
	client  *sqs.SQS
}

func NewSQS(session *session.Session, timeout time.Duration) SQS {
	return SQS{
		timeout: timeout,
		client:  sqs.New(session),
	}
}

func (s SQS) CreateQueue(ctx context.Context, queueName *string) (string, error) {

	if !strings.Contains(*queueName, ".fifo") {
		*queueName += ".fifo"
	}
	result, err := s.client.CreateQueueWithContext(ctx, &sqs.CreateQueueInput{
		QueueName: queueName,
		Attributes: map[string]*string{
			"FifoQueue":                 aws.String("true"),
			"ContentBasedDeduplication": aws.String("true"),
		},
	})
	if err != nil {
		return "", err
	}

	return *result.QueueUrl, nil
}

func (s SQS) PurgeQueue(ctx context.Context, queueURL string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout))
	defer cancel()

	if _, err := s.client.PurgeQueueWithContext(ctx, &sqs.PurgeQueueInput{
		QueueUrl: aws.String(queueURL),
	}); err != nil {
		return fmt.Errorf("purge: %w", err)
	}

	return nil
}

func (s SQS) GetQueues() (*sqs.ListQueuesOutput, error) {
	result, err := s.client.ListQueues(nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s SQS) GetQueueURL(ctx context.Context, queue *string) (string, error) {
	queueName := *queue + ".fifo"
	result, err := s.client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", err
	}

	return *result.QueueUrl, nil
}

func (s SQS) SendBatch(ctx context.Context, reqs []*SendRequest) ([]*string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout))
	defer cancel()

	queueUrl := ""
	id := 0
	var entries []*sqs.SendMessageBatchRequestEntry

	for _, req := range reqs {
		if queueUrl == "" {
			queueUrl = req.QueueURL
		}
		attrs := make(map[string]*sqs.MessageAttributeValue, len(req.Attributes))
		for _, attr := range req.Attributes {
			attrs[attr.Key] = &sqs.MessageAttributeValue{
				StringValue: aws.String(attr.Value),
				DataType:    aws.String(attr.Type),
			}
		}

		id++
		entries = append(entries, &sqs.SendMessageBatchRequestEntry{
			Id:                aws.String(strconv.Itoa(id)),
			MessageAttributes: attrs,
			MessageBody:       &req.Body,
			MessageGroupId:    aws.String("ER-GroupID"),
		})
	}

	res, err := s.client.SendMessageBatchWithContext(ctx, &sqs.SendMessageBatchInput{
		Entries:  entries,
		QueueUrl: aws.String(queueUrl),
	})

	if err != nil {
		return []*string{}, fmt.Errorf("send: %w", err)
	}

	var msgsId []*string
	for _, success := range res.Successful {
		msgsId = append(msgsId, success.MessageId)
	}

	return msgsId, nil
}

func (s SQS) Send(ctx context.Context, req *SendRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout))
	defer cancel()

	attrs := make(map[string]*sqs.MessageAttributeValue, len(req.Attributes))
	for _, attr := range req.Attributes {
		attrs[attr.Key] = &sqs.MessageAttributeValue{
			StringValue: aws.String(attr.Value),
			DataType:    aws.String(attr.Type),
		}
	}

	var err error
	var res *sqs.SendMessageOutput
	if strings.Contains(req.QueueURL, ".fifo") {
		res, err = s.client.SendMessageWithContext(ctx, &sqs.SendMessageInput{
			MessageAttributes: attrs,
			MessageBody:       aws.String(req.Body),
			QueueUrl:          aws.String(req.QueueURL),
			MessageGroupId:    aws.String("ER-GroupID"),
		})
	} else {
		res, err = s.client.SendMessageWithContext(ctx, &sqs.SendMessageInput{
			MessageAttributes: attrs,
			MessageBody:       aws.String(req.Body),
			QueueUrl:          aws.String(req.QueueURL),
		})
	}
	if err != nil {
		return "", fmt.Errorf("aws sqs send: %w", err)
	}

	return *res.MessageId, nil
}

func (s SQS) Receive(ctx context.Context, queueURL string, maxMsg int64) ([]*sqs.Message, error) {
	if maxMsg < 1 || maxMsg > 10 {
		return nil, fmt.Errorf("receive argument: msgMax valid values: 1 to 10: given %d", maxMsg)
	}

	if s.timeout < 1*time.Second || s.timeout > 20*time.Second {
		return nil, fmt.Errorf("receive argument: timeout valid values: 1 to 20: given %d", s.timeout)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	res, err := s.client.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   aws.Int64(maxMsg),
		MessageAttributeNames: aws.StringSlice([]string{"All"}),
	})
	if err != nil {
		return nil, fmt.Errorf("receive: %w", err)
	}

	return res.Messages, nil
}

func (s SQS) DeleteMessage(ctx context.Context, queueURL, rcvHandle string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout))
	defer cancel()

	if _, err := s.client.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(rcvHandle),
	}); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}
