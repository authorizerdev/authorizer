package http_handlers

import (
	"github.com/authorizerdev/authorizer/internal/codestate"
)

// consumeAuthorizeState resolves the OpenID Connect `/authorize` state (stateValue) into either:
// - (code, codeChallenge, nonce, redirectURI) for authorization-code + PKCE flows, OR
// - (nonce) for implicit/hybrid-style flows.
//
// It is a best-effort bridge used by the social OAuth callback:
// - For standalone social login (`/oauth_login/:provider`) there is no `/authorize` entry, so it returns empty values.
// - For OIDC authorize flows, it consumes the entry to keep it single-use.
func (h *httpProvider) consumeAuthorizeState(stateValue string) (code, codeChallenge, nonce, redirectURI, clientID string, err error) {
	if stateValue == "" {
		return "", "", "", "", "", nil
	}

	authorizeState, err := h.MemoryStoreProvider.GetAndRemoveState(stateValue)
	if err != nil || authorizeState == "" {
		return "", "", "", "", "", err
	}

	// One owner for this positional format — see internal/codestate. A blob
	// written by an older build simply decodes with the trailing fields empty.
	if !codestate.HasCode(authorizeState) {
		return "", "", authorizeState, "", "", nil
	}
	as := codestate.DecodeAuthorize(authorizeState)
	return as.Code, as.Challenge, as.Nonce, as.RedirectURI, as.ClientID, nil
}
