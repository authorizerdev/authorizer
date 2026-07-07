package arangodb

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	arangoDriver "github.com/arangodb/go-driver"
)

// ArangoDB's go-driver (de)serializes whole schema structs through
// encoding/json (the default HTTP JSON connection). encoding/json honors
// `json:"-"`, which is set on secret fields such as User.Password and
// ServiceAccount.ClientSecret purely to keep them out of API/GraphQL JSON
// responses. As a side effect those fields were silently dropped from the
// persisted document on write AND from the struct on read. The helpers below
// reproduce the exact JSON document shape for every normally-tagged field, then
// re-add / backfill any `json:"-"` field under its `bson` tag key so it is
// actually persisted and loaded. Mirrors the couchbase provider's shared.go.
//
// ponytail: structToDocument and readDocument are paired. If a schema gains a
// new `json:"-"` field, its write path MUST go through structToDocument and its
// read path MUST go through readDocument, or the field will silently load/store
// empty. Today only User.Password and ServiceAccount.ClientSecret are affected.

// structToDocument converts a schema struct into a map for ArangoDB writes,
// re-adding any `json:"-"` field under its `bson` tag key (encoding/json drops
// such fields on marshal). All Create/Update call sites for structs carrying a
// `json:"-"` field pass their document through this helper.
func structToDocument(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	doc := map[string]interface{}{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return doc, nil
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Tag.Get("json") != "-" {
			continue
		}
		key := persistKey(f)
		if key == "" {
			continue
		}
		doc[key] = rv.Field(i).Interface()
	}
	return doc, nil
}

// readDocument reads the next document from the cursor as raw JSON, unmarshals
// it into dest, then populates any `json:"-"` field from its `bson` tag key
// (encoding/json ignores such fields on unmarshal too). The driver decodes a
// *json.RawMessage target into the untouched document body, so every normal
// field decodes exactly as a direct ReadDocument into the struct would.
// dest must be a non-nil pointer to a struct. The returned DocumentMeta and
// error mirror cursor.ReadDocument (including NoMoreDocuments).
func readDocument(ctx context.Context, cursor arangoDriver.Cursor, dest interface{}) (arangoDriver.DocumentMeta, error) {
	var raw json.RawMessage
	meta, err := cursor.ReadDocument(ctx, &raw)
	if err != nil {
		return meta, err
	}
	return meta, decodeDocument(raw, dest)
}

// decodeDocument unmarshals a raw JSON document into dest, then backfills any
// `json:"-"` field from its `bson` tag key. See structToDocument.
func decodeDocument(data []byte, dest interface{}) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return err
	}
	rv := reflect.Indirect(reflect.ValueOf(dest))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var rawFields map[string]json.RawMessage
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Tag.Get("json") != "-" {
			continue
		}
		key := persistKey(f)
		if key == "" {
			continue
		}
		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}
		if rawFields == nil {
			rawFields = map[string]json.RawMessage{}
			if err := json.Unmarshal(data, &rawFields); err != nil {
				return err
			}
		}
		raw, ok := rawFields[key]
		if !ok {
			continue
		}
		nv := reflect.New(f.Type)
		if err := json.Unmarshal(raw, nv.Interface()); err != nil {
			return err
		}
		fv.Set(nv.Elem())
	}
	return nil
}

// persistKey returns the document key a `json:"-"` field is stored under.
// ArangoDB document keys mirror the bson/json tag names, so it uses the field's
// `bson` tag, falling back to the lowercased field name.
func persistKey(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("bson"), ",")[0]
	if name == "" || name == "-" {
		return strings.ToLower(f.Name)
	}
	return name
}
