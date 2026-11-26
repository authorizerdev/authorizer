package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerificationRequest saves a verification request. It is an upsert on
// (email, identifier), matching every other backend: a pending request for the
// same pair is replaced in place (new token, expiry, nonce, redirect URI)
// rather than rejected by the unique hash index.
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}
	if verificationRequest.Key == "" {
		verificationRequest.Key = verificationRequest.ID
	}
	now := time.Now().Unix()
	query := fmt.Sprintf(`UPSERT { email: @email, identifier: @identifier }
		INSERT { _key: @key, email: @email, identifier: @identifier, token: @token, expires_at: @expires_at, nonce: @nonce, redirect_uri: @redirect_uri, created_at: @now, updated_at: @now }
		UPDATE { token: @token, expires_at: @expires_at, nonce: @nonce, redirect_uri: @redirect_uri, updated_at: @now }
		IN %s RETURN NEW`, schemas.Collections.VerificationRequest)
	bindVars := map[string]interface{}{
		"key":          verificationRequest.Key,
		"email":        verificationRequest.Email,
		"identifier":   verificationRequest.Identifier,
		"token":        verificationRequest.Token,
		"expires_at":   verificationRequest.ExpiresAt,
		"nonce":        verificationRequest.Nonce,
		"redirect_uri": verificationRequest.RedirectURI,
		"now":          now,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if arangoDriver.IsArangoErrorWithErrorNum(err, arangoErrUniqueConstraintViolated) {
		// UPSERT's lookup and insert are not atomic: two concurrent first
		// requests for the same pair can both miss and both insert, and the
		// unique index rejects the loser. Retrying once finds the winner's
		// document and takes the UPDATE branch instead.
		cursor, err = p.db.Query(ctx, query, bindVars)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	if _, err := cursor.ReadDocument(ctx, verificationRequest); err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// arangoErrUniqueConstraintViolated is ArangoDB's ERROR_ARANGO_UNIQUE_CONSTRAINT_VIOLATED.
const arangoErrUniqueConstraintViolated = 1210

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest
	query := fmt.Sprintf("FOR d in %s FILTER d.token == @token LIMIT 1 RETURN d", schemas.Collections.VerificationRequest)
	bindVars := map[string]interface{}{
		"token": token,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if verificationRequest == nil {
				return verificationRequest, fmt.Errorf("verification request not found: %w", ErrNotFound)
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &verificationRequest)
		if err != nil {
			return nil, err
		}
	}
	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest
	query := fmt.Sprintf("FOR d in %s FILTER d.email == @email FILTER d.identifier == @identifier LIMIT 1 RETURN d", schemas.Collections.VerificationRequest)
	bindVars := map[string]interface{}{
		"email":      email,
		"identifier": identifier,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if verificationRequest == nil {
				return verificationRequest, fmt.Errorf("verification request not found: %w", ErrNotFound)
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &verificationRequest)
		if err != nil {
			return nil, err
		}
	}
	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	var verificationRequests []*schemas.VerificationRequest
	sctx := arangoDriver.WithQueryFullCount(ctx)
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT @offset, @limit RETURN d", schemas.Collections.VerificationRequest)
	cursor, err := p.db.Query(sctx, query, map[string]interface{}{
		"offset": pagination.Offset,
		"limit":  pagination.Limit,
	})
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var verificationRequest *schemas.VerificationRequest
		meta, err := cursor.ReadDocument(ctx, &verificationRequest)

		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}

		if meta.Key != "" {
			verificationRequests = append(verificationRequests, verificationRequest)
		}

	}
	return verificationRequests, paginationClone, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	collection, _ := p.db.Collection(ctx, schemas.Collections.VerificationRequest)
	_, err := collection.RemoveDocument(ctx, verificationRequest.Key)
	if err != nil {
		return err
	}
	return nil
}
