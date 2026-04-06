package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
		verificationRequest.CreatedAt = time.Now().Unix()
		verificationRequest.UpdatedAt = time.Now().Unix()
		if err := p.putItem(ctx, schemas.Collections.VerificationRequest, verificationRequest); err != nil {
			return nil, err
		}
	}
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	items, err := p.queryEq(ctx, schemas.Collections.VerificationRequest, "token", "token", token, nil)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	var v schemas.VerificationRequest
	if err := unmarshalItem(items[0], &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	f := expression.Name("email").Equal(expression.Value(email)).And(expression.Name("identifier").Equal(expression.Value(identifier)))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.VerificationRequest, nil, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	var v schemas.VerificationRequest
	if err := unmarshalItem(items[0], &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var verificationRequests []*schemas.VerificationRequest

	count, err := p.scanCount(ctx, schemas.Collections.VerificationRequest, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.VerificationRequest, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var v schemas.VerificationRequest
			if err := unmarshalItem(it, &v); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				verificationRequests = append(verificationRequests, &v)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return verificationRequests, paginationClone, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	if verificationRequest == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.VerificationRequest, "id", verificationRequest.ID)
}
