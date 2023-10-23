package couchbase

import (
	"context"
	"fmt"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
	"time"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}

	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Authenticators).Insert(authenticators.ID, authenticators, &insertOpt)
	if err != nil {
		return &authenticators, err
	}
	return &authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	authenticators.UpdatedAt = time.Now().Unix()
	upsertOpt := gocb.UpsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Authenticators).Upsert(authenticators.ID, authenticators, &upsertOpt)
	if err != nil {
		return &authenticators, err
	}
	return &authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticators, error) {
	var authenticators *models.Authenticators
	query := fmt.Sprintf("SELECT id, user_id, method, secret, recovery_code, verified_at, created_at, updated_at FROM %s.%s WHERE user_id = $1 AND method = $2 LIMIT 1", p.scopeName, models.Collections.Authenticators)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{userId, authenticatorType},
	})
	if err != nil {
		return authenticators, err
	}
	err = q.One(&authenticators)
	if err != nil {
		return authenticators, err
	}
	return authenticators, nil
}
