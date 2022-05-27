package resolvers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ValidateJwtTokenResolver is used to validate a jwt token without its rotation
// this can be used at API level (backend)
// it can validate:
// access_token
// id_token
// refresh_token
func ValidateJwtTokenResolver(ctx context.Context, params model.ValidateJWTTokenInput) (*model.ValidateJWTTokenResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	tokenType := params.TokenType
	if tokenType != "access_token" && tokenType != "refresh_token" && tokenType != "id_token" {
		log.Debug("Invalid token type: ", tokenType)
		return nil, errors.New("invalid token type")
	}

	userID := ""
	nonce := ""
	// access_token and refresh_token should be validated from session store as well
	if tokenType == "access_token" || tokenType == "refresh_token" {
		savedSession := memorystore.Provider.GetState(params.Token)
		if savedSession == "" {
			return &model.ValidateJWTTokenResponse{
				IsValid: false,
			}, nil
		}
		savedSessionSplit := strings.Split(savedSession, "@")
		nonce = savedSessionSplit[0]
		userID = savedSessionSplit[1]
	}

	hostname := utils.GetHost(gc)
	var claimRoles []string
	var claims jwt.MapClaims

	// we cannot validate sub and nonce in case of id_token as that token is not persisted in session store
	if userID != "" && nonce != "" {
		claims, err = token.ParseJWTToken(params.Token, hostname, nonce, userID)
		if err != nil {
			log.Debug("Failed to parse jwt token: ", err)
			return &model.ValidateJWTTokenResponse{
				IsValid: false,
			}, nil
		}
	} else {
		claims, err = token.ParseJWTTokenWithoutNonce(params.Token, hostname)
		if err != nil {
			log.Debug("Failed to parse jwt token without nonce: ", err)
			return &model.ValidateJWTTokenResponse{
				IsValid: false,
			}, nil
		}

	}

	claimRolesInterface := claims["roles"]
	roleSlice := utils.ConvertInterfaceToSlice(claimRolesInterface)
	for _, v := range roleSlice {
		claimRoles = append(claimRoles, v.(string))
	}

	if params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug("Token does not have required role: ", v)
				return nil, fmt.Errorf(`unauthorized`)
			}
		}
	}
	return &model.ValidateJWTTokenResponse{
		IsValid: true,
	}, nil
}
