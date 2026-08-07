// Package codestate owns the encoding of the two positional, "@@"-delimited
// blobs the authorization-code flow persists in the memory store.
//
// Both formats were previously built and parsed by hand at a dozen call sites
// across internal/http_handlers and internal/service — every one an independent
// chance to put a field in the wrong slot, forget a url.QueryEscape, or add a
// segment to some producers but not others. Adding the client-id binding
// (RFC 6749 §4.1.3) meant touching all of them at once, so the format now has
// exactly one owner.
//
// # Backward compatibility
//
// Decoding is length-guarded field by field, so a blob written by an older
// build (four or five segments, no client id) still decodes — the missing
// fields come back empty and their checks are skipped. Codes issued before a
// deploy therefore remain redeemable across it. Never renumber a slot; only
// append.
package codestate

import (
	"net/url"
	"strings"
)

const delimiter = "@@"

// Code is the state persisted under an authorization code and consumed by the
// token endpoint.
type Code struct {
	// Challenge is the PKCE code_challenge, suffixed "::<method>" when present.
	Challenge string
	// Session is the session token / fingerprint hash minted at authorize time.
	Session string
	// Nonce is the OIDC nonce from the /authorize request.
	Nonce string
	// RedirectURI is the redirect_uri from /authorize (RFC 6749 §4.1.3).
	RedirectURI string
	// Resource is the RFC 8707 resource indicator bound at /authorize.
	Resource string
	// ClientID is the client the code was issued to. RFC 6749 §4.1.3 requires
	// the token endpoint to "ensure that the authorization code was issued to
	// the authenticated confidential client"; without it, two clients sharing a
	// redirect origin can redeem each other's codes.
	ClientID string
}

// EncodeCode serialises the code state. Every free-text field is escaped so a
// value containing the delimiter cannot shift the fields after it.
func EncodeCode(c Code) string {
	return strings.Join([]string{
		c.Challenge,
		c.Session,
		c.Nonce,
		url.QueryEscape(c.RedirectURI),
		url.QueryEscape(c.Resource),
		url.QueryEscape(c.ClientID),
	}, delimiter)
}

// DecodeCode parses the code state, tolerating blobs written by older builds
// that carry fewer segments.
func DecodeCode(raw string) Code {
	parts := strings.Split(raw, delimiter)
	c := Code{}
	if len(parts) > 0 {
		c.Challenge = parts[0]
	}
	if len(parts) > 1 {
		c.Session = parts[1]
	}
	if len(parts) > 2 {
		c.Nonce = parts[2]
	}
	if len(parts) > 3 {
		c.RedirectURI, _ = url.QueryUnescape(parts[3])
	}
	if len(parts) > 4 {
		c.Resource, _ = url.QueryUnescape(parts[4])
	}
	if len(parts) > 5 {
		c.ClientID, _ = url.QueryUnescape(parts[5])
	}
	return c
}

// Authorize is the state persisted under the `state` parameter while an
// /authorize request detours through a login, signup or social-provider round
// trip. Whatever completes that detour rebinds these onto the code state.
type Authorize struct {
	Code        string
	Challenge   string
	Nonce       string
	RedirectURI string
	Resource    string
	ClientID    string
}

// EncodeAuthorize serialises the authorize-detour state.
func EncodeAuthorize(a Authorize) string {
	return strings.Join([]string{
		a.Code,
		a.Challenge,
		a.Nonce,
		url.QueryEscape(a.RedirectURI),
		url.QueryEscape(a.Resource),
		url.QueryEscape(a.ClientID),
	}, delimiter)
}

// DecodeAuthorize parses the authorize-detour state.
//
// A blob with no delimiter at all is the legacy "nonce only" form written when
// the request had no code flow; callers detect that via HasCode.
func DecodeAuthorize(raw string) Authorize {
	parts := strings.Split(raw, delimiter)
	a := Authorize{}
	if len(parts) < 2 {
		a.Nonce = raw
		return a
	}
	a.Code = parts[0]
	a.Challenge = parts[1]
	if len(parts) > 2 {
		a.Nonce = parts[2]
	}
	if len(parts) > 3 {
		a.RedirectURI, _ = url.QueryUnescape(parts[3])
	}
	if len(parts) > 4 {
		a.Resource, _ = url.QueryUnescape(parts[4])
	}
	if len(parts) > 5 {
		a.ClientID, _ = url.QueryUnescape(parts[5])
	}
	return a
}

// HasCode reports whether the blob carried a code flow (as opposed to the
// bare-nonce form).
func HasCode(raw string) bool {
	return strings.Contains(raw, delimiter)
}
