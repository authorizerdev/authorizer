package couchbase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}
	verificationRequest.Key = verificationRequest.ID
	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.VerificationRequest).Insert(verificationRequest.ID, verificationRequest, &insertOpt)
	if err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest
	params := make(map[string]interface{}, 1)
	params["token"] = token
	query := fmt.Sprintf("SELECT _id, token, identifier, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM %s.%s WHERE token=$1 LIMIT 1", p.scopeName, schemas.Collections.VerificationRequest)

	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{token},
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
	})

	if err != nil {
		return nil, err
	}
	err = queryResult.One(&verificationRequest)

	if err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {

	query := fmt.Sprintf("SELECT _id, identifier, token, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM %s.%s WHERE email=$1 AND identifier=$2 LIMIT 1", p.scopeName, schemas.Collections.VerificationRequest)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{email, identifier},
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
	})
	if err != nil {
		return nil, err
	}
	var verificationRequest *schemas.VerificationRequest
	err = queryResult.One(&verificationRequest)
	if err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	var verificationRequests []*schemas.VerificationRequest
	paginationClone := pagination
	total, err := p.GetTotalDocs(ctx, schemas.Collections.VerificationRequest)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, env, created_at, updated_at FROM %s.%s OFFSET $1 LIMIT $2", p.scopeName, schemas.Collections.VerificationRequest)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{paginationClone.Offset, paginationClone.Limit},
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var verificationRequest schemas.VerificationRequest
		err := queryResult.Row(&verificationRequest)
		if err != nil {
			log.Fatal(err)
		}
		verificationRequests = append(verificationRequests, &verificationRequest)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err

	}
	return verificationRequests, paginationClone, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.VerificationRequest).Remove(verificationRequest.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
