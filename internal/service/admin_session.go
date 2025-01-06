package service

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
func (s *service) AdminSession(ctx context.Context) (*model.Response, error) {
	log := s.Log.With().Str("func", "AdminLogout").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	hashedKey, err := crypto.EncryptPassword(s.Config.AdminSecret)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to encrypt admin secret")
		return nil, err
	}
	cookie.SetAdminCookie(gc, hashedKey)

	res := &model.Response{
		Message: "admin session refreshed successfully",
	}
	return res, nil
}
