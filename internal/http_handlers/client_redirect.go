package http_handlers

import (
	"context"
	"errors"

	"github.com/authorizerdev/authorizer/internal/clientmetadata"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// redirectCheck is the outcome of validating a presented redirect_uri against
// the client that presented it.
type redirectCheck struct {
	// Valid reports whether the URI may be redirected to.
	Valid bool
	// SelfRegistered reports whether the client asserted its own identity
	// (RFC 7591 self-registration). Callers use it to decide whether consent and
	// mandatory PKCE apply. Meaningless when Valid is false.
	SelfRegistered bool
}

// errClientUnavailable means the client could not be CHECKED — a storage
// failure or an unresolvable metadata document — as opposed to a client whose
// redirect_uri simply did not match. Callers must fail closed on it rather than
// falling back to the origin allow-list.
var errClientUnavailable = errors.New("could not verify the client")

// errClientUnresolvable distinguishes the metadata-document case, which is the
// client's fault (a bad client_id) rather than this server's, and so maps to
// invalid_client / 400 instead of temporarily_unavailable / 503.
var errClientUnresolvable = errors.New("could not resolve the client metadata document")

// checkClientRedirectURI decides whether redirectURI is acceptable for clientID.
//
// It exists as one function called from BOTH /authorize and /app because those
// two are halves of a single decision. /authorize validates the redirect and
// then, for an unauthenticated visitor, hands the SAME redirect_uri to /app to
// render the login page. When /app applied a different (AllowedOrigins-only)
// rule, every client whose registered redirect_uri was not also an allowed
// origin passed the first check and failed the second — the login page returned
// "invalid redirect url" and the flow died before the user could type anything.
//
// That gap was invisible in testing for a long time because every fixture
// allow-listed its callback origin. It cannot be papered over that way in
// production: an MCP client binds an EPHEMERAL loopback port, so there is no
// origin an operator could have added in advance.
//
// The precedence deliberately mirrors RFC 6749 §3.1.2.3 / RFC 9700: a client
// that registered exact redirect URIs is held to an exact match against them —
// never a prefix or origin match, which alone would let any path under an
// allowed host through, including a suffix appended to another client's
// callback. AllowedOrigins remains the fallback only for clients that have
// registered nothing, which is what the deployment's own reserved client relies
// on.
func (h *httpProvider) checkClientRedirectURI(ctx context.Context, clientID, redirectURI, hostname string) (redirectCheck, error) {
	// No client_id: this is a non-OAuth use of the login page. The global
	// allow-list is the only rule that could apply.
	if clientID == "" {
		return redirectCheck{Valid: validators.IsValidRedirectURI(redirectURI, h.Config.AllowedOrigins, hostname)}, nil
	}

	// A Client ID Metadata Document client carries its redirect_uris in a
	// document at its own client_id URL rather than in the registry. A
	// resolution failure is fatal rather than a fall-through to the
	// AllowedOrigins fallback: falling through would let an unresolvable
	// client_id inherit a LAXER check than a resolved one, which is backwards.
	if h.ClientMetadataProvider != nil && clientmetadata.IsMetadataClientIDFor(clientID, h.Config.ClientID) {
		doc, err := h.ClientMetadataProvider.Resolve(ctx, clientID)
		if err != nil {
			return redirectCheck{}, errClientUnresolvable
		}
		return redirectCheck{Valid: matchesAny(doc.RedirectURIs, redirectURI)}, nil
	}

	client, err := h.StorageProvider.GetClientByClientID(ctx, clientID)
	// A storage error is NOT "no such client". Collapsing the two silently
	// downgrades the request to the laxer AllowedOrigins check AND leaves a
	// self-asserted client looking operator-registered, so it would skip
	// consent. GetClientByClientID is the documented (nil, nil)-on-absent
	// exception precisely so absent and unavailable stay distinguishable here.
	if err != nil {
		return redirectCheck{}, errClientUnavailable
	}
	if client == nil {
		// Legacy path: no registry row, so the global allow-list stands.
		return redirectCheck{Valid: validators.IsValidRedirectURI(redirectURI, h.Config.AllowedOrigins, hostname)}, nil
	}

	out := redirectCheck{SelfRegistered: constants.IsSelfRegistered(client.Kind)}
	if registered := client.ParsedRedirectURIs(); len(registered) > 0 {
		out.Valid = matchesAny(registered, redirectURI)
		return out, nil
	}
	// A registry row that registered no redirect URIs still falls back, which is
	// what the reserved interactive client depends on.
	out.Valid = validators.IsValidRedirectURI(redirectURI, h.Config.AllowedOrigins, hostname)
	return out, nil
}

// matchesAny reports whether presented satisfies any registered URI, using the
// same matcher everywhere so RFC 8252 loopback rules apply identically.
func matchesAny(registered []string, presented string) bool {
	for _, r := range registered {
		if redirectURIMatches(r, presented) {
			return true
		}
	}
	return false
}
