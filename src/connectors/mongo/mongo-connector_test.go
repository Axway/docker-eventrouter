package mongoConnector

import (
	"context"
	"log"
	"testing"

	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/processor"
)

func TestMongoConnector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}

	processor.RegisteredProcessors.Register("mem-writer", &mem.MemWriterConf{})
	processor.RegisteredProcessors.Register("mem-reader", &mem.MemReaderConf{[]string{"{\"field1\":\"value1\"}", "{\"field1\":\"value2\"}"}})
	processor.RegisteredProcessors.Register("mongo-writer", &MongoWriterConf{})
	processor.RegisteredProcessors.Register("mongo-reader", &MongoReaderConf{})

	conf, err := processor.ParseConfigFile("test", "testdata/mongo-connector-test-config.ser.yml")
	if err != nil {
		t.Error("Error Parsing config:", err)
		return
	}

	ctl := make(chan processor.ControlEvent, 100)
	processors := &processor.RegisteredProcessors
	channels := processor.NewChannels()
	errorCount := 0
	for _, flow := range conf.Streams {
		_, err := flow.Start(context.Background(), context.Background(), false, ctl, channels, processors)
		if err != nil {
			t.Error("Error start flow '"+flow.Name+"'", err)
			errorCount++
		}
	}
	if errorCount > 0 {
		return
	}
	for {
		op := <-ctl
		op.Log()
		if op.From.Name == "mem-reader" && op.Id == "ACK_ALL_DONE" /* && rp.Out_ack == int64(all_count)*/ {
			log.Printf("op %+v", op.From)
			break
		}
		if op.Id == "ERROR" {
			t.Error("Error", op.Id, op.From.Name, op.Msg)
			return
		}
	}
	for {
		op := <-ctl
		op.Log()
		if op.From.Name == "mongo-reader" && op.Id == "ACK_ALL_DONE" /* && rp.Out_ack == int64(all_count)*/ {
			log.Printf("op %+v", op.From)
			break
		}
		if op.Id == "ERROR" {
			t.Error("Error", op.Id, op.From.Name, op.Msg)
			return
		}
	}

	// FIXME: need to verify the message numebr and content and the different stages !!!!
	// t.Error("***Success***")
}
