package graphql

import (
	"context"
	"errors"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateSession is used to validate a cookie session without its rotation
// Permission: authorized:user
func (g *graphqlProvider) ValidateSession(ctx context.Context, params *model.ValidateSessionInput) (*model.ValidateSessionResponse, error) {
	log := g.Log.With().Str("func", "ValidateSession").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	sessionToken := ""
	if params != nil && params.Cookie != "" {
		sessionToken = params.Cookie
	} else {
		sessionToken, err = cookie.GetSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get session token")
			return nil, errors.New("unauthorized")
		}
	}
	if sessionToken == "" {
		sessionToken, err = cookie.GetSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get session token")
			return nil, errors.New("unauthorized")
		}
	}
	claims, err := g.TokenProvider.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate session")
		return nil, errors.New("unauthorized")
	}
	userID := claims.Subject
	log.Debug().Str("userID", userID).Msg("Validated session")
	user, err := g.StorageProvider.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByID")
		return nil, err
	}
	// refresh token has "roles" as claim
	claimRoleInterface := claims.Roles
	claimRoles := []string{}
	claimRoles = append(claimRoles, claimRoleInterface...)
	if params != nil && params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug().Str("role", v).Msg("Role not found in claims")
				return nil, fmt.Errorf(`unauthorized`)
			}
		}
	}
	return &model.ValidateSessionResponse{
		IsValid: true,
		User:    user.AsAPIUser(),
	}, nil
}
