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
	ErrFieldRequired      = errors.New("required field")
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
			required := typ.Field(i).Tag.Get("required") == "true"
			name := typ.Field(i).Name
			name = strings.ToLower(string(name[0])) + name[1:] // Ack to ensure first letter as lowercase by default
			value := y.Content[c+1].Value
			// fmt.Println(name, tag, v.Value, up)
			if tag == up {
				found = true
			} else if tag == "" && name == up {
				found = true
				v.Value = strings.ToLower(name) // No tag yaml.v3 only find lowercase names !!!
			}
			if found == true {
				if required && value == "" {
					return fmt.Errorf("%w : '%s' (line=%d)", ErrFieldRequired, v.Value, v.Line)
				}
				break
			}

			// For error case only
			key := tag
			if tag == "" {
				key = name
			}
			fields = append(fields, key)
		}

		if !found {
			return fmt.Errorf("%w : '%s' (line=%d) %s", ErrFieldNotFound, v.Value, v.Line, " for "+name+" expecting: "+strings.Join(fields, ","))
		}
	}

	// check required fields
	{
		typ := reflect.TypeOf(obj).Elem()
		for i := 0; i < typ.NumField(); i++ {
			required := typ.Field(i).Tag.Get("required") == "true"
			if required {
				found := false
				name := typ.Field(i).Name
				name = strings.ToLower(string(name[0])) + name[1:]
				tag := typ.Field(i).Tag.Get("yaml")

				for c := 0; c < len(y.Content); c += 2 {
					up := strings.ToLower(y.Content[c].Value)
					if tag == up || (tag == "" && strings.ToLower(name) == strings.ToLower(up)) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("%w : '%s' (line=%d)", ErrFieldRequired, name, y.Line)
				}
			}
		}
	}
	err := y.Decode(obj)

	return err
}
