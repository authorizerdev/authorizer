package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Revoke is the method to revoke refresh token
func (g *graphqlProvider) Revoke(ctx context.Context, params *model.OAuthRevokeInput) (*model.Response, error) {
	log := g.Log.With().Str("func", "Revoke").Logger()
	if err := g.MemoryStoreProvider.RemoveState(params.RefreshToken); err != nil {
		log.Debug().Err(err).Msg("Failed to revoke given token")
		return nil, err
	}
	return &model.Response{
		Message: "Token revoked",
	}, nil
}
