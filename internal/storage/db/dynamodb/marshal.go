package dynamodb

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func dynamoFieldNameFromTag(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return ""
	}
	return name
}

func dynamoOmitEmpty(tag string) bool {
	return strings.Contains(tag, "omitempty")
}

func marshalStruct(v interface{}) (map[string]types.AttributeValue, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("nil value")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or pointer to struct")
	}
	rt := rv.Type()
	out := make(map[string]types.AttributeValue)
	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := sf.Tag.Get("dynamo")
		name := dynamoFieldNameFromTag(tag)
		if name == "" {
			continue
		}
		fv := rv.Field(i)
		if dynamoOmitEmpty(tag) && isEmptyValue(fv) {
			continue
		}
		if fv.Kind() == reflect.Ptr && fv.IsNil() {
			continue
		}
		av, err := attributevalue.Marshal(fv.Interface())
		if err != nil {
			return nil, err
		}
		out[name] = av
	}
	return out, nil
}

func isEmptyValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

func marshalMapStringInterface(m map[string]interface{}) (map[string]types.AttributeValue, error) {
	out := make(map[string]types.AttributeValue, len(m))
	for k, v := range m {
		av, err := attributevalue.Marshal(v)
		if err != nil {
			return nil, err
		}
		out[k] = av
	}
	return out, nil
}

func unmarshalItem(av map[string]types.AttributeValue, out interface{}) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("out must be non-nil pointer")
	}
	ev := rv.Elem()
	if ev.Kind() != reflect.Struct {
		return fmt.Errorf("out must be pointer to struct")
	}
	et := ev.Type()
	for i := 0; i < ev.NumField(); i++ {
		sf := et.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := sf.Tag.Get("dynamo")
		name := dynamoFieldNameFromTag(tag)
		if name == "" {
			continue
		}
		dv, ok := av[name]
		if !ok {
			continue
		}
		field := ev.Field(i)
		if !field.CanSet() {
			continue
		}
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			if err := attributevalue.Unmarshal(dv, field.Interface()); err != nil {
				return err
			}
			continue
		}
		if err := attributevalue.Unmarshal(dv, field.Addr().Interface()); err != nil {
			return err
		}
	}
	return nil
}
