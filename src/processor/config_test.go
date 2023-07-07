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

	c1 := `streams:
- name: flow1
  description: "flow1 description"
  flow: 
      - name: "file_raw_reader"
        conf: 
           filename: "zoufile"
- name: flow2
  flow:
        - name: "file_raw_writer"
          conf:
            filename: "pathfile"
- name: flow3
`

	processor.RegisteredProcessors.Register("file_raw_writer", &file.FileStoreRawWriterConfig{})
	processor.RegisteredProcessors.Register("file_raw_reader", &file.FileStoreRawReaderConfig{})

	c := make(map[string]interface{})
	yaml.Unmarshal([]byte(c1), &c)
	log.Printf("%+v", c)

	conf, err := processor.ParseConfigRawData([]byte(c1))
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
