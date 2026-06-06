package service

import (
	"context"

	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateJwtToken validates a JWT without rotating it. Used at the API
// level (backend) — accepts access_token, id_token, or refresh_token.
// Transport-agnostic port of graphqlProvider.ValidateJWTToken.
func (p *provider) ValidateJwtToken(ctx context.Context, meta RequestMetadata, params *model.ValidateJWTTokenRequest) (*model.ValidateJWTTokenResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ValidateJwtToken").Logger()

	tokenType := params.TokenType
	if tokenType != constants.TokenTypeAccessToken && tokenType != constants.TokenTypeRefreshToken && tokenType != constants.TokenTypeIdentityToken {
		log.Debug().Str("token_type", tokenType).Msg("Invalid token type")
		return nil, nil, InvalidArgument("invalid token type")
	}

	var claimRoles []string
	var claims jwt.MapClaims
	userID := ""
	nonce := ""

	claims, err := p.TokenProvider.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse jwt token")
		return nil, nil, err
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		log.Debug().Msg("Invalid subject in token")
		return nil, nil, Unauthenticated("invalid token")
	}
	userID = sub

	if tokenType == constants.TokenTypeAccessToken || tokenType == constants.TokenTypeRefreshToken {
		nonceVal, ok := claims["nonce"].(string)
		if !ok || nonceVal == "" {
			log.Debug().Msg("Invalid nonce in token")
			return nil, nil, Unauthenticated("invalid token")
		}
		nonce = nonceVal
		loginMethod := claims["login_method"]
		sessionKey := userID
		if lm, ok := loginMethod.(string); ok && lm != "" {
			sessionKey = lm + ":" + userID
		}
		tok, err := p.MemoryStoreProvider.GetUserSession(sessionKey, tokenType+"_"+nonceVal)
		if err != nil || tok == "" {
			log.Debug().Err(err).Msg("Failed to get token from session store")
			return nil, nil, Unauthenticated("invalid token")
		}
	}

	hostname := meta.HostURL
	if nonce != "" {
		if ok, err := p.TokenProvider.ValidateJWTClaims(claims, &token.AuthTokenConfig{
			HostName: hostname,
			Nonce:    nonce,
			User:     &schemas.User{ID: userID},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Failed to validate jwt claims")
			return nil, nil, Unauthenticated("invalid claims")
		}
	} else {
		if ok, err := p.TokenProvider.ValidateJWTTokenWithoutNonce(claims, &token.AuthTokenConfig{
			HostName: hostname,
			User:     &schemas.User{ID: userID},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Failed to validate jwt claims")
			return nil, nil, Unauthenticated("invalid claims")
		}
	}

	// Read roles from the configured claim key (used for id_token), falling
	// back to the hardcoded "roles" claim that CreateAccessToken emits.
	claimRolesInterface := claims[p.Config.JWTRoleClaim]
	roleSlice := utils.ConvertInterfaceToSlice(claimRolesInterface)
	if len(roleSlice) == 0 {
		roleSlice = utils.ConvertInterfaceToSlice(claims["roles"])
	}
	for _, v := range roleSlice {
		roleStr, ok := v.(string)
		if !ok || roleStr == "" {
			log.Debug().Msg("Invalid role claim value")
			return nil, nil, Unauthenticated("invalid claims")
		}
		claimRoles = append(claimRoles, roleStr)
	}

	if len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug().Str("role", v).Msg("Role not found in claims")
				return nil, nil, Unauthenticated("unauthorized")
			}
		}
	}
	if err := p.enforceRequiredPermissions(ctx, log, metrics.RequiredPermissionsEndpointValidateJWTToken, userID, claimRoles, params.RequiredPermissions); err != nil {
		return nil, nil, err
	}
	return &model.ValidateJWTTokenResponse{
		IsValid: true,
		Claims:  claims,
	}, nil, nil
}
