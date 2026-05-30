package mcp

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// jsonSchema is a tiny JSON-Schema subset — enough to describe the input of
// a typical Authorizer RPC. We don't bring in a full schema library because
// the MCP host only needs property names, types, and descriptions for tool
// discovery.
type jsonSchema struct {
	Type        string                `json:"type"`
	Properties  map[string]jsonSchema `json:"properties,omitempty"`
	Items       *jsonSchema           `json:"items,omitempty"`
	Description string                `json:"description,omitempty"`
	Required    []string              `json:"required,omitempty"`
}

// schemaForMessage derives a JSON Schema (object form) for a proto message
// descriptor. Field naming uses the proto field name (snake_case), matching
// the gateway's UseProtoNames=true configuration.
func schemaForMessage(md protoreflect.MessageDescriptor) jsonSchema {
	root := jsonSchema{
		Type:       "object",
		Properties: map[string]jsonSchema{},
	}
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		root.Properties[string(f.Name())] = schemaForField(f)
	}
	return root
}

func schemaForField(f protoreflect.FieldDescriptor) jsonSchema {
	// repeated → JSON array
	if f.IsList() {
		item := schemaForKind(f)
		return jsonSchema{Type: "array", Items: &item}
	}
	if f.IsMap() {
		return jsonSchema{Type: "object"}
	}
	return schemaForKind(f)
}

func schemaForKind(f protoreflect.FieldDescriptor) jsonSchema {
	switch f.Kind() {
	case protoreflect.BoolKind:
		return jsonSchema{Type: "boolean"}
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return jsonSchema{Type: "integer"}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return jsonSchema{Type: "number"}
	case protoreflect.StringKind, protoreflect.BytesKind:
		return jsonSchema{Type: "string"}
	case protoreflect.EnumKind:
		return jsonSchema{Type: "string"}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return schemaForMessage(f.Message())
	default:
		return jsonSchema{Type: "string"}
	}
}
