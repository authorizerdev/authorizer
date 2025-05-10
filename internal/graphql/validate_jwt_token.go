package graphql

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateJwtToken is used to validate a jwt token without its rotation
// this can be used at API level (backend)
// it can validate:
// access_token
// id_token
// refresh_token
// Permission: none
func (g *graphqlProvider) ValidateJWTToken(ctx context.Context, params *model.ValidateJWTTokenRequest) (*model.ValidateJWTTokenResponse, error) {
	log := g.Log.With().Str("func", "ValidateJWTToken").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	tokenType := params.TokenType
	if tokenType != constants.TokenTypeAccessToken && tokenType != constants.TokenTypeRefreshToken && tokenType != constants.TokenTypeIdentityToken {
		log.Debug().Str("token_type", tokenType).Msg("Invalid token type")
		return nil, errors.New("invalid token type")
	}

	var claimRoles []string
	var claims jwt.MapClaims
	userID := ""
	nonce := ""

	claims, err = g.TokenProvider.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse jwt token")
		return nil, err
	}
	userID = claims["sub"].(string)

	// access_token and refresh_token should be validated from session store as well
	if tokenType == constants.TokenTypeAccessToken || tokenType == constants.TokenTypeRefreshToken {
		nonce = claims["nonce"].(string)
		loginMethod := claims["login_method"]
		sessionKey := userID
		if loginMethod != nil && loginMethod != "" {
			sessionKey = loginMethod.(string) + ":" + userID
		}
		token, err := g.MemoryStoreProvider.GetUserSession(sessionKey, tokenType+"_"+claims["nonce"].(string))
		if err != nil || token == "" {
			log.Debug().Err(err).Msg("Failed to get token from session store")
			return nil, errors.New("invalid token")
		}
	}

	hostname := parsers.GetHost(gc)

	// we cannot validate nonce in case of id_token as that token is not persisted in session store
	if nonce != "" {
		if ok, err := g.TokenProvider.ValidateJWTClaims(claims, &token.AuthTokenConfig{
			HostName: hostname,
			Nonce:    nonce,
			User: &schemas.User{
				ID: userID,
			},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Failed to validate jwt claims")
			return nil, errors.New("invalid claims")
		}
	} else {
		if ok, err := g.TokenProvider.ValidateJWTTokenWithoutNonce(claims, &token.AuthTokenConfig{
			HostName: hostname,
			User: &schemas.User{
				ID: userID,
			},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Failed to validate jwt claims")
			return nil, errors.New("invalid claims")
		}
	}

	claimKey := g.Config.JWTRoleClaim
	claimRolesInterface := claims[claimKey]
	roleSlice := utils.ConvertInterfaceToSlice(claimRolesInterface)
	for _, v := range roleSlice {
		claimRoles = append(claimRoles, v.(string))
	}

	if len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug().Str("role", v).Msg("Role not found in claims")
				return nil, fmt.Errorf(`unauthorized`)
			}
		}
	}
	return &model.ValidateJWTTokenResponse{
		IsValid: true,
		Claims:  claims,
	}, nil
}
