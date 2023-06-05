package processor

import (
	"bytes"
	"io/ioutil"

	log "axway.com/qlt-router/src/log"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// processors map[string]ProcessorConf
	Streams []*Flow `yaml:""`
}

func ParseConfigFile(ctx string, filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Errorc(ctx, "Failed to open ", "filename", filename, "err", err)
		return nil, err
	}
	log.Debugc(ctx, "config file raw", "filename", filename, "data", string(data))
	return ParseConfigRawData(data)
}

func ParseConfigRawData(data []byte) (*Config, error) {
	var config Config

	r := yaml.NewDecoder(bytes.NewReader(data))
	r.KnownFields(true)
	err := r.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
