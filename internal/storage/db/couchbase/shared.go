package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/couchbase/gocb/v2"
)

// structToDocument converts a schema struct into a map for Couchbase persistence.
//
// Couchbase is the only storage provider that (de)serializes whole schema structs
// through encoding/json (the gocb default JSON transcoder). encoding/json honors
// `json:"-"`, which is set on secret fields such as User.Password purely to keep them
// out of API/GraphQL JSON responses. As a side effect those fields were silently
// dropped from the persisted document. structToDocument reproduces the exact JSON
// document shape for every normally-tagged field, then re-adds any `json:"-"` field
// under its `bson` tag key so it is actually persisted. All Insert/Upsert call sites
// pass their document through this helper.
//
// ponytail: paired with decodeDocument. Any struct carrying a `json:"-"` field MUST use
// structToDocument on its write path and decodeDocument on its read path, or the field is
// silently dropped. The `json:"-"` secret fields in the schemas are User.Password,
// Client.ClientSecret and TrustedIssuer.SSOClientSecretEnc — every one of their
// Insert/Upsert and read paths goes through this helper pair.
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
		fv := rv.Field(i)
		if !fv.CanInterface() {
			continue
		}
		key := persistKey(f)
		if key == "" {
			continue
		}
		doc[key] = fv.Interface()
	}
	return doc, nil
}

// decodeDocument unmarshals a Couchbase row/document into dest, then populates any
// `json:"-"` field from its `bson` tag key (encoding/json ignores such fields on
// unmarshal too). dest must be a non-nil pointer to a struct. See structToDocument.
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

// persistKey returns the document key a `json:"-"` field is stored under. Couchbase
// document keys mirror the bson/json tag names, so it uses the field's `bson` tag,
// falling back to the lowercased field name.
func persistKey(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("bson"), ",")[0]
	if name == "" || name == "-" {
		return strings.ToLower(f.Name)
	}
	return name
}

// GetSetFields to get set fields
func GetSetFields(webhookMap map[string]interface{}) (string, map[string]interface{}) {
	params := make(map[string]interface{}, 1)
	updateFields := ""
	for key, value := range webhookMap {
		if key == "_id" {
			continue
		}
		if key == "_key" {
			continue
		}
		// Backtick the column identifier so N1QL reserved words (e.g. `roles`)
		// are always legal in the SET clause. The bind-param name ($key) is not
		// an identifier and needs no quoting.
		if value == nil {
			updateFields += fmt.Sprintf("`%s`=$%s,", key, key)
			// Bind an actual nil so the gocb N1QL driver serializes it to JSON
			// null (a real N1QL NULL). Binding the string "null" here would
			// persist the 4-char literal instead of clearing the field.
			params[key] = nil
			continue
		}
		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("`%s` = $%s, ", key, key)
			params[key] = value.(string)

		} else {
			updateFields += fmt.Sprintf("`%s` = $%s, ", key, key)
			params[key] = value
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	return updateFields, params
}

// GetTotalDocs to get total documents in a collection
func (p *provider) GetTotalDocs(ctx context.Context, collection string) (int64, error) {
	totalDocs := TotalDocs{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s.%s", p.scopeName, collection)
	queryRes, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context: ctx,
	})
	_ = queryRes.One(&totalDocs)
	if err != nil {
		return 0, err
	}
	return totalDocs.Total, nil
}

type TotalDocs struct {
	Total int64
}
