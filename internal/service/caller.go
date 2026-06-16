package service

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/token"
)

// callerTokenData returns the authenticated caller identity. The gRPC auth
// interceptor attaches authctx.Principal on success; GraphQL and legacy HTTP
// paths fall back to TokenProvider via the gin shim over meta.Request.
func (p *provider) callerTokenData(ctx context.Context, meta RequestMetadata) (*token.SessionOrAccessTokenData, error) {
	if principal, ok := authctx.FromContext(ctx); ok && principal.UserID != "" {
		return &token.SessionOrAccessTokenData{
			UserID:      principal.UserID,
			LoginMethod: principal.LoginMethod,
			Nonce:       principal.Nonce,
		}, nil
	}
	gc := &gin.Context{Request: meta.Request}
	return p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
}
