package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminLogout is the method to logout as admin.
// Permissions: authorizer:admin
func (s *service) AdminLogout(ctx context.Context) (*model.Response, error) {
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

	cookie.DeleteAdminCookie(gc)

	res := &model.Response{
		Message: "admin logged out successfully",
	}
	return res, nil
}
