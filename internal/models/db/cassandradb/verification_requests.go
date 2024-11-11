package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/models/schemas"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}

	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()

	query := fmt.Sprintf("INSERT INTO %s (id, jwt_token, identifier, expires_at, email, nonce, redirect_uri, created_at, updated_at) VALUES ('%s', '%s', '%s', %d, '%s', '%s', '%s', %d, %d)", KeySpace+"."+schemas.Collections.VerificationRequest, verificationRequest.ID, verificationRequest.Token, verificationRequest.Identifier, verificationRequest.ExpiresAt, verificationRequest.Email, verificationRequest.Nonce, verificationRequest.RedirectURI, verificationRequest.CreatedAt, verificationRequest.UpdatedAt)
	err := p.db.Query(query).Exec()
	if err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	var verificationRequest schemas.VerificationRequest
	query := fmt.Sprintf(`SELECT id, jwt_token, identifier, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM %s WHERE jwt_token = '%s' LIMIT 1`, KeySpace+"."+schemas.Collections.VerificationRequest, token)

	err := p.db.Query(query).Consistency(gocql.One).Scan(&verificationRequest.ID, &verificationRequest.Token, &verificationRequest.Identifier, &verificationRequest.ExpiresAt, &verificationRequest.Email, &verificationRequest.Nonce, &verificationRequest.RedirectURI, &verificationRequest.CreatedAt, &verificationRequest.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	var verificationRequest schemas.VerificationRequest
	query := fmt.Sprintf(`SELECT id, jwt_token, identifier, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM %s WHERE email = '%s' AND identifier = '%s' LIMIT 1 ALLOW FILTERING`, KeySpace+"."+schemas.Collections.VerificationRequest, email, identifier)

	err := p.db.Query(query).Consistency(gocql.One).Scan(&verificationRequest.ID, &verificationRequest.Token, &verificationRequest.Identifier, &verificationRequest.ExpiresAt, &verificationRequest.Email, &verificationRequest.Nonce, &verificationRequest.RedirectURI, &verificationRequest.CreatedAt, &verificationRequest.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) (*model.VerificationRequests, error) {
	var verificationRequests []*model.VerificationRequest
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.VerificationRequest)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, err
	}
	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf(`SELECT id, jwt_token, identifier, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM %s LIMIT %d`, KeySpace+"."+schemas.Collections.VerificationRequest, pagination.Limit+pagination.Offset)

	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var verificationRequest schemas.VerificationRequest
			err := scanner.Scan(&verificationRequest.ID, &verificationRequest.Token, &verificationRequest.Identifier, &verificationRequest.ExpiresAt, &verificationRequest.Email, &verificationRequest.Nonce, &verificationRequest.RedirectURI, &verificationRequest.CreatedAt, &verificationRequest.UpdatedAt)
			if err != nil {
				return nil, err
			}
			verificationRequests = append(verificationRequests, verificationRequest.AsAPIVerificationRequest())
		}
		counter++
	}

	return &model.VerificationRequests{
		VerificationRequests: verificationRequests,
		Pagination:           paginationClone,
	}, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.VerificationRequest, verificationRequest.ID)
	err := p.db.Query(query).Exec()
	if err != nil {
		return err
	}
	return nil
}
