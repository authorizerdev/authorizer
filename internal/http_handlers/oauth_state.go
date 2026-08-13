package http_handlers

import (
	"encoding/json"
	"errors"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// The social-login `state` used to carry four values through the provider round
// trip, joined with "___" and split apart on the way back:
//
//	state + "___" + redirectURI + "___" + roles + "___" + scope
//
// The FIRST of them is supplied by the caller, so a caller whose state contained
// the delimiter shifted every later field left. Sending
// `state=A___https://evil.example___admin___openid` made the callback read its
// redirect URI from the caller's segment and its ROLES from the next one — and
// the callback checks roles only against ProtectedRoles, not the allowed-roles
// list that /oauth_login enforces on the way in. The RFC 9700 browser binding
// does not help: an attacker crafting this is attacking their own signup, so the
// state is bound to their own browser.
//
// The benign face of the same bug was far more common. A caller's state is
// typically base64url, whose alphabet includes "_", so a token that merely ENDED
// in one produced "____", split a character early, and yielded a redirect URI
// with a leading underscore — a hard "invalid redirect uri" on roughly 1 in 64
// social logins, for every provider.
//
// The fix is not a better delimiter. NOTHING the caller controls travels to the
// provider any more: the state is an opaque random handle, and the four values
// live server-side in the state store, which already held an entry per login.
// There is no format for a caller to collide with because there is nothing to
// parse on the way back.
//
// Encoding the fields instead (base64 per field, joined by a character outside
// the alphabet) would also have closed the injection, but it inflates the state
// by ~28% — and X/Twitter documents a 100-character limit on `state`, which a
// realistic redirect URI already approaches. A fixed-size handle is shorter than
// what this server sent before, so no provider limit gets closer.
//
// It is also less to leak: the redirect URI, roles and scope no longer pass
// through a third party at all.

// oauthStateHandleBytes is the entropy behind the handle. 32 bytes is the same
// budget as the PKCE verifier and yields a 43-character base64url value — well
// inside every provider limit found, including X/Twitter's 100.
const oauthStateHandleBytes = 32

// errMalformedOAuthState means the store held something this server did not
// write. Callers must refuse rather than guess: a lenient parse is exactly how
// caller input became roles in the first place.
var errMalformedOAuthState = errors.New("malformed oauth state")

// oauthStatePayload is what the handle resolves to. It never leaves this server.
type oauthStatePayload struct {
	// Provider is the provider the flow was started for. The callback compares
	// it against its own route parameter, so a code obtained at one provider
	// cannot be redeemed at another.
	Provider string `json:"provider"`
	// State is the CALLER's opaque value, returned to them unchanged at the end
	// of the flow. It is data here, never structure.
	State string `json:"state"`
	// RedirectURI, Roles and Scope were all validated by /oauth_login before
	// this record was written, and nothing between here and the callback can
	// alter them — they never travel to the provider.
	RedirectURI string `json:"redirect_uri"`
	Roles       string `json:"roles"`
	Scope       string `json:"scope"`
}

// newOAuthStateHandle returns the opaque value sent to the provider as `state`.
func newOAuthStateHandle() (string, error) {
	return crypto.NewRandomString(oauthStateHandleBytes)
}

// marshalOAuthState serialises the payload for the state store.
func marshalOAuthState(p oauthStatePayload) (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalOAuthState reads a payload back, failing closed on anything this
// server did not write — including an entry left by a previous release, whose
// value was the bare provider name rather than JSON.
func unmarshalOAuthState(raw string) (oauthStatePayload, error) {
	var p oauthStatePayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return oauthStatePayload{}, errMalformedOAuthState
	}
	if p.Provider == "" {
		return oauthStatePayload{}, errMalformedOAuthState
	}
	return p, nil
}
