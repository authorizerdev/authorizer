package couchbase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest models.VerificationRequest) (models.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}

	verificationRequest.Key = verificationRequest.ID
	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.VerificationRequest).Insert(verificationRequest.ID, verificationRequest, &insertOpt)
	if err != nil {
		return verificationRequest, err
	}

	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (models.VerificationRequest, error) {
	verificationRequest := models.VerificationRequest{}
	scope := p.db.Scope("_default")
	params := make(map[string]interface{}, 1)
	params["token"] = token
	query := fmt.Sprintf("SELECT _id, token, identifier, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM auth._default.%s WHERE token=$1 LIMIT 1", models.Collections.VerificationRequest)

	queryResult, err := scope.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{token},
	})

	if err != nil {
		return verificationRequest, err
	}
	err = queryResult.One(&verificationRequest)

	if err != nil {
		return verificationRequest, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (models.VerificationRequest, error) {

	scope := p.db.Scope("_default")
	time.Sleep(200 * time.Millisecond)

	query := fmt.Sprintf("SELECT _id, identifier, token, expires_at, email, nonce, redirect_uri, created_at, updated_at FROM auth._default.%s WHERE email=$1 AND identifier=$2 LIMIT 1", models.Collections.VerificationRequest)
	queryResult, err := scope.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{email, identifier},
	})
	verificationRequest := models.VerificationRequest{}

	if err != nil {
		return verificationRequest, err
	}

	err = queryResult.One(&verificationRequest)

	if err != nil {
		return verificationRequest, err
	}
	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination model.Pagination) (*model.VerificationRequests, error) {
	var verificationRequests []*model.VerificationRequest
	scope := p.db.Scope("_default")
	paginationClone := pagination

	query := fmt.Sprintf("SELECT _id, env, created_at, updated_at FROM auth._default.%s OFFSET %d LIMIT %d", models.Collections.VerificationRequest, paginationClone.Offset, paginationClone.Limit)
	queryResult, err := scope.Query(query, &gocb.QueryOptions{})

	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var verificationRequest models.VerificationRequest
		err := queryResult.Row(&verificationRequest)
		if err != nil {
			log.Fatal(err)
		}
		verificationRequests = append(verificationRequests, verificationRequest.AsAPIVerificationRequest())
	}

	if err := queryResult.Err(); err != nil {
		return nil, err

	}
	return &model.VerificationRequests{
		VerificationRequests: verificationRequests,
		Pagination:           &paginationClone,
	}, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest models.VerificationRequest) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.VerificationRequest).Remove(verificationRequest.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
