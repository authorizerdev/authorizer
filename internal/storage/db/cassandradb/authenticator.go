package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
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
	convertMapValues(authenticatorsMap)

	fields := "("
	placeholders := "("
	var insertValues []interface{}
	for key, value := range authenticatorsMap {
		if value != nil {
			if key == "_id" {
				fields += "id,"
			} else {
				fields += key + ","
			}
			placeholders += "?,"
			insertValues = append(insertValues, value)
		}
	}

	fields = fields[:len(fields)-1] + ")"
	placeholders = placeholders[:len(placeholders)-1] + ")"

	query := fmt.Sprintf("INSERT INTO %s %s VALUES %s IF NOT EXISTS", KeySpace+"."+schemas.Collections.Authenticators, fields, placeholders)
	err = p.db.Query(query, insertValues...).Exec()
	if err != nil {
		return nil, err
	}

	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
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
	convertMapValues(authenticatorsMap)

	updateFields := ""
	var updateValues []interface{}
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

		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	updateValues = append(updateValues, authenticators.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Authenticators, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}

	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators schemas.Authenticator
	query := fmt.Sprintf("SELECT id, user_id, method, secret, recovery_codes, verified_at, created_at, updated_at FROM %s WHERE user_id = ? AND method = ? LIMIT 1 ALLOW FILTERING", KeySpace+"."+schemas.Collections.Authenticators)
	err := p.db.Query(query, userId, authenticatorType).Consistency(gocql.One).Scan(&authenticators.ID, &authenticators.UserID, &authenticators.Method, &authenticators.Secret, &authenticators.RecoveryCodes, &authenticators.VerifiedAt, &authenticators.CreatedAt, &authenticators.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &authenticators, nil
}
