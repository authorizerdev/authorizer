package http_handlers

import (
	"net/url"
	"strings"
)

// consumeAuthorizeState resolves the OpenID Connect `/authorize` state (stateValue) into either:
// - (code, codeChallenge, nonce, redirectURI) for authorization-code + PKCE flows, OR
// - (nonce) for implicit/hybrid-style flows.
//
// It is a best-effort bridge used by the social OAuth callback:
// - For standalone social login (`/oauth_login/:provider`) there is no `/authorize` entry, so it returns empty values.
// - For OIDC authorize flows, it consumes the entry to keep it single-use.
func (h *httpProvider) consumeAuthorizeState(stateValue string) (code, codeChallenge, nonce, redirectURI string, err error) {
	if stateValue == "" {
		return "", "", "", "", nil
	}

	authorizeState, err := h.MemoryStoreProvider.GetState(stateValue)
	if err != nil || authorizeState == "" {
		return "", "", "", "", err
	}

	authorizeStateSplit := strings.Split(authorizeState, "@@")
	if len(authorizeStateSplit) > 1 {
		code = authorizeStateSplit[0]
		codeChallenge = authorizeStateSplit[1]
		// Third part carries the OIDC nonce from the /authorize request.
		if len(authorizeStateSplit) > 2 {
			nonce = authorizeStateSplit[2]
		}
		// Fourth part carries the URL-encoded redirect_uri from the /authorize
		// request for RFC 6749 §4.1.3 validation at the token endpoint.
		// It is URL-encoded to prevent the @@ delimiter from being confused
		// with @@ characters that may appear in the redirect_uri.
		if len(authorizeStateSplit) > 3 {
			redirectURI, _ = url.QueryUnescape(authorizeStateSplit[3])
		}
	} else {
		nonce = authorizeState
	}

	// Consume authorize state; it should be single-use.
	_ = h.MemoryStoreProvider.RemoveState(stateValue)

	return code, codeChallenge, nonce, redirectURI, nil
}
