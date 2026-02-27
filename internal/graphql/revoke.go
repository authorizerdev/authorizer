package graphql

import (
	"context"
	"errors"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Revoke is the method to revoke refresh token
func (g *graphqlProvider) Revoke(ctx context.Context, params *model.OAuthRevokeRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "Revoke").Logger()
	token := strings.TrimSpace(params.RefreshToken)
	if token == "" {
		log.Error().Msg("Refresh token is empty")
		return nil, errors.New("missing refresh token")
	}
	claims, err := g.TokenProvider.ParseJWTToken(token)
	if err != nil {
		log.Debug().Err(err).Msg("failed to parse jwt")
		return nil, err
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		log.Debug().Msg("Invalid subject in token")
		return nil, errors.New("invalid token")
	}
	loginMethod := claims["login_method"]
	sessionToken := userID
	if lm, ok := loginMethod.(string); ok && lm != "" {
		sessionToken = lm + ":" + userID
	}

	nonce, ok := claims["nonce"].(string)
	if !ok || nonce == "" {
		log.Debug().Msg("Invalid nonce in token")
		return nil, errors.New("invalid token")
	}

	existingToken, err := g.MemoryStoreProvider.GetUserSession(sessionToken, constants.TokenTypeRefreshToken+"_"+nonce)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get refresh token")
		return nil, err
	}

	if existingToken == "" {
		log.Debug().Msg("Token not found")
		return nil, errors.New("token not found")
	}

	if existingToken != token {
		log.Debug().Msg("Token does not match")
		return nil, errors.New("token does not match")
	}

	// Remove the token from the memory store
	if err := g.MemoryStoreProvider.DeleteUserSession(sessionToken, nonce); err != nil {
		log.Debug().Err(err).Msg("failed to delete user session")
		return nil, err
	}
	return &model.Response{
		Message: "Token revoked",
	}, nil
}
