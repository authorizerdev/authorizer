package http_handlers

import "strings"

// consumeAuthorizeState resolves the OpenID Connect `/authorize` state (stateValue) into either:
// - (code, codeChallenge) for authorization-code + PKCE flows, OR
// - (nonce) for implicit/hybrid-style flows.
//
// It is a best-effort bridge used by the social OAuth callback:
// - For standalone social login (`/oauth_login/:provider`) there is no `/authorize` entry, so it returns empty values.
// - For OIDC authorize flows, it consumes the entry to keep it single-use.
func (h *httpProvider) consumeAuthorizeState(stateValue string) (code, codeChallenge, nonce string, err error) {
	if stateValue == "" {
		return "", "", "", nil
	}

	authorizeState, err := h.MemoryStoreProvider.GetState(stateValue)
	if err != nil || authorizeState == "" {
		return "", "", "", err
	}

	authorizeStateSplit := strings.Split(authorizeState, "@@")
	if len(authorizeStateSplit) > 1 {
		code = authorizeStateSplit[0]
		codeChallenge = authorizeStateSplit[1]
	} else {
		nonce = authorizeState
	}

	// Consume authorize state; it should be single-use.
	_ = h.MemoryStoreProvider.RemoveState(stateValue)

	return code, codeChallenge, nonce, nil
}

