package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ProfileResolver is a resolver for profile query
func ProfileResolver(ctx context.Context) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.User
	if err != nil {
		return res, err
	}

	token, err := utils.GetAuthToken(gc)
	if err != nil {
		return res, err
	}

	claim, err := utils.VerifyAuthToken(token)
	if err != nil {
		return res, err
	}

	userID := fmt.Sprintf("%v", claim["id"])
	email := fmt.Sprintf("%v", claim["email"])
	sessionToken := session.GetUserSession(userID, token)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	user, err := db.Provider.GetUserByEmail(email)
	if err != nil {
		return res, err
	}

	res = utils.GetResponseUserData(user)

	return res, nil
}
