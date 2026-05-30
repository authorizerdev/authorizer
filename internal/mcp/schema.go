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
	return schemaForMessageWithVisited(md, map[protoreflect.FullName]struct{}{})
}

// schemaForMessageWithVisited recurses into nested message fields while
// guarding against cycles. The descriptor full-name is the visit key —
// well-known types like google.protobuf.Value reference themselves via
// repeated-Value lists, which would stack-overflow without this.
//
// On a re-visit we emit an opaque `object` rather than the full schema,
// which is the most honest thing to tell an MCP host about a self-recursive
// type (it can pass any JSON object; the server validates at the proto
// layer via protovalidate).
func schemaForMessageWithVisited(md protoreflect.MessageDescriptor, visited map[protoreflect.FullName]struct{}) jsonSchema {
	if _, seen := visited[md.FullName()]; seen {
		return jsonSchema{Type: "object"}
	}
	visited[md.FullName()] = struct{}{}
	defer delete(visited, md.FullName())

	root := jsonSchema{
		Type:       "object",
		Properties: map[string]jsonSchema{},
	}
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		root.Properties[string(f.Name())] = schemaForField(f, visited)
	}
	return root
}

func schemaForField(f protoreflect.FieldDescriptor, visited map[protoreflect.FullName]struct{}) jsonSchema {
	// repeated → JSON array
	if f.IsList() {
		item := schemaForKind(f, visited)
		return jsonSchema{Type: "array", Items: &item}
	}
	if f.IsMap() {
		return jsonSchema{Type: "object"}
	}
	return schemaForKind(f, visited)
}

func schemaForKind(f protoreflect.FieldDescriptor, visited map[protoreflect.FullName]struct{}) jsonSchema {
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
		return schemaForMessageWithVisited(f.Message(), visited)
	default:
		return jsonSchema{Type: "string"}
	}
}
