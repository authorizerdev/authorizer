package arangodb

import (
	"context"
	"fmt"
	"time"

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
	loginAttemptCollection, _ := p.db.Collection(ctx, schemas.Collections.LoginAttempt)
	_, err := loginAttemptCollection.CreateDocument(ctx, loginAttempt)
	return err
}

// CountFailedLoginAttemptsSince counts failed login attempts for an email since the given Unix timestamp
func (p *provider) CountFailedLoginAttemptsSince(ctx context.Context, email string, since int64) (int64, error) {
	query := fmt.Sprintf(
		"RETURN LENGTH(FOR d IN %s FILTER d.email == @email AND d.successful == false AND d.attempted_at >= @since RETURN 1)",
		schemas.Collections.LoginAttempt,
	)
	bindVariables := map[string]interface{}{
		"email": email,
		"since": since,
	}
	cursor, err := p.db.Query(ctx, query, bindVariables)
	if err != nil {
		return 0, err
	}
	defer cursor.Close()
	var count int64
	_, err = cursor.ReadDocument(ctx, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteLoginAttemptsBefore removes all login attempts older than the given Unix timestamp
func (p *provider) DeleteLoginAttemptsBefore(ctx context.Context, before int64) error {
	query := fmt.Sprintf(
		"FOR d IN %s FILTER d.attempted_at < @before REMOVE d IN %s",
		schemas.Collections.LoginAttempt, schemas.Collections.LoginAttempt,
	)
	bindVariables := map[string]interface{}{
		"before": before,
	}
	cursor, err := p.db.Query(ctx, query, bindVariables)
	if err != nil {
		return err
	}
	cursor.Close()
	return nil
}
