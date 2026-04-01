package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddLoginAttempt records a login attempt in the database
func (p *provider) AddLoginAttempt(ctx context.Context, loginAttempt *schemas.LoginAttempt) error {
	if loginAttempt.ID == "" {
		loginAttempt.ID = uuid.New().String()
	}
	loginAttempt.Key = loginAttempt.ID
	loginAttempt.CreatedAt = time.Now().Unix()
	if loginAttempt.AttemptedAt == 0 {
		loginAttempt.AttemptedAt = loginAttempt.CreatedAt
	}
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.LoginAttempt).Insert(loginAttempt.ID, loginAttempt, &insertOpt)
	return err
}

// CountFailedLoginAttemptsSince counts failed login attempts for an email since the given Unix timestamp
func (p *provider) CountFailedLoginAttemptsSince(ctx context.Context, email string, since int64) (int64, error) {
	query := fmt.Sprintf(
		`SELECT COUNT(*) as count FROM %s.%s WHERE email = $email AND successful = false AND attempted_at >= $since`,
		p.scopeName, schemas.Collections.LoginAttempt,
	)
	params := map[string]interface{}{
		"email": email,
		"since": since,
	}
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return 0, err
	}
	var result struct {
		Count int64 `json:"count"`
	}
	if queryResult.Next() {
		if err := queryResult.Row(&result); err != nil {
			return 0, err
		}
	}
	if err := queryResult.Err(); err != nil {
		return 0, err
	}
	return result.Count, nil
}

// DeleteLoginAttemptsBefore removes all login attempts older than the given Unix timestamp
func (p *provider) DeleteLoginAttemptsBefore(ctx context.Context, before int64) error {
	query := fmt.Sprintf(
		`DELETE FROM %s.%s WHERE attempted_at < $before`,
		p.scopeName, schemas.Collections.LoginAttempt,
	)
	params := map[string]interface{}{
		"before": before,
	}
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	return err
}
