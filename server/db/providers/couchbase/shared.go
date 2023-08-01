package couchbase

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/couchbase/gocb/v2"
)

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
		if value == nil {
			updateFields += fmt.Sprintf("%s=$%s,", key, key)
			params[key] = "null"
			continue
		}
		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("%s = $%s, ", key, key)
			params[key] = value.(string)

		} else {
			updateFields += fmt.Sprintf("%s = $%s, ", key, key)
			params[key] = value
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	return updateFields, params
}

func (p *provider) GetTotalDocs(ctx context.Context, collection string) (int64, error) {
	totalDocs := TotalDocs{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s.%s", p.scopeName, collection)
	queryRes, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context: ctx,
	})
	queryRes.One(&totalDocs)
	if err != nil {
		return totalDocs.Total, err
	}
	return totalDocs.Total, nil
}

type TotalDocs struct {
	Total int64
}
