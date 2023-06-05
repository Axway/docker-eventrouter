package tools

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrFieldNotFound      = errors.New("field not found")
	ErrUnexpectedYamlNode = errors.New("unexpected yaml node")
)

// Ensure that the yaml obj only contains obj struct members
func YamlParseVerify(name string, obj interface{}, y *yaml.Node) error {
	if y.Kind == 0 {
		return nil
	}
	if y.Kind != yaml.MappingNode {
		return fmt.Errorf("%w : %s", ErrUnexpectedYamlNode, "expecting map node for '"+name+"' ")
	}

	for c := 0; c < len(y.Content); c += 2 {
		v := y.Content[c]
		found := false
		typ := reflect.TypeOf(obj).Elem()
		var fields []string
		// fmt.Println(name, "typ", typ)
		for i := 0; i < typ.NumField(); i++ {
			up := v.Value
			tag := typ.Field(i).Tag.Get("yaml")
			name := typ.Field(i).Name
			name = strings.ToLower(string(name[0])) + name[1:] // Ack to ensure first letter as lowercase by default
			// fmt.Println(name, tag, v.Value, up)
			if tag == up {
				found = true
				break
			} else if tag == "" && name == up {
				found = true
				v.Value = strings.ToLower(name) // No tag yaml.v3 only find lowercase names !!!
				break
			}

			// For error case only
			key := tag
			if tag == "" && name == up {
				key = name
			}
			fields = append(fields, key)
		}
		if !found {
			return fmt.Errorf("%w : %s", ErrFieldNotFound, "'"+v.Value+"' for "+name+" expecting: "+strings.Join(fields, ","))
		}
	}
	err := y.Decode(obj)

	return err
}
