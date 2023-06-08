package tools

import (
	"errors"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYamlParseVerify(t *testing.T) {
	t.Parallel()
	type TestZou struct {
		FieldR1 string
		FieldR2 string `yaml:"fieldR2"`
		FieldR3 string `yaml:"FieldR3"`
	}

	y1 := `
fieldR1: zouzou
fieldR2: zaza
FieldR3: z1	
`
	p1 := TestZou{}
	err := yaml.Unmarshal([]byte(y1), &p1)
	if err != nil {
		t.Error("classic yaml unmarshall failed", err)
		return
	}

	if p1.FieldR1 != "" {
		t.Error("classic yaml unmarshall: fieldR1 should be empty")
	}

	if p1.FieldR2 != "zaza" {
		t.Error("classic yaml unmarshall: fieldR2 should be zaza")
	}

	if p1.FieldR3 != "z1" {
		t.Error("classic yaml unmarshall: fieldR3 should be z1")
	}

	var p2 yaml.Node
	err = yaml.Unmarshal([]byte(y1), &p2)
	if err != nil {
		t.Error("yaml.Node yaml unmarshall failed", err)
		return
	}

	p3 := TestZou{}
	err = YamlParseVerify("test", &p3, p2.Content[0])
	if err != nil {
		t.Error("yaml.Node yamlParseVerify failed", err)
		return
	}
	if p3.FieldR1 != "zouzou" {
		t.Error("yamlParseVerify: fieldR1 should be zouzou")
	}

	if p3.FieldR2 != "zaza" {
		t.Error("yamlParseVerify: fieldR2 should be zaza")
	}

	if p3.FieldR3 != "z1" {
		t.Error("yamlParseVerify: fieldR3 should be z1")
	}

	y2 := `
FieldR1: zouzou
fieldR2: zaza
FieldR3: z1	
`
	var p4 yaml.Node
	err = yaml.Unmarshal([]byte(y2), &p4)
	if err != nil {
		t.Error("yaml.Node yaml unmarshall failed", err)
		return
	}
	err = YamlParseVerify("test", &TestZou{}, p4.Content[0])
	if err == nil {
		t.Error("yamlParseVerify: expecting error")
	}
	if !errors.Is(err, ErrFieldNotFound) {
		t.Error("yamlParseVerify: expecting field not found error")
	}
	// t.Error("=== SUCCESSFUL ===")
}
