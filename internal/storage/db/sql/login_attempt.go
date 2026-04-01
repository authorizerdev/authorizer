package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

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
	res := p.db.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&loginAttempt)
	return res.Error
}

// CountFailedLoginAttemptsSince counts failed login attempts for an email since the given Unix timestamp
func (p *provider) CountFailedLoginAttemptsSince(ctx context.Context, email string, since int64) (int64, error) {
	var count int64
	res := p.db.Model(&schemas.LoginAttempt{}).
		Where("email = ? AND successful = ? AND attempted_at >= ?", email, false, since).
		Count(&count)
	if res.Error != nil {
		return 0, res.Error
	}
	return count, nil
}

// DeleteLoginAttemptsBefore removes all login attempts older than the given Unix timestamp
func (p *provider) DeleteLoginAttemptsBefore(ctx context.Context, before int64) error {
	res := p.db.Where("attempted_at < ?", before).Delete(&schemas.LoginAttempt{})
	return res.Error
}
