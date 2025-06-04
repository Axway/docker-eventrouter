package awssqs

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var TestQueueName = "TestAwsSqs"
var CredentialsFile = "/.aws/credentials"
var ConfigFile = "/.aws/config"

const (
	Url_access_key_ci = "https://10.128.150.194:16660/s3_default_access_key"
	Url_secret_key_ci = "https://10.128.150.194:16660/s3_default_secret_key"
	Region_ci         = "eu-west-3"
)

func HttpsClient(t *testing.T, url string) []byte {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	msg, _ := io.ReadAll(resp.Body)
	return msg
}

func CleanQueue(t *testing.T, client SQS, queueURL string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout := time.After(5 * time.Second)
	tick := time.Tick(500 * time.Millisecond)

loop:
	for {
		select {
		case <-timeout:
			break loop
		case <-tick:
			msgs, err := client.Receive(ctx, queueURL, 10)
			if err != nil {
				t.Fatal(err)
			}
			for _, msg := range msgs {
				err = client.DeleteMessage(ctx, queueURL, *msg.ReceiptHandle)
				if err != nil {
					t.Fatal(err)
				}
			}
			if len(msgs) > 0 {
				break loop
			}
		}
	}
	return nil
}

func TestAwsSqsCreateQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	access_key := HttpsClient(t, Url_access_key_ci)
	secret_access_key := HttpsClient(t, Url_secret_key_ci)

	// Create a session instance.
	ses, err := New(SQSConfig{
		Region:          string(Region_ci),
		AccessKeyID:     string(access_key),
		SecretAccessKey: string(secret_access_key),
	})

	if err != nil {
		t.Fatal(err)
	}

	// Instantiate client.
	client := NewSQS(ses, 15*time.Second)
	_, err = client.CreateQueue(ctx, &TestQueueName)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAwsSqsSessionEnv(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	access_key := HttpsClient(t, Url_access_key_ci)
	secret_access_key := HttpsClient(t, Url_secret_key_ci)

	os.Setenv("AWS_REGION", Region_ci)
	os.Setenv("AWS_ACCESS_KEY_ID", string(access_key))
	os.Setenv("AWS_SECRET_ACCESS_KEY", string(secret_access_key))

	// Create a session instance.
	ses, err := New(SQSConfig{})

	if err != nil {
		t.Fatal(err)
	}

	// Instantiate client.
	client := NewSQS(ses, 15*time.Second)
	_, err = client.CreateQueue(ctx, &TestQueueName)
	if err != nil {
		t.Fatal(err)
	}
}

/*
	func WriteFile(t *testing.T, fileName string, lines []string) {
		f, err := os.Create(fileName) //os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //
		if err != nil {
			t.Fatal(err)
		}
		// remember to close the file
		defer f.Close()

		// create new buffer
		buffer := bufio.NewWriter(f)

		for _, line := range lines {
			_, err := buffer.WriteString(line + "\n")
			if err != nil {
				t.Fatal(err)
			}
		}

		// flush buffered data to the file
		if err := buffer.Flush(); err != nil {
			t.Fatal(err)
		}

}

	func WriteConfigFiles(t *testing.T) {
		access_key := HttpsClient(t, Url_access_key_ci)
		secret_access_key := HttpsClient(t, Url_secret_key_ci)

		lines := []string{
			"[default]",
			"aws_access_key_id = " + string(access_key),
			"aws_secret_access_key = " + string(secret_access_key),
		}

		u, _ := user.Current()
		WriteFile(t, u.HomeDir+CredentialsFile, lines)

		lines = []string{
			"[default]",
			"region = eu-west-3",
		}

		WriteFile(t, u.HomeDir+ConfigFile, lines)
	}

	func DeleteConfigFiles(t *testing.T) {
		u, _ := user.Current()
		e := os.Remove(u.HomeDir + CredentialsFile)
		if e != nil {
			log.Fatal(e)
		}

		e = os.Remove(u.HomeDir + ConfigFile)
		if e != nil {
			log.Fatal(e)
		}
	}

	func TestAwsSqsSessionSharedConfig(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a session instance.
		// see https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
		// ~/.aws/credentials and ~/.aws/config must exist and must have
		// credentials :
		// [default]
		// aws_access_key_id = <YOUR_DEFAULT_ACCESS_KEY_ID>
		// aws_secret_access_key = <YOUR_DEFAULT_SECRET_ACCESS_KEY>
		// config :
		// [default]
		// region = <REGION>
		WriteConfigFiles(t)

		ses, err := New(SQSConfig{})

		if err != nil {
			t.Fatal(err)
		}

		// Instantiate client.
		client := NewSQS(ses, 15*time.Second)
		_, err = client.CreateQueue(ctx, &TestQueueName)
		if err != nil {
			t.Fatal(err)
		}

		DeleteConfigFiles(t)
	}
*/
func TestAwsSqsSend(t *testing.T) {
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	access_key := HttpsClient(t, Url_access_key_ci)
	secret_access_key := HttpsClient(t, Url_secret_key_ci)

	// Create a session instance.
	ses, err := New(SQSConfig{
		Region:          Region_ci,
		AccessKeyID:     string(access_key),
		SecretAccessKey: string(secret_access_key),
	})

	if err != nil {
		t.Fatal(err)
	}

	// Instantiate client.
	client := NewSQS(ses, 15*time.Second)
	url, err := client.CreateQueue(ctx, &TestQueueName)
	if err != nil {
		t.Fatal(err)
	}

	defer CleanQueue(t, client, url)

	body := time.Now()
	_, err = client.Send(ctx, &SendRequest{
		QueueURL: url,
		Body:     body.String(),
		Attributes: []Attribute{
			{
				Key:   "FromEmail",
				Value: "from@example.com",
				Type:  "String",
			},
			{
				Key:   "ToEmail",
				Value: "to@example.com",
				Type:  "String",
			},
			{
				Key:   "Subject",
				Value: "this is subject",
				Type:  "String",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}
