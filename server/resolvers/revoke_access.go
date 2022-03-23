package resolvers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// RevokeAccessResolver is a resolver for delete user mutation
func RevokeAccessResolver(ctx context.Context, params model.UpdateAccessInput) (*model.Response, error) {
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

	user.RevokedTimestamp = time.Now().Unix()

	user, err = db.Provider.UpdateUser(user)
	if err != nil {
		log.Println("error updating user:", err)
		return res, err
	}

	go sessionstore.DeleteAllUserSession(fmt.Sprintf("%x", user.ID))

	res = &model.Response{
		Message: `access revoked successfully`,
	}

	return res, nil
}
