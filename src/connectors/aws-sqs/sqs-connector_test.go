package awssqs

import (
	"context"
	"testing"
	"time"

	"axway.com/qlt-router/src/connectors/memtest"
)

func initSQSAndCleanQueue(t *testing.T, access_key, secret_access_key string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ses, _ := New(SQSConfig{
		Region:          string(Region_ci),
		AccessKeyID:     string(access_key),
		SecretAccessKey: string(secret_access_key),
	})

	client := NewSQS(ses, 15*time.Second)
	url, err := client.CreateQueue(ctx, &TestQueueName)
	if err != nil {
		t.Fatal(err)
	}
	CleanQueue(t, client, url)
}

func TestSqsConnectorGen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}
	t.Parallel()

	access_key := HttpsClient(t, Url_access_key_ci)
	secret_access_key := HttpsClient(t, Url_secret_key_ci)

	initSQSAndCleanQueue(t, string(access_key), string(secret_access_key))

	writer := &AwsSQSWriterConf{
		Region:          string(Region_ci),
		QueueName:       string(TestQueueName),
		AccessKeyID:     string(access_key),
		SecretAccessKey: string(secret_access_key),
	}
	reader := &AwsSQSReaderConf{
		Region:          string(Region_ci),
		QueueName:       string(TestQueueName),
		AccessKeyID:     string(access_key),
		SecretAccessKey: string(secret_access_key),
	}
	memtest.TestConnector(t, writer, reader)

	initSQSAndCleanQueue(t, string(access_key), string(secret_access_key))
}
