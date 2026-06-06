package service

import (
	"context"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Revoke invalidates a refresh token. Typed mirror of RFC 7009.
// Transport-agnostic port of graphqlProvider.Revoke.
func (p *provider) Revoke(ctx context.Context, meta RequestMetadata, params *model.OAuthRevokeRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Revoke").Logger()
	tok := strings.TrimSpace(params.RefreshToken)
	if tok == "" {
		log.Error().Msg("Refresh token is empty")
		return nil, nil, InvalidArgument("missing refresh token")
	}
	claims, err := p.TokenProvider.ParseJWTToken(tok)
	if err != nil {
		log.Debug().Err(err).Msg("failed to parse jwt")
		return nil, nil, err
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		log.Debug().Msg("Invalid subject in token")
		return nil, nil, InvalidArgument("invalid token")
	}
	loginMethod := claims["login_method"]
	sessionKey := userID
	if lm, ok := loginMethod.(string); ok && lm != "" {
		sessionKey = lm + ":" + userID
	}

	nonce, ok := claims["nonce"].(string)
	if !ok || nonce == "" {
		log.Debug().Msg("Invalid nonce in token")
		return nil, nil, InvalidArgument("invalid token")
	}

	existing, err := p.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+nonce)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get refresh token")
		return nil, nil, err
	}
	if existing == "" {
		log.Debug().Msg("Token not found")
		return nil, nil, NotFound("token not found")
	}
	if existing != tok {
		log.Debug().Msg("Token does not match")
		return nil, nil, InvalidArgument("token does not match")
	}

	if err := p.MemoryStoreProvider.DeleteUserSession(sessionKey, nonce); err != nil {
		log.Debug().Err(err).Msg("failed to delete user session")
		return nil, nil, err
	}
	return &model.Response{Message: "Token revoked"}, nil, nil
}
