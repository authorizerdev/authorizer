package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

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
	authenticators.Key = authenticators.ID
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	// Target the (user_id, method) unique index so a concurrent enrollment that
	// slips past the check-then-insert race above upserts the existing row
	// instead of creating a duplicate. Without this, GetAuthenticatorDetailsByUserId's
	// First() would return an arbitrary row and cause intermittent MFA failures.
	res := p.db.Clauses(
		clause.OnConflict{
			UpdateAll: true,
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "method"}},
		}).Create(&authenticators)
	if res.Error != nil {
		return nil, res.Error
	}
	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&authenticators)
	if result.Error != nil {
		return authenticators, result.Error
	}
	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators schemas.Authenticator
	result := p.db.Where("user_id = ?", userId).Where("method = ?", authenticatorType).First(&authenticators)
	if result.Error != nil {
		return nil, result.Error
	}
	return &authenticators, nil
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// Used by admin MFA reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	return p.db.Where("user_id = ?", userID).Delete(&schemas.Authenticator{}).Error
}
