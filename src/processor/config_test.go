package processor_test

import (
	"log"
	"testing"

	"axway.com/qlt-router/src/connectors/file"
	"axway.com/qlt-router/src/processor"
	"gopkg.in/yaml.v3"
)

func TestParseConfigRaw(t *testing.T) {
	t.Parallel()

	processor.RegisteredProcessors.Register("file_raw_writer", &file.FileStoreRawWriterConfig{})
	processor.RegisteredProcessors.Register("file_raw_reader", &file.FileStoreRawReaderConfig{})

	conf, err := processor.ParseConfigFile("test", "testdata/config-test.ser.yml")
	if err != nil {
		t.Error("error parsing file", "err", err)
		return
	}
	log.Printf("%+v", *conf)

	if len(conf.Streams) < 1 {
		t.Error("wrong number of flows", len(conf.Streams))
		return
	}

	out, err := yaml.Marshal(conf)
	log.Println("OUT:", err, string(out))
	// t.Error("===Success===")
}
