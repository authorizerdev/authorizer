package resolvers

import (
	"context"
	"fmt"
	"log"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// EnableAccessResolver is a resolver for enabling user access
func EnableAccessResolver(ctx context.Context, params model.UpdateAccessInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response
	if err != nil {
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	user, err := db.Provider.GetUserByID(params.UserID)
	if err != nil {
		return res, err
	}

	user.RevokedTimestamp = nil

	user, err = db.Provider.UpdateUser(user)
	if err != nil {
		log.Println("error updating user:", err)
		return res, err
	}

	res = &model.Response{
		Message: `user access enabled successfully`,
	}

	return res, nil
}
