package dynamodb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddLoginAttempt records a login attempt in the database
func (p *provider) AddLoginAttempt(ctx context.Context, loginAttempt *schemas.LoginAttempt) error {
	collection := p.db.Table(schemas.Collections.LoginAttempt)
	if loginAttempt.ID == "" {
		loginAttempt.ID = uuid.New().String()
	}
	loginAttempt.Key = loginAttempt.ID
	loginAttempt.CreatedAt = time.Now().Unix()
	if loginAttempt.AttemptedAt == 0 {
		loginAttempt.AttemptedAt = loginAttempt.CreatedAt
	}
	return collection.Put(loginAttempt).RunWithContext(ctx)
}

// CountFailedLoginAttemptsSince counts failed login attempts for an email since the given Unix timestamp
func (p *provider) CountFailedLoginAttemptsSince(ctx context.Context, email string, since int64) (int64, error) {
	collection := p.db.Table(schemas.Collections.LoginAttempt)
	var loginAttempts []*schemas.LoginAttempt
	err := collection.Scan().
		Filter("'email' = ? AND 'successful' = ? AND 'attempted_at' >= ?", email, false, since).
		AllWithContext(ctx, &loginAttempts)
	if err != nil {
		return 0, err
	}
	return int64(len(loginAttempts)), nil
}

// DeleteLoginAttemptsBefore removes all login attempts older than the given Unix timestamp
func (p *provider) DeleteLoginAttemptsBefore(ctx context.Context, before int64) error {
	collection := p.db.Table(schemas.Collections.LoginAttempt)
	var loginAttempts []*schemas.LoginAttempt
	err := collection.Scan().
		Filter("'attempted_at' < ?", before).
		AllWithContext(ctx, &loginAttempts)
	if err != nil {
		return err
	}
	for _, la := range loginAttempts {
		if err := collection.Delete("id", la.ID).RunWithContext(ctx); err != nil {
			return err
		}
	}
	return nil
}
