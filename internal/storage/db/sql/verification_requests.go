package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerificationRequest saves a verification request, replacing any pending
// request for the same (email, identifier), and returns the stored row.
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}
	verificationRequest.Key = verificationRequest.ID
	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()
	result := p.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}, {Name: "identifier"}},
		DoUpdates: clause.AssignmentColumns([]string{"token", "expires_at", "nonce", "redirect_uri", "updated_at"}),
	}).Create(&verificationRequest)
	if result.Error != nil {
		return verificationRequest, result.Error
	}
	// On conflict the existing row keeps its own id and created_at, so the
	// struct above no longer describes what is stored. Re-read it (into a
	// fresh value: First on a populated struct also filters by its id).
	var stored schemas.VerificationRequest
	if err := p.db.WithContext(ctx).Where("email = ? AND identifier = ?", verificationRequest.Email, verificationRequest.Identifier).First(&stored).Error; err != nil {
		return nil, err
	}
	*verificationRequest = stored
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest
	result := p.db.WithContext(ctx).Where("token = ?", token).First(&verificationRequest)
	if result.Error != nil {
		return verificationRequest, result.Error
	}
	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest
	result := p.db.WithContext(ctx).Where("email = ? AND identifier = ?", email, identifier).First(&verificationRequest)
	if result.Error != nil {
		return verificationRequest, result.Error
	}
	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	var verificationRequests []*schemas.VerificationRequest
	result := p.db.WithContext(ctx).Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&verificationRequests)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.WithContext(ctx).Model(&schemas.VerificationRequest{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total
	return verificationRequests, paginationClone, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	result := p.db.WithContext(ctx).Delete(&verificationRequest)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
