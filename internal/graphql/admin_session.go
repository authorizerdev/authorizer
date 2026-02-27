package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminSession is the method to get admin session.
// Permissions: authorizer:admin
func (g *graphqlProvider) AdminSession(ctx context.Context) (*model.Response, error) {
	log := g.Log.With().Str("func", "AdminLogout").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	hashedKey, err := crypto.EncryptPassword(g.Config.AdminSecret)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to encrypt admin secret")
		return nil, err
	}
	cookie.SetAdminCookie(gc, hashedKey, g.Config.AdminCookieSecure)

	res := &model.Response{
		Message: "admin session refreshed successfully",
	}
	return res, nil
}
