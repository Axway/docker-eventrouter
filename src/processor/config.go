package processor

import (
	"bytes"
	"os"

	log "axway.com/qlt-router/src/log"
	"github.com/a8m/envsubst"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// processors map[string]ProcessorConf
	Streams     []*Flow `yaml:""`
	Instance_id string  `yaml:"instance_id"`
}

func ParseConfigFile(ctx, filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Errorc(ctx, "Failed to open ", "filename", filename, "err", err)
		return nil, err
	}
	log.Debugc(ctx, "config file raw", "filename", filename, "data", string(data))
	return ParseConfigRawData(data)
}

func ParseConfigRawData(data []byte) (*Config, error) {
	var config Config

	buf, err := envsubst.Bytes(data)
	if err != nil {
		return nil, err
	}

	r := yaml.NewDecoder(bytes.NewReader(buf))
	r.KnownFields(true)
	err = r.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
