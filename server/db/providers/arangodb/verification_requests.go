package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(verificationRequest models.VerificationRequest) (models.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}

	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()
	verificationRequestRequestCollection, _ := p.db.Collection(nil, models.Collections.VerificationRequest)
	meta, err := verificationRequestRequestCollection.CreateDocument(nil, verificationRequest)
	if err != nil {
		return verificationRequest, err
	}
	verificationRequest.Key = meta.Key
	verificationRequest.ID = meta.ID.String()

	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(token string) (models.VerificationRequest, error) {
	var verificationRequest models.VerificationRequest
	query := fmt.Sprintf("FOR d in %s FILTER d.token == @token LIMIT 1 RETURN d", models.Collections.VerificationRequest)
	bindVars := map[string]interface{}{
		"token": token,
	}

	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		return verificationRequest, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if verificationRequest.Key == "" {
				return verificationRequest, fmt.Errorf("verification request not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &verificationRequest)
		if err != nil {
			return verificationRequest, err
		}
	}

	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(email string, identifier string) (models.VerificationRequest, error) {
	var verificationRequest models.VerificationRequest

	query := fmt.Sprintf("FOR d in %s FILTER d.email == @email FILTER d.identifier == @identifier LIMIT 1 RETURN d", models.Collections.VerificationRequest)
	bindVars := map[string]interface{}{
		"email":      email,
		"identifier": identifier,
	}

	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		return verificationRequest, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if verificationRequest.Key == "" {
				return verificationRequest, fmt.Errorf("verification request not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &verificationRequest)
		if err != nil {
			return verificationRequest, err
		}
	}

	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(pagination model.Pagination) (*model.VerificationRequests, error) {
	var verificationRequests []*model.VerificationRequest
	ctx := driver.WithQueryFullCount(context.Background())
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", models.Collections.VerificationRequest, pagination.Offset, pagination.Limit)

	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()

	for {
		var verificationRequest models.VerificationRequest
		meta, err := cursor.ReadDocument(nil, &verificationRequest)

		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}

		if meta.Key != "" {
			verificationRequests = append(verificationRequests, verificationRequest.AsAPIVerificationRequest())
		}

	}

	return &model.VerificationRequests{
		VerificationRequests: verificationRequests,
		Pagination:           &paginationClone,
	}, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(verificationRequest models.VerificationRequest) error {
	collection, _ := p.db.Collection(nil, models.Collections.VerificationRequest)
	_, err := collection.RemoveDocument(nil, verificationRequest.Key)
	if err != nil {
		return err
	}
	return nil
}
