package postgres

import (
	"context"
	"log"
	"testing"

	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/processor"
)

func TestPGConnector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}

	processor.RegisteredProcessors.Register("mem-writer", &mem.MemWriterConf{})
	processor.RegisteredProcessors.Register("mem-reader", &mem.MemReaderConf{[]string{"zouzou", "zaza"}})
	processor.RegisteredProcessors.Register("pg-writer", &PGWriterConf{})
	processor.RegisteredProcessors.Register("pg-reader", &PGReaderConf{})

	conf, err := processor.ParseConfigRawData([]byte(`
streams:
  - name: "fr-pg-write"
    disable: false
    description: ""
    flow:
      - name: "mem-reader"
      - name: "pg-writer"
        conf:
          url: "postgresql://mypguser:mypgsecretpassword@${POSTGRESQL:-localhost}:5432/mypgdb"
          initialize: true
         
  - name: "fr-pg-read"
    disable: false
    description: ""
    flow:
      - name: "pg-reader"
        conf:
          url: "postgresql://mypguser:mypgsecretpassword@${POSTGRESQL:-localhost}:5432/mypgdb"
          readerName: "Test1"
      - name: "mem-writer"
`))
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
			return
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
	}
	for {
		op := <-ctl
		op.Log()
		if op.From.Name == "pg-reader" && op.Id == "ACK_ALL_DONE" /* && rp.Out_ack == int64(all_count)*/ {
			log.Printf("op %+v", op.From)
			break
		}
	}
	// FIXME: need to verify the message numebr and content and the different stages !!!!
	// t.Error("***Success***")
}
