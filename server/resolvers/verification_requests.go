package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerificationRequestsResolver is a resolver for verification requests query
// This is admin only query
func VerificationRequestsResolver(ctx context.Context) ([]*model.VerificationRequest, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res []*model.VerificationRequest
	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	verificationRequests, err := db.Mgr.GetVerificationRequests()
	if err != nil {
		return res, err
	}

	for i := 0; i < len(verificationRequests); i++ {
		res = append(res, &model.VerificationRequest{
			ID:         fmt.Sprintf("%v", verificationRequests[i].ID),
			Email:      &verificationRequests[i].Email,
			Token:      &verificationRequests[i].Token,
			Identifier: &verificationRequests[i].Identifier,
			Expires:    &verificationRequests[i].ExpiresAt,
			CreatedAt:  &verificationRequests[i].CreatedAt,
			UpdatedAt:  &verificationRequests[i].UpdatedAt,
		})
	}

	return res, nil
}
