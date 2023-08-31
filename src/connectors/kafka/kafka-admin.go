package kafka

import (
	"context"
	"fmt"
	"os"
	"time"

	"axway.com/qlt-router/src/log"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func KafkaCreateTopic(ctx, url, topic string) error {
	log.Infoc(ctx, "kafka create topic", "url", url)

	conf := kafka.ConfigMap{
		"bootstrap.servers": url,
	}
	log.Infoc(ctx, "kafka New Admin", "conf", fmt.Sprintf("%+v", conf))
	k, err := kafka.NewAdminClient(&conf)
	if err != nil {
		log.Errorc(ctx, "err", err)
		return err
	}
	defer k.Close()
	defer log.Infoc(ctx, "Closed")
	ctxd, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxDur, err := time.ParseDuration("10s")
	if err != nil {
		panic("ParseDuration(10s)")
	}

	results, err := k.CreateTopics(
		ctxd,
		// Multiple topics can be created simultaneously
		// by providing more TopicSpecification structs here.
		[]kafka.TopicSpecification{{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}},
		// Admin options
		kafka.SetAdminOperationTimeout(maxDur))
	if err != nil {
		log.Errorc(ctx, "Failed to create topic", "err", err)
		return err
	}

	// Print results
	for _, result := range results {
		log.Infoc(ctx, "topic creation result", "topic", result)
	}
	return nil
}

func KafkaFlushTopicFromUrl(ctx, url, group, topic string) error {
	log.Infoc(ctx, "Opening kafka", "url", url)
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalc(ctx, "err", err)
		return err
	}
	conf := kafka.ConfigMap{
		"bootstrap.servers":                  url,
		"client.id":                          hostname + "-reader",
		"group.id":                           group,
		"auto.offset.reset":                  "latest",
		"broker.address.family":              "v4",
		"topic.metadata.refresh.interval.ms": "100",
		"allow.auto.create.topics":           true,
	}

	log.Infoc(ctx, "New Consumer", "conf", fmt.Sprintf("%+v", conf))
	k, err := kafka.NewConsumer(&conf)
	if err != nil {
		log.Errorc(ctx, "err", err)
		return err
	}
	defer k.Close()
	defer log.Infoc(ctx, "Closed")
	metadata, err := k.GetMetadata(nil, true, 100)
	if err != nil {
		log.Errorc(ctx, "metadata", "err", err)
		return err
	}
	log.Infoc(ctx, "Kafka info", "url", url, "metadata", metadata)

	err = k.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Errorc(ctx, "error subscribing topics", "err", err)
		return err
	}

	for {
		msg, err := k.ReadMessage(100 * time.Millisecond)
		if err != nil {
			if err.(kafka.Error).Code() != kafka.ErrTimedOut {
				log.Errorc(ctx, "readMessage", "err", err)
				return err
			}
			log.Errorc(ctx, "readMessage", "err", err)
			break
		}
		log.Infoc(ctx, "Flush Message", "msg", msg)
		k.CommitMessage(msg)
	}

	log.Infoc(ctx, "All Message flushed", "topic", topic)
	return nil
}
