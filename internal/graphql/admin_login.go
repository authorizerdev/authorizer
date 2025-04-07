package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminLogin is the method to login as admin.
// Permissions: none
func (g *graphqlProvider) AdminLogin(ctx context.Context, params *model.AdminLoginInput) (*model.Response, error) {
	log := g.Log.With().Str("func", "AdminLogin").Logger()
	var res *model.Response
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return res, err
	}
	if params.AdminSecret != g.Config.AdminSecret {
		log.Debug().Msg("Invalid admin secret")
		return res, fmt.Errorf(`invalid admin secret`)
	}

	hashedKey, err := crypto.EncryptPassword(g.Config.AdminSecret)
	if err != nil {
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey, g.Config.AdminCookieSecure)

	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}
