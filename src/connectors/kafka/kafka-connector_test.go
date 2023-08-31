package kafka

import (
	"testing"
	"time"

	"axway.com/qlt-router/src/connectors/memtest"
	"github.com/a8m/envsubst"
)

func TestKafkaConnectorGen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping indtegration test")
		return
	}
	t.Parallel()

	// FIXME: reusing the same topic name is not working... one message is always left!
	tag := time.Now().Format("-2006-01-02T15.04.05")
	// tag = "-a2"

	url, _ := envsubst.String("${KAFKA:-localhost:9094}")

	topic := "zouzou" + tag
	err := KafkaCreateTopic(t.Name(), url, "zouzou"+tag)
	if err != nil {
		t.Fatal("Error creating topic", err)
		return
	}

	writer := &KafkaWriterConf{
		Servers: url,
		Topic:   topic,
		Group:   "g5",
	}
	reader := &KafkaReaderConf{
		Servers: url,
		Topic:   topic,
		Group:   "group" + tag,
	}
	memtest.TestConnector(t, writer, reader)
}
