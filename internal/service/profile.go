package service

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Profile returns the authenticated user. Requires a valid session cookie or
// access-token bearer. Transport-agnostic port of graphqlProvider.Profile.
//
// Permissions: authenticated user.
func (p *provider) Profile(ctx context.Context, meta RequestMetadata) (*model.User, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Profile").Logger()

	// TokenProvider.GetUserIDFromSessionOrAccessToken takes *gin.Context but
	// only reads Request headers (Authorization) and cookies. Synthesize a
	// minimal gin context wrapping the inbound *http.Request — same shim
	// pattern as the original SignUp migration.
	// TODO(grpc): refactor TokenProvider to take *http.Request directly.
	gc := &gin.Context{Request: meta.Request}
	tokenData, err := p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, nil, fmt.Errorf("unauthorized")
	}
	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Str("user_id", tokenData.UserID).Msg("Failed to get user by id")
		return nil, nil, err
	}
	return user.AsAPIUser(), nil, nil
}
