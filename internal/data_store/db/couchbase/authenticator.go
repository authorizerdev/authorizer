package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}

	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.Key = authenticators.ID
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Authenticators).Insert(authenticators.ID, authenticators, &insertOpt)
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
	authenticator := map[string]interface{}{}
	err = decoder.Decode(&authenticator)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(authenticator)
	query := fmt.Sprintf("UPDATE %s.%s SET %s WHERE _id = '%s'", p.scopeName, schemas.Collections.Authenticators, updateFields, authenticators.ID)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators *schemas.Authenticator
	query := fmt.Sprintf("SELECT _id, user_id, method, secret, recovery_code, verified_at, created_at, updated_at FROM %s.%s WHERE user_id = $1 AND method = $2 LIMIT 1", p.scopeName, schemas.Collections.Authenticators)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{userId, authenticatorType},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&authenticators)
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}
