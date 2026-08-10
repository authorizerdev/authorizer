package interceptors

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/token"
)

// MCPTokenResolver builds the TokenResolver used by the MCP surface's own,
// bufconn-only gRPC server.
//
// Two differences from the default resolver
// (token.Provider.GetUserIDFromSessionOrAccessToken), both narrowing:
//
//  1. Bearer only. There is no cookie fallback: MCP is a cookieless,
//     Authorization-header protocol (MCP authorization §Access Token Usage), and
//     a browser session cookie must never authenticate a tool call. Dropping the
//     cookie path is also what makes /mcp safe to exempt from CSRF.
//  2. The MCP audience rule. resource is the canonical MCP resource URI,
//     computed once from --url at wiring time and captured here, so no request
//     header can influence which audience is accepted.
//
// Why this exists at all: ValidateAccessToken rejects every resource-bound
// audience at Authorizer's first-party surfaces, and an MCP token's audience is
// required to be exactly that. Rather than relaxing the shared rule — which would
// make an MCP token valid at /graphql too, defeating the binding — the MCP
// transport runs its own gRPC server whose auth interceptor uses this resolver.
//
// Passing a non-nil resolver also makes it the interceptor's SOLE authority: the
// super-admin check and the Session RPC's cookie-only branch are disabled, so
// point 1 above is enforced by the interceptor and not merely by this function
// declining to read cookies. Without that, a credential this resolver never saw
// could still authenticate a tool call. See interceptors.Auth.
//
// What this does NOT cover: a method marked `public` in proto is invoked with no
// principal at all, and the service layer then resolves the caller itself with
// the DEFAULT rule (service.callerTokenData / resolveFgaCaller). An MCP-exposed
// method that is both `public` and identity-resolving would therefore bypass
// everything here. TestExposedMCPToolsCannotBypassTheMCPTokenRule
// (internal/mcp) is what keeps that intersection empty.
func MCPTokenResolver(tp token.Provider, resource string) TokenResolver {
	return func(gc *gin.Context) (*token.SessionOrAccessTokenData, error) {
		accessToken, err := tp.GetAccessToken(gc)
		if err != nil || accessToken == "" {
			return nil, fmt.Errorf(`unauthorized`)
		}
		claims, err := tp.ValidateMCPAccessToken(gc, accessToken, resource)
		if err != nil {
			return nil, err
		}
		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			return nil, fmt.Errorf(`unauthorized: missing sub claim`)
		}
		loginMethod, _ := claims["login_method"].(string)
		nonce, _ := claims["nonce"].(string)
		return &token.SessionOrAccessTokenData{
			UserID:      userID,
			LoginMethod: loginMethod,
			Nonce:       nonce,
			ActorID:     token.ImmediateActor(claims),
			Scope:       token.ClaimScopes(claims),
		}, nil
	}
}
