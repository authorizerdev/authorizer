package resolvers

import (
	"context"
	"errors"
	"fmt"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	log "github.com/sirupsen/logrus"
)

// ValidateSessionResolver is used to validate a cookie session without its rotation
func ValidateSessionResolver(ctx context.Context, params *model.ValidateSessionInput) (*model.ValidateSessionResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}
	sessionToken := params.Cookie
	if sessionToken == "" {
		sessionToken, err = cookie.GetSession(gc)
		if err != nil {
			log.Debug("Failed to get session token: ", err)
			return nil, errors.New("unauthorized")
		}
	}
	claims, err := token.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug("Failed to validate session token", err)
		return nil, errors.New("unauthorized")
	}
	userID := claims.Subject
	log := log.WithFields(log.Fields{
		"user_id": userID,
	})
	_, err = db.Provider.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// refresh token has "roles" as claim
	claimRoleInterface := claims.Roles
	claimRoles := []string{}
	claimRoles = append(claimRoles, claimRoleInterface...)
	if params != nil && params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug("User does not have required role: ", claimRoles, v)
				return nil, fmt.Errorf(`unauthorized`)
			}
		}
	}
	return &model.ValidateSessionResponse{
		IsValid: true,
	}, nil
}
