package token

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// backchannelLogoutEventKey is the OIDC BCL 1.0 §2.4 event identifier.
const backchannelLogoutEventKey = "http://schemas.openid.net/event/backchannel-logout"

// backchannelLogoutHTTPTimeout bounds the outbound POST so a slow
// receiver cannot delay the user-facing logout flow. Fire-and-forget.
const backchannelLogoutHTTPTimeout = 5 * time.Second

// backchannelLogoutMaxRedirects bounds redirect chains so a hostile
// receiver cannot bounce us through an arbitrary number of hops.
const backchannelLogoutMaxRedirects = 3

// ErrBackchannelURIEmpty is returned when NotifyBackchannelLogout is
// called with an empty back-channel logout URI.
var ErrBackchannelURIEmpty = errors.New("backchannel logout uri is empty")

// ErrBackchannelMissingHostName is returned when the supplied
// BackchannelLogoutConfig has no HostName (used as the JWT issuer).
var ErrBackchannelMissingHostName = errors.New("backchannel logout requires host name (issuer)")

// ErrBackchannelMissingSubAndSid is returned when the supplied
// BackchannelLogoutConfig has neither a Subject nor a SessionID.
// OIDC BCL 1.0 §2.4 requires at least one of sub or sid.
var ErrBackchannelMissingSubAndSid = errors.New("backchannel logout requires sub or sid")

// BackchannelLogoutConfig holds the per-logout data needed to build
// and send a logout_token per OIDC Back-Channel Logout 1.0 §2.4.
type BackchannelLogoutConfig struct {
	// HostName is the issuer ("iss" claim) of the logout_token. Must be
	// the OP issuer URL and must not be empty.
	HostName string
	// Subject identifies the user being logged out and becomes the
	// "sub" claim. May be empty if SessionID is provided.
	Subject string
	// SessionID identifies the session being terminated and becomes
	// the optional "sid" claim. May be empty if Subject is provided;
	// if empty the sid claim is omitted entirely.
	SessionID string
}

// NotifyBackchannelLogout signs a logout_token JWT per OIDC Back-Channel
// Logout 1.0 §2.4 and POSTs it to the supplied URI. The HTTP request
// is bounded by a 5-second timeout and uses an SSRF-hardened HTTP
// client (see validators.SafeHTTPClient) to prevent the token from
// being delivered to private/internal/loopback addresses. Redirects
// are bounded and re-validated. Intended to be invoked from a goroutine
// so the user-facing logout flow is never blocked. Returns an error
// for local failures (empty URI, missing sub/sid, signing error,
// malformed URI, SSRF rejection) and for remote HTTP failures within
// the timeout — callers running in a goroutine should log and discard.
// The returned error never includes the full URI or any token contents
// (only the host).
func (p *provider) NotifyBackchannelLogout(ctx context.Context, uri string, cfg *BackchannelLogoutConfig) error {
	if strings.TrimSpace(uri) == "" {
		return ErrBackchannelURIEmpty
	}
	if cfg == nil {
		return ErrBackchannelMissingHostName
	}
	if strings.TrimSpace(cfg.HostName) == "" {
		return ErrBackchannelMissingHostName
	}
	if strings.TrimSpace(cfg.Subject) == "" && strings.TrimSpace(cfg.SessionID) == "" {
		return ErrBackchannelMissingSubAndSid
	}

	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": cfg.HostName,
		"aud": p.config.ClientID,
		"iat": now,
		// OIDC BCL §2.4 says SHOULD NOT include exp, but a short bound
		// guards against accidental replay if the receiver caches.
		"exp": now + 300,
		"jti": uuid.New().String(),
		"events": map[string]interface{}{
			backchannelLogoutEventKey: map[string]interface{}{},
		},
	}
	if strings.TrimSpace(cfg.Subject) != "" {
		claims["sub"] = cfg.Subject
	}
	// sid is OPTIONAL per OIDC BCL 1.0 §2.4. Omit when caller did not
	// supply one (e.g. when only sub-based logout is desired). Branch 5
	// of the logout flow relies on this contract.
	if strings.TrimSpace(cfg.SessionID) != "" {
		claims["sid"] = cfg.SessionID
	}
	// nonce is explicitly prohibited by OIDC BCL 1.0 §2.4 — never set.

	signed, err := p.SignJWTToken(claims)
	if err != nil {
		return fmt.Errorf("failed to sign logout_token: %w", err)
	}

	form := url.Values{}
	form.Set("logout_token", signed)

	reqCtx, cancel := context.WithTimeout(ctx, backchannelLogoutHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, uri, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to build backchannel logout request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Resolve the host of the original URI for error reporting and for
	// re-validation of redirect targets.
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("failed to parse backchannel logout uri: %w", err)
	}
	originalHost := parsed.Hostname()

	// SSRF protection: pin the dial to a validated public IP so the HTTP
	// stack cannot be tricked into re-resolving (DNS rebinding TOCTOU).
	// Tests run with Env=test against httptest 127.0.0.1 servers and
	// reuse the existing SkipTestEndpointSSRFValidation toggle as a
	// blanket "this build is a test" indicator — production builds must
	// never enable that flag.
	skipSSRF := p.config.Env == constants.TestEnv && p.config.SkipTestEndpointSSRFValidation
	var client *http.Client
	if skipSSRF {
		client = &http.Client{Timeout: backchannelLogoutHTTPTimeout}
	} else {
		client, err = validators.SafeHTTPClient(ctx, uri, backchannelLogoutHTTPTimeout)
		if err != nil {
			return fmt.Errorf("backchannel logout uri rejected by SSRF filter (host=%s): %w", originalHost, err)
		}
	}

	// Bound redirect chains and re-validate every hop. SafeHTTPClient
	// pins the dialer to the IP of the *original* host, so a redirect
	// to a different host would otherwise still be dialed against that
	// pinned IP — we explicitly reject cross-host redirects, and cap the
	// chain length.
	client.CheckRedirect = func(redirReq *http.Request, via []*http.Request) error {
		if len(via) >= backchannelLogoutMaxRedirects {
			return fmt.Errorf("too many redirects")
		}
		if redirReq.URL.Hostname() != originalHost {
			return fmt.Errorf("cross-host redirect not permitted")
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		// net/http wraps the URL into the error string; strip it so we
		// never log the full URI (which may include path/query secrets).
		return fmt.Errorf("backchannel logout request failed (host=%s): %w", originalHost, sanitizeHTTPError(err, uri))
	}
	defer resp.Body.Close()
	// Drain the body so the underlying connection can be reused and so
	// the receiver does not see a half-closed write.
	_, _ = io.Copy(io.Discard, resp.Body)

	// OIDC BCL 1.0 §2.8: the OP MUST treat 2xx as success and any other
	// status as failure.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("backchannel logout receiver returned status %d (host=%s)", resp.StatusCode, originalHost)
	}
	return nil
}

// sanitizeHTTPError removes the request URL from a net/http error so
// callers can safely log it without leaking the full backchannel URI.
func sanitizeHTTPError(err error, fullURL string) error {
	if err == nil {
		return nil
	}
	msg := strings.ReplaceAll(err.Error(), fullURL, "<redacted>")
	return errors.New(msg)
}
