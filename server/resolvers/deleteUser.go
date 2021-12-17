package resolvers

import (
	"context"
	"fmt"
	"log"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func DeleteUser(ctx context.Context, params model.DeleteUserInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response
	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	user, err := db.Mgr.GetUserByEmail(params.Email)
	if err != nil {
		return res, err
	}

	session.DeleteUserSession(fmt.Sprintf("%x", user.ID))

	err = db.Mgr.DeleteUser(user)
	if err != nil {
		log.Println("Err:", err)
		return res, err
	}

	res = &model.Response{
		Message: `user deleted successfully`,
	}

	return res, nil
}
