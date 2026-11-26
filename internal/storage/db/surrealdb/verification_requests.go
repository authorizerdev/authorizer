package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerificationRequest saves a verification request. A second request for
// the same (email, identifier) replaces the pending one — token, expiry,
// nonce and redirect URI — exactly as the SQL provider's upsert does, so the
// newest emailed link is the one that works.
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}
	verificationRequest.Key = verificationRequest.ID
	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()
	err := p.create(ctx, schemas.Collections.VerificationRequest, verificationRequest.ID, verificationRequest)
	if isDuplicate(err) {
		err = p.exec(ctx, `UPDATE type::table($tb)
			SET token = $tok, expires_at = $expires_at, nonce = $nonce, redirect_uri = $redirect_uri, updated_at = $updated_at
			WHERE email = $email AND identifier = $identifier RETURN NONE;`, map[string]any{
			"tb": schemas.Collections.VerificationRequest, "tok": verificationRequest.Token,
			"expires_at": verificationRequest.ExpiresAt, "nonce": verificationRequest.Nonce,
			"redirect_uri": verificationRequest.RedirectURI, "updated_at": verificationRequest.UpdatedAt,
			"email": verificationRequest.Email, "identifier": verificationRequest.Identifier,
		})
		if err == nil {
			// The updated record keeps its own id and created_at; return it
			// rather than the fresh id that was never stored.
			var stored *schemas.VerificationRequest
			stored, err = p.GetVerificationRequestByEmail(ctx, verificationRequest.Email, verificationRequest.Identifier)
			if err == nil {
				*verificationRequest = *stored
			}
		}
	}
	if err != nil {
		return verificationRequest, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	return getWhere[schemas.VerificationRequest](ctx, p, schemas.Collections.VerificationRequest, "token = $v", "verification request", map[string]any{"v": token})
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	return getWhere[schemas.VerificationRequest](ctx, p, schemas.Collections.VerificationRequest, "email = $e AND identifier = $i", "verification request", map[string]any{"e": email, "i": identifier})
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	rows, pag, err := page[schemas.VerificationRequest](ctx, p, schemas.Collections.VerificationRequest, "", nil, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	return p.deleteByID(ctx, schemas.Collections.VerificationRequest, verificationRequest.ID)
}
