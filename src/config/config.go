package config

import (
	"time"

	log "github.com/sirupsen/logrus"

	bytesize "github.com/inhies/go-bytesize"
)

type ConfigValue struct {
	Value       interface{}
	Name        string
	Description string
}

// No lock protection : supposed to be static (at initialization)
var configValues = make(map[string]ConfigValue)

func DeclareInt(name string, defaultValue int, description string) int {
	configValues[name] = ConfigValue{defaultValue, name, description}
	return defaultValue
}

func DeclareString(name string, defaultValue string, description string) string {
	configValues[name] = ConfigValue{defaultValue, name, description}
	return defaultValue
}

func DeclareBool(name string, defaultValue bool, description string) bool {
	configValues[name] = ConfigValue{defaultValue, name, description}
	return defaultValue
}

func DeclareFloat(name string, defaultValue float64, description string) float64 {
	configValues[name] = ConfigValue{defaultValue, name, description}
	return defaultValue
}

func DeclareDuration(name string, defaultValue string, description string) time.Duration {
	value, err := time.ParseDuration(defaultValue)
	if err != nil {
		panic("bad duration string: '" + defaultValue + "' " + err.Error())
	}
	configValues[name] = ConfigValue{value, name, description}
	return value
}

func DeclareSize(name string, defaultValue string, description string) int64 {
	value, err := bytesize.Parse(defaultValue)
	if err != nil {
		panic("bad size string: '" + defaultValue + "' " + err.Error())
	}
	configValues[name] = ConfigValue{value, name, description}
	return int64(value)
}

func Print() {
	for k, v := range configValues {
		log.Println("", k, "=", v.Value, v.Description)
	}
}
