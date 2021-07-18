package resolvers

import (
	"context"
	"fmt"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/utils"
)

func VerificationRequests(ctx context.Context) ([]*model.VerificationRequest, error) {
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

	for _, verificationRequest := range verificationRequests {
		res = append(res, &model.VerificationRequest{
			ID:         fmt.Sprintf("%d", verificationRequest.ID),
			Email:      &verificationRequest.Email,
			Token:      &verificationRequest.Token,
			Identifier: &verificationRequest.Identifier,
			Expires:    &verificationRequest.ExpiresAt,
			CreatedAt:  &verificationRequest.CreatedAt,
			UpdatedAt:  &verificationRequest.UpdatedAt,
		})
	}

	return res, nil
}
