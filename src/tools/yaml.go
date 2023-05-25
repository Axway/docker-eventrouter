package tools

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

func YamlParseVerify(name string, obj interface{}, y *yaml.Node) error {
	if y.Kind == 0 {
		return nil
	}
	if y.Kind != yaml.MappingNode {
		return errors.New("yaml: unexpected non map '" + name + "' ")
	}

	for c := 0; c < len(y.Content); c += 2 {
		v := y.Content[c]
		found := false
		typ := reflect.TypeOf(obj).Elem()
		var fields []string
		for i := 0; i < typ.NumField(); i++ {
			up := v.Value // strings.ToUpper(string(v.Value[0])) + v.Value[1:]
			tag := typ.Field(i).Tag.Get("yaml")
			name := typ.Field(i).Name
			name = strings.ToLower(string(name[0])) + name[1:] // Ack to ensure first letter as lowercase by default
			fmt.Println(name, tag, v.Value, up)
			key := tag
			if key == "" {
				key = name
			}
			if key == up {
				found = true
				v.Value = strings.ToLower(name)
				break
			}
			fields = append(fields, key)
		}
		if !found {
			return errors.New("unknown field '" + v.Value + "' for " + name + " expecting: " + strings.Join(fields, ","))
		}
	}
	err := y.Decode(obj)

	return err
}
