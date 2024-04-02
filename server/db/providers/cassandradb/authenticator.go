package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/server/db/models"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *models.Authenticator) (*models.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}

	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}

	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()

	bytes, err := json.Marshal(authenticators)
	if err != nil {
		return nil, err
	}

	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	authenticatorsMap := map[string]interface{}{}
	err = decoder.Decode(&authenticatorsMap)
	if err != nil {
		return nil, err
	}

	fields := "("
	values := "("
	for key, value := range authenticatorsMap {
		if value != nil {
			if key == "_id" {
				fields += "id,"
			} else {
				fields += key + ","
			}

			valueType := reflect.TypeOf(value)
			if valueType.Name() == "string" {
				values += fmt.Sprintf("'%s',", value.(string))
			} else {
				values += fmt.Sprintf("%v,", value)
			}
		}
	}

	fields = fields[:len(fields)-1] + ")"
	values = values[:len(values)-1] + ")"

	query := fmt.Sprintf("INSERT INTO %s %s VALUES %s IF NOT EXISTS", KeySpace+"."+models.Collections.Authenticators, fields, values)
	err = p.db.Query(query).Exec()
	if err != nil {
		return nil, err
	}

	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *models.Authenticator) (*models.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()

	bytes, err := json.Marshal(authenticators)
	if err != nil {
		return nil, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	authenticatorsMap := map[string]interface{}{}
	err = decoder.Decode(&authenticatorsMap)
	if err != nil {
		return nil, err
	}

	updateFields := ""
	for key, value := range authenticatorsMap {
		if key == "_id" {
			continue
		}

		if key == "_key" {
			continue
		}

		if value == nil {
			updateFields += fmt.Sprintf("%s = null, ", key)
			continue
		}

		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("%s = '%s', ", key, value.(string))
		} else {
			updateFields += fmt.Sprintf("%s = %v, ", key, value)
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = '%s'", KeySpace+"."+models.Collections.Authenticators, updateFields, authenticators.ID)
	err = p.db.Query(query).Exec()
	if err != nil {
		return nil, err
	}

	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticator, error) {
	var authenticators models.Authenticator
	query := fmt.Sprintf("SELECT id, user_id, method, secret, recovery_codes, verified_at, created_at, updated_at FROM %s WHERE user_id = '%s' AND method = '%s' LIMIT 1 ALLOW FILTERING", KeySpace+"."+models.Collections.Authenticators, userId, authenticatorType)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&authenticators.ID, &authenticators.UserID, &authenticators.Method, &authenticators.Secret, &authenticators.RecoveryCodes, &authenticators.VerifiedAt, &authenticators.CreatedAt, &authenticators.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &authenticators, nil
}
