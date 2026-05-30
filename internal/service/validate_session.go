package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateSession validates a cookie session without rotating it.
// Transport-agnostic port of graphqlProvider.ValidateSession.
//
// Resolution order for the session cookie: explicit params.Cookie first,
// then the request cookies (via cookie.GetSession with a gin shim). Both
// are checked because the GraphQL path historically fell back to the
// cookie when params was empty.
func (p *provider) ValidateSession(ctx context.Context, meta RequestMetadata, params *model.ValidateSessionRequest) (*model.ValidateSessionResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ValidateSession").Logger()

	// TokenProvider.ValidateBrowserSession + cookie.GetSession both take
	// *gin.Context but only read Request fields. Shim it.
	gc := &gin.Context{Request: meta.Request}

	sessionToken := ""
	if params != nil && params.Cookie != "" {
		sessionToken = params.Cookie
	} else {
		var err error
		sessionToken, err = cookie.GetSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get session token")
			return nil, nil, errors.New("unauthorized")
		}
	}
	if sessionToken == "" {
		log.Debug().Msg("Empty session token")
		return nil, nil, errors.New("unauthorized")
	}

	claims, err := p.TokenProvider.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate session")
		return nil, nil, errors.New("unauthorized")
	}
	userID := claims.Subject
	log.Debug().Str("userID", userID).Msg("Validated session")
	user, err := p.StorageProvider.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByID")
		return nil, nil, err
	}

	claimRoles := append([]string{}, claims.Roles...)
	if params != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug().Str("role", v).Msg("Role not found in claims")
				return nil, nil, fmt.Errorf(`unauthorized`)
			}
		}
	}
	if params != nil {
		if err := p.enforceRequiredPermissions(ctx, log, metrics.RequiredPermissionsEndpointValidateSession, user.ID, claimRoles, params.RequiredPermissions); err != nil {
			return nil, nil, err
		}
	}
	return &model.ValidateSessionResponse{
		IsValid: true,
		User:    user.AsAPIUser(),
	}, nil, nil
}
