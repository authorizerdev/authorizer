package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ProfileResolver is a resolver for profile query
func ProfileResolver(ctx context.Context) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.User
	if err != nil {
		return res, err
	}

	accessToken, err := token.GetAccessToken(gc)
	if err != nil {
		return res, err
	}

	claims, err := token.ValidateAccessToken(gc, accessToken)
	if err != nil {
		return res, err
	}

	userID := claims["sub"].(string)
	user, err := db.Provider.GetUserByID(userID)
	if err != nil {
		return res, err
	}

	return user.AsAPIUser(), nil
}
