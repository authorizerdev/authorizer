package service

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
func (s *service) AdminLogin(ctx context.Context, params *model.AdminLoginInput) (*model.Response, error) {
	log := s.Log.With().Str("func", "AdminLogin").Logger()
	var res *model.Response
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return res, err
	}
	if params.AdminSecret != s.Config.AdminSecret {
		log.Debug().Msg("Invalid admin secret")
		return res, fmt.Errorf(`invalid admin secret`)
	}

	hashedKey, err := crypto.EncryptPassword(s.Config.AdminSecret)
	if err != nil {
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}
