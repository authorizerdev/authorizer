package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/validators"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// backchannelLogoutEventKey is the OIDC BCL 1.0 §2.4 event identifier.
const backchannelLogoutEventKey = "http://schemas.openid.net/event/backchannel-logout"

// backchannelLogoutHTTPTimeout bounds the outbound POST so a slow
// receiver cannot delay the user-facing logout flow. Fire-and-forget.
const backchannelLogoutHTTPTimeout = 5 * time.Second

// BackchannelLogoutConfig holds the per-logout data needed to build
// and send a logout_token. The HostName is the issuer; the Subject
// identifies the user; SessionID is echoed as the sid claim.
type BackchannelLogoutConfig struct {
	HostName  string
	Subject   string
	SessionID string
}

// NotifyBackchannelLogout signs a logout_token JWT per OIDC Back-Channel
// Logout 1.0 §2.4 and POSTs it to the supplied URI. The HTTP request
// is bounded by a 5-second timeout. Intended to be invoked from a
// goroutine so the user-facing logout flow is never blocked. Returns
// an error for local failures (empty URI, missing sub/sid, signing
// error, malformed URI) and for remote HTTP failures within the
// timeout — callers running in a goroutine should log and discard.
func (p *provider) NotifyBackchannelLogout(ctx context.Context, uri string, cfg *BackchannelLogoutConfig) error {
	if strings.TrimSpace(uri) == "" {
		return errors.New("backchannel logout uri is empty")
	}
	if cfg == nil || (strings.TrimSpace(cfg.Subject) == "" && strings.TrimSpace(cfg.SessionID) == "") {
		return errors.New("backchannel logout requires sub or sid")
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
	if strings.TrimSpace(cfg.SessionID) != "" {
		claims["sid"] = cfg.SessionID
	}
	// nonce is explicitly prohibited by OIDC BCL 1.0 §2.4 — never set.

	signed, err := p.SignJWTToken(claims)
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("logout_token", signed)

	reqCtx, cancel := context.WithTimeout(ctx, backchannelLogoutHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, uri, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client, err := validators.SafeHTTPClient(reqCtx, uri, backchannelLogoutHTTPTimeout)
	if err != nil {
		return fmt.Errorf("backchannel logout SSRF check: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
