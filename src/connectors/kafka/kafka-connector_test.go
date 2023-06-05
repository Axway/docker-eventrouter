package kafka

import (
	"context"
	"fmt"
	"testing"

	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

func TestKafkaConnector(t *testing.T) {
	processor.RegisteredProcessors.Register("mem-writer", &mem.MemWriterConf{})
	processor.RegisteredProcessors.Register("mem-reader", &mem.MemReaderConf{[]string{"zouzou", "zaza"}})
	processor.RegisteredProcessors.Register("kafka-writer", &KafkaWriterConf{})
	processor.RegisteredProcessors.Register("kafka-reader", &KafkaReaderConf{})

	conf, err := processor.ParseConfigRawData([]byte(`
streams:
  - name: "fr-kafka-write"
    disable: false
    description: ""
    flow:
      - name: "mem-reader"
        conf:
          filename: "./data/sample10.xml"
          zou: ""
      - name: "kafka-writer"
        conf:
          servers: "localhost:9093"
          topic: "zouzou"
          group: "g1"
  - name: "fr-kafka-read"
    disable: false
    description: ""
    flow:
      - name: "kafka-reader"
        conf:
          servers: "localhost:9093"
          topic: "zouzou"
          group: "g1"
      - name: "mem-writer"
        conf:
          filename: "./data/filename-kafka"
`))
	if err != nil {
		t.Error("Error Parsing config:", err)
		return
	}

	ctl := make(chan processor.ControlEvent, 100)
	processors := &processor.RegisteredProcessors
	channels := processor.NewChannels()
	for _, flow := range conf.Streams {
		_, err := flow.Start(context.Background(), false, ctl, channels, processors)
		if err != nil {
			t.Error("Error start flow '"+flow.Name+"'", err)
		}
	}

	for {
		op := <-ctl
		op.Log()
		if op.From.Name == "mem-reader" && op.Id == "ACK_ALL_DONE" /* && rp.Out_ack == int64(all_count)*/ {
			log.Infoc("test", "op ", "from", fmt.Sprint("%+v", op.From))
			break
		}
	}
	for {
		op := <-ctl
		op.Log()
		if op.From.Name == "kafka-reader" && op.Id == "ACK_ALL_DONE" /* && rp.Out_ack == int64(all_count)*/ {
			log.Infoc("test", "op ", "from", fmt.Sprint("%+v", op.From))
			break
		}
	}
	// FIXME: need to verify the message numebr and content and the different stages !!!!
	// t.Error("***Success***")
}
