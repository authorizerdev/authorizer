package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"github.com/authorizerdev/authorizer/internal/models/schemas"
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
	res := p.db.Clauses(
		clause.OnConflict{
			UpdateAll: true,
			Columns:   []clause.Column{{Name: "id"}},
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
