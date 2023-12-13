package kafka

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"axway.com/qlt-router/src/log"
	"github.com/segmentio/kafka-go"
)

func KafkaCreateTopic(ctx, url, topic string) error {
	log.Infoc(ctx, "kafka create topic", "url", url)

	conn, err := kafka.Dial("tcp", url)
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		panic(err.Error())
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		log.Errorc(ctx, "Failed to create topic", "err", err)
		return err
	}

	return nil
}

func KafkaFlushTopicFromUrl(ctx, url, group, topic string) error {
	log.Infoc(ctx, "Opening kafka", "url", url)
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalc(ctx, "hostname", "err", err)
	}

	dialer := &kafka.Dialer{
		// Timeout:   10 * time.Second,
		DualStack: true,
		ClientID:  hostname + "-reader",
		//TLS: tls.Config
	}

	log.Infoc(ctx, "New Reader")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{url},
		GroupID:        group,
		Topic:          topic,
		MaxBytes:       10e6, // 10MB
		Dialer:         dialer,
		CommitInterval: time.Second, // flushes commits to Kafka every second
	})
	defer reader.Close()
	defer log.Infoc(ctx, "Closed")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Errorc(ctx, "readMessage", "err", err)
			break
		}
		log.Infoc(ctx, "Flush Message", "msg", msg)
		reader.CommitMessages(context.Background(), msg)
	}

	log.Infoc(ctx, "All Message flushed", "topic", topic)
	return nil
}
