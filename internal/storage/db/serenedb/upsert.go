package serenedb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddAuthenticator mirrors the SQL provider's upsert on (user_id, method).
//
// SereneDB resolves an ON CONFLICT target only against an inline UNIQUE table
// constraint, never against a CREATE UNIQUE INDEX — which is what GORM emits
// for the `uniqueIndex` tag, and what ALTER TABLE cannot retrofit here. So the
// insert runs bare and the conflict is handled from the 23505 the index still
// raises: the loser of a concurrent enrollment updates the winner's row instead
// of leaving a duplicate behind. Same end state as ON CONFLICT DO UPDATE, and
// the same protection against the check-then-insert race above it.
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

	err := func() error {
		err := p.DB().WithContext(ctx).Create(&authenticators).Error
		if err == nil || !uniqueViolation(err) {
			return err
		}
		// Lost the race. Update every column ON CONFLICT ... UPDATE ALL would
		// have written — GORM excludes the primary key from UpdateAll, so id is
		// left on the winning row.
		return p.DB().WithContext(ctx).Model(&schemas.Authenticator{}).
			Where("user_id = ? AND method = ?", authenticators.UserID, authenticators.Method).
			Updates(map[string]any{
				"key":            authenticators.Key,
				"user_id":        authenticators.UserID,
				"method":         authenticators.Method,
				"secret":         authenticators.Secret,
				"recovery_codes": authenticators.RecoveryCodes,
				"verified_at":    authenticators.VerifiedAt,
				"updated_at":     authenticators.UpdatedAt,
			}).Error
	}()
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

// AddVerificationRequest mirrors the SQL provider's upsert on
// (email, identifier). See AddAuthenticator for why ON CONFLICT is unavailable.
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}
	verificationRequest.Key = verificationRequest.ID
	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()

	err := func() error {
		err := p.DB().WithContext(ctx).Create(&verificationRequest).Error
		if err == nil || !uniqueViolation(err) {
			return err
		}
		// Re-requesting verification for the same (email, identifier) replaces
		// the live token, exactly as the DoUpdates column list does on
		// PostgreSQL.
		return p.DB().WithContext(ctx).Model(&schemas.VerificationRequest{}).
			Where("email = ? AND identifier = ?", verificationRequest.Email, verificationRequest.Identifier).
			Updates(map[string]any{
				"token":        verificationRequest.Token,
				"expires_at":   verificationRequest.ExpiresAt,
				"nonce":        verificationRequest.Nonce,
				"redirect_uri": verificationRequest.RedirectURI,
			}).Error
	}()
	if err != nil {
		return verificationRequest, err
	}
	return verificationRequest, nil
}
