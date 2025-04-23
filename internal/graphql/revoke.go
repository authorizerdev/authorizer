package graphql

import (
	"context"
	"errors"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Revoke is the method to revoke refresh token
func (g *graphqlProvider) Revoke(ctx context.Context, params *model.OAuthRevokeInput) (*model.Response, error) {
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

	userID := claims["sub"].(string)
	loginMethod := claims["login_method"]
	sessionToken := userID
	if loginMethod != nil && loginMethod != "" {
		sessionToken = loginMethod.(string) + ":" + userID
	}

	existingToken, err := g.MemoryStoreProvider.GetUserSession(sessionToken, constants.TokenTypeRefreshToken+"_"+claims["nonce"].(string))
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
	if err := g.MemoryStoreProvider.DeleteUserSession(sessionToken, claims["nonce"].(string)); err != nil {
		log.Debug().Err(err).Msg("failed to delete user session")
		return nil, err
	}
	return &model.Response{
		Message: "Token revoked",
	}, nil
}
