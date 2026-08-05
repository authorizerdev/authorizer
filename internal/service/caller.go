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
			// Carried so both branches describe the caller identically. Dropping
			// it here made an agent's actions on gRPC indistinguishable from the
			// user performing them, while the same call on GraphQL — which goes
			// through GetUserIDFromSessionOrAccessToken and does populate it —
			// attributed them correctly. Divergence between transports on WHO
			// did something is not a cosmetic bug.
			ActorID: principal.ActorID,
		}, nil
	}
	gc := &gin.Context{Request: meta.Request}
	return p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
}
