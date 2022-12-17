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

func GetTotalDocs(ctx context.Context, scope *gocb.Scope, collection string) (error, int64) {
	totalDocs := TotalDocs{}

	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM auth._default.%s", collection)
	queryRes, err := scope.Query(countQuery, &gocb.QueryOptions{
		Context: ctx,
	})

	queryRes.One(&totalDocs)

	if err != nil {
		return err, totalDocs.Total
	}
	return nil, totalDocs.Total
}

type TotalDocs struct {
	Total int64
}
