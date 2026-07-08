package http_handlers

// Per-organization enterprise OIDC SSO — Authorizer acting as the Relying Party
// (broker). An org configures its upstream OIDC IdP (Okta/Entra/Google) as a
// sso_oidc TrustedIssuer row; its users log in through that IdP and Authorizer
// JIT-provisions them and issues a normal Authorizer session.
//
// SECURITY MODEL (design §4.4 / §5.2):
//   - PKCE (S256), unguessable `state`, and `nonce` are sent to the upstream and
//     verified on return. `state` is single-use (GetAndRemoveState) — CSRF/replay.
//   - Mix-up defense (G3 / RFC 9207): the returned ID token's `iss` MUST equal the
//     issuer the dispatching connection discovered (stored in the per-state flow);
//     an `iss` query parameter, when present, must match too. A response bound to
//     a different connection is rejected.
//   - The ID token is verified against the upstream JWKS (SSRF-hardened fetch),
//     `aud` == our upstream client_id, `nonce` bound, `exp` valid, asymmetric alg.
//   - JIT provisioning is namespaced by (org_id, issuer, sub) via FederatedIdentity
//     — an email that merely collides with an existing account is NEVER linked
//     (account-takeover defense); such a collision is rejected fail-closed.
//   - All upstream fetches (discovery, JWKS, token exchange) use
//     validators.SafeHTTPClient (host-pinned, redirect-refused, size-capped).
//
// ponytail: SAML SP (kind=sso_saml) is intentionally NOT here — its signature-
// wrapping (XSW) validation ships in a separate PR for a focused security review.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

const (
	// ssoFlowPrefix namespaces the per-state broker flow entry in the shared store.
	// SetState applies the store's short OAuth-state TTL, so the flow expires if
	// the callback never arrives.
	ssoFlowPrefix = "sso_flow:"
	// ssoHTTPTimeout caps a single upstream fetch (discovery/JWKS/token).
	ssoHTTPTimeout = 10 * time.Second
	// ssoMaxRespBytes caps a response body read from an issuer-controlled endpoint.
	ssoMaxRespBytes = 1 << 20 // 1 MiB
	// ssoClockSkew tolerates minor clock drift on the upstream ID token exp/iat.
	ssoClockSkew = 60 * time.Second
)

// ssoAllowedAlgs is the asymmetric ID-token signature allow-list (RFC 8725 §3.1);
// alg:none and every symmetric (HS*) algorithm are absent by construction.
var ssoAllowedAlgs = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
}

// ssoFlowState is the per-state broker context, stored single-use in the shared
// store between login and callback. It pins the connection the response must come
// from (ExpectedIssuer) so a mixed-up response is rejected (G3).
type ssoFlowState struct {
	ConnID         string `json:"conn_id"`
	OrgID          string `json:"org_id"`
	OrgSlug        string `json:"org_slug"`
	ExpectedIssuer string `json:"issuer"`
	TokenEndpoint  string `json:"token_endpoint"`
	JWKSURI        string `json:"jwks_uri"`
	CodeVerifier   string `json:"code_verifier"`
	Nonce          string `json:"nonce"`
	CallbackURI    string `json:"callback_uri"`
	AppRedirect    string `json:"app_redirect"`
	AppState       string `json:"app_state"`
}

// oidcDiscovery is the subset of the upstream discovery document we consume.
type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// SSOLoginHandler starts a per-org OIDC broker login: it resolves the org's
// sso_oidc connection, discovers the upstream endpoints, and redirects the
// browser to the upstream /authorize with PKCE + state + nonce.
func (h *httpProvider) SSOLoginHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SSOLoginHandler").Logger()
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("org_slug"))
		appRedirect := strings.TrimSpace(c.Query("redirect_uri"))
		appState := strings.TrimSpace(c.Query("state"))
		hostname := parsers.GetHost(c)

		if appRedirect == "" || !validators.IsValidRedirectURI(appRedirect, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("invalid or missing redirect_uri")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "invalid redirect_uri"})
			return
		}
		if h.MemoryStoreProvider == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		conn, ok := h.resolveActiveOIDCConnection(c, slug, &log)
		if !ok {
			return
		}

		disc, err := h.fetchOIDCDiscovery(c.Request.Context(), conn.IssuerURL)
		if err != nil {
			log.Debug().Err(err).Msg("failed to fetch upstream OIDC discovery")
			c.JSON(http.StatusBadGateway, gin.H{"error": "sso_upstream_error", "error_description": "could not reach the identity provider"})
			return
		}

		verifier, err := randURLString(64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		state, err := randURLString(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		nonce, err := randURLString(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		challenge := pkceS256(verifier)

		callbackURI := strings.TrimSpace(conn.SSORedirectURI)
		if callbackURI == "" {
			callbackURI = strings.TrimRight(hostname, "/") + "/oauth/sso/" + url.PathEscape(slug) + "/callback"
		}

		flow := ssoFlowState{
			ConnID:         conn.ID,
			OrgID:          conn.OrgID,
			OrgSlug:        slug,
			ExpectedIssuer: disc.Issuer,
			TokenEndpoint:  disc.TokenEndpoint,
			JWKSURI:        disc.JWKSURI,
			CodeVerifier:   verifier,
			Nonce:          nonce,
			CallbackURI:    callbackURI,
			AppRedirect:    appRedirect,
			AppState:       appState,
		}
		flowJSON, err := json.Marshal(flow)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if err := h.MemoryStoreProvider.SetState(ssoFlowPrefix+state, string(flowJSON)); err != nil {
			log.Debug().Err(err).Msg("failed to persist sso flow state")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		scopes := strings.TrimSpace(conn.SSOScopes)
		if scopes == "" {
			scopes = "openid profile email"
		}
		q := url.Values{}
		q.Set("response_type", "code")
		q.Set("client_id", conn.SSOClientID)
		q.Set("redirect_uri", callbackURI)
		q.Set("scope", scopes)
		q.Set("state", state)
		q.Set("nonce", nonce)
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")
		sep := "?"
		if strings.Contains(disc.AuthorizationEndpoint, "?") {
			sep = "&"
		}
		authURL := disc.AuthorizationEndpoint + sep + q.Encode()

		metrics.RecordAuthEvent(metrics.EventOAuthLogin, metrics.StatusSuccess)
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditSSOLoginInitiatedEvent,
			ActorType:    constants.AuditActorTypeUser,
			ResourceType: constants.AuditResourceTypeSession,
			Metadata:     slug,
			IPAddress:    utils.GetIP(c.Request),
			UserAgent:    utils.GetUserAgent(c.Request),
		})
		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

// SSOCallbackHandler completes the broker flow: single-use state lookup, code
// exchange, ID-token verification (incl. mix-up defense), JIT provisioning, and
// Authorizer session issuance.
func (h *httpProvider) SSOCallbackHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SSOCallbackHandler").Logger()
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		slug := strings.TrimSpace(c.Param("org_slug"))
		state := strings.TrimSpace(c.Query("state"))
		if state == "" || h.MemoryStoreProvider == nil {
			ssoFail(c, &log, "invalid_state", "missing or invalid state")
			return
		}
		// Single-use: atomically fetch and delete. A replayed callback finds nothing.
		raw, err := h.MemoryStoreProvider.GetAndRemoveState(ssoFlowPrefix + state)
		if err != nil && err != goredis.Nil {
			log.Debug().Err(err).Msg("failed to read sso flow state")
		}
		if strings.TrimSpace(raw) == "" {
			ssoFail(c, &log, "invalid_state", "unknown or expired state")
			return
		}
		var flow ssoFlowState
		if err := json.Unmarshal([]byte(raw), &flow); err != nil {
			ssoFail(c, &log, "invalid_state", "corrupt state")
			return
		}
		// Bind the callback route to the flow that dispatched it.
		if flow.OrgSlug != slug {
			ssoFail(c, &log, "invalid_state", "state/route mismatch")
			return
		}
		// Mix-up defense (RFC 9207): if the IdP echoed an `iss`, it MUST match the
		// issuer the dispatching connection discovered.
		if issParam := strings.TrimSpace(c.Query("iss")); issParam != "" && issParam != flow.ExpectedIssuer {
			metrics.RecordSecurityEvent("sso_mixup_iss_mismatch", slug)
			ssoFail(c, &log, "invalid_issuer", "issuer mismatch")
			return
		}
		if upstreamErr := strings.TrimSpace(c.Query("error")); upstreamErr != "" {
			log.Debug().Str("upstream_error", upstreamErr).Msg("upstream idp returned an error")
			ssoFail(c, &log, "sso_upstream_error", "identity provider returned an error")
			return
		}
		code := strings.TrimSpace(c.Query("code"))
		if code == "" {
			ssoFail(c, &log, "invalid_request", "missing code")
			return
		}

		conn, err := h.StorageProvider.GetTrustedIssuerByID(ctx, flow.ConnID)
		if err != nil || conn == nil || conn.EffectiveKind() != constants.TrustKindSSOOIDC || !conn.IsActive {
			ssoFail(c, &log, "sso_not_configured", "connection unavailable")
			return
		}
		secret, err := crypto.DecryptAES(h.Config.ClientSecret, conn.SSOClientSecretEnc)
		if err != nil {
			log.Debug().Err(err).Msg("failed to decrypt upstream client secret")
			ssoFail(c, &log, "sso_config_error", "connection misconfigured")
			return
		}

		idToken, err := h.exchangeSSOCode(ctx, flow, conn.SSOClientID, secret, code)
		if err != nil {
			log.Debug().Err(err).Msg("upstream code exchange failed")
			ssoFail(c, &log, "sso_exchange_failed", "code exchange failed")
			return
		}
		claims, err := h.verifySSOIDToken(ctx, &flow, conn, idToken)
		if err != nil {
			log.Debug().Err(err).Msg("upstream id_token verification failed")
			metrics.RecordSecurityEvent("sso_id_token_invalid", slug)
			ssoFail(c, &log, "sso_id_token_invalid", "id token verification failed")
			return
		}

		user, isSignUp, err := h.jitProvisionSSOUser(ctx, &flow, claims)
		if err != nil {
			log.Debug().Err(err).Msg("sso JIT provisioning rejected")
			ssoFail(c, &log, "sso_provisioning_failed", err.Error())
			return
		}

		if err := h.issueSSOSession(c, &flow, user, isSignUp); err != nil {
			log.Debug().Err(err).Msg("failed to issue session")
			ssoFail(c, &log, "sso_session_failed", "could not establish session")
			return
		}
	}
}

// resolveActiveOIDCConnection looks up the org by slug and its active sso_oidc
// connection, writing an error response and returning ok=false on any failure.
func (h *httpProvider) resolveActiveOIDCConnection(c *gin.Context, slug string, log *zerolog.Logger) (*schemas.TrustedIssuer, bool) {
	ctx := c.Request.Context()
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "missing organization"})
		return nil, false
	}
	org, err := h.StorageProvider.GetOrganizationByName(ctx, slug)
	if err != nil || org == nil {
		log.Debug().Err(err).Str("org", slug).Msg("organization not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "sso_not_configured", "error_description": "unknown organization"})
		return nil, false
	}
	if !org.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "sso_not_configured", "error_description": "organization disabled"})
		return nil, false
	}
	conn, err := h.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, org.ID, constants.TrustKindSSOOIDC)
	if err != nil || conn == nil || !conn.IsActive {
		log.Debug().Err(err).Str("org", slug).Msg("no active OIDC connection for org")
		c.JSON(http.StatusNotFound, gin.H{"error": "sso_not_configured", "error_description": "SSO is not configured for this organization"})
		return nil, false
	}
	return conn, true
}

// fetchOIDCDiscovery SSRF-hardened fetches the upstream OpenID configuration.
func (h *httpProvider) fetchOIDCDiscovery(ctx context.Context, issuerURL string) (*oidcDiscovery, error) {
	discoveryURL := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	body, err := ssoSafeGet(ctx, discoveryURL)
	if err != nil {
		return nil, err
	}
	var disc oidcDiscovery
	if err := json.Unmarshal(body, &disc); err != nil {
		return nil, fmt.Errorf("malformed discovery document: %w", err)
	}
	if disc.Issuer == "" || disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" || disc.JWKSURI == "" {
		return nil, fmt.Errorf("discovery document missing required fields")
	}
	return &disc, nil
}

// exchangeSSOCode exchanges the authorization code at the upstream token endpoint
// (SSRF-hardened POST) and returns the raw id_token string.
func (h *httpProvider) exchangeSSOCode(ctx context.Context, flow ssoFlowState, clientID, clientSecret, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", flow.CallbackURI)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code_verifier", flow.CodeVerifier)

	client, err := validators.SafeHTTPClient(ctx, flow.TokenEndpoint, ssoHTTPTimeout)
	if err != nil {
		return "", err
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, flow.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, ssoMaxRespBytes))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}
	var tr struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("malformed token response: %w", err)
	}
	if strings.TrimSpace(tr.IDToken) == "" {
		return "", fmt.Errorf("token response has no id_token")
	}
	return tr.IDToken, nil
}

// verifySSOIDToken fetches the upstream JWKS (SSRF-hardened) then verifies the
// ID token via verifyIDTokenAgainstJWKS.
func (h *httpProvider) verifySSOIDToken(ctx context.Context, flow *ssoFlowState, conn *schemas.TrustedIssuer, rawIDToken string) (jwt.MapClaims, error) {
	jwks, err := h.fetchSSOJWKS(ctx, flow.JWKSURI)
	if err != nil {
		return nil, err
	}
	return verifyIDTokenAgainstJWKS(flow, conn, rawIDToken, jwks)
}

// verifyIDTokenAgainstJWKS is the pure (no-network) ID-token verifier: asymmetric
// signature against jwks, iss == the flow's expected issuer (mix-up defense),
// aud contains our client_id, nonce bound, exp valid, asymmetric alg only. Kept
// separate from the network fetch so it can be unit-tested exhaustively.
func verifyIDTokenAgainstJWKS(flow *ssoFlowState, conn *schemas.TrustedIssuer, rawIDToken string, jwks *jose.JSONWebKeySet) (jwt.MapClaims, error) {
	// Enforce the alg allow-list on the header before verifying (defence in depth).
	unverified := jwt.MapClaims{}
	parsed, _, err := jwt.NewParser().ParseUnverified(rawIDToken, unverified)
	if err != nil {
		return nil, fmt.Errorf("unparseable id_token: %w", err)
	}
	headerAlg, _ := parsed.Header["alg"].(string)
	if !ssoIsAllowedAlg(headerAlg) {
		return nil, fmt.Errorf("disallowed id_token alg %q", headerAlg)
	}

	keyfunc := func(t *jwt.Token) (interface{}, error) {
		if !ssoIsAllowedAlg(t.Method.Alg()) {
			return nil, fmt.Errorf("disallowed alg %q", t.Method.Alg())
		}
		kid, _ := t.Header["kid"].(string)
		if kid != "" {
			keys := jwks.Key(kid)
			if len(keys) == 0 {
				return nil, fmt.Errorf("no JWKS key for kid %q", kid)
			}
			return keys[0].Key, nil
		}
		if len(jwks.Keys) != 1 {
			return nil, fmt.Errorf("id_token has no kid and JWKS is not single-key")
		}
		return jwks.Keys[0].Key, nil
	}

	claims := jwt.MapClaims{}
	if _, err := jwt.NewParser(jwt.WithValidMethods(ssoAllowedAlgs), jwt.WithoutClaimsValidation()).ParseWithClaims(rawIDToken, claims, keyfunc); err != nil {
		return nil, fmt.Errorf("signature validation failed: %w", err)
	}

	// iss MUST equal the issuer discovered by the dispatching connection (G3).
	iss, _ := claims["iss"].(string)
	if strings.TrimSpace(iss) == "" || iss != flow.ExpectedIssuer {
		return nil, fmt.Errorf("id_token iss does not match the connection issuer")
	}
	// aud MUST contain our upstream client_id.
	if !ssoAudienceContains(claims["aud"], conn.SSOClientID) {
		return nil, fmt.Errorf("id_token aud does not contain our client_id")
	}
	// OIDC Core §3.1.3.7 step 4: when aud is multi-valued, azp MUST be present and
	// equal our client_id (defends against a token minted for another RP that also
	// lists us in aud).
	if ssoAudMultiValued(claims["aud"]) {
		if azp, _ := claims["azp"].(string); azp != conn.SSOClientID {
			return nil, fmt.Errorf("id_token azp missing or mismatched for a multi-audience token")
		}
	}
	// nonce MUST match the one we sent.
	if n, _ := claims["nonce"].(string); n != flow.Nonce {
		return nil, fmt.Errorf("id_token nonce mismatch")
	}
	// exp present and valid (with skew); iat not absurdly in the future.
	now := time.Now()
	exp, ok := ssoClaimInt64(claims, "exp")
	if !ok || now.Unix() > exp+int64(ssoClockSkew.Seconds()) {
		return nil, fmt.Errorf("id_token is expired or missing exp")
	}
	if iat, ok := ssoClaimInt64(claims, "iat"); ok && iat > now.Unix()+int64(ssoClockSkew.Seconds()) {
		return nil, fmt.Errorf("id_token iat is in the future")
	}
	return claims, nil
}

// fetchSSOJWKS SSRF-hardened fetches and parses the upstream JWKS.
func (h *httpProvider) fetchSSOJWKS(ctx context.Context, jwksURI string) (*jose.JSONWebKeySet, error) {
	body, err := ssoSafeGet(ctx, jwksURI)
	if err != nil {
		return nil, err
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("malformed JWKS: %w", err)
	}
	if len(set.Keys) == 0 {
		return nil, fmt.Errorf("JWKS contains no keys")
	}
	return &set, nil
}

// jitProvisionSSOUser maps ID-token claims to an Authorizer user, namespaced by
// (org_id, issuer, sub). Returns (user, isSignUp, error).
//
// SECURITY (account-takeover): a returning federated principal is found ONLY via
// the (org_id, issuer, sub) FederatedIdentity row. A first-time principal whose
// email collides with ANY existing account is rejected fail-closed — it is never
// silently linked to that account.
func (h *httpProvider) jitProvisionSSOUser(ctx context.Context, flow *ssoFlowState, claims jwt.MapClaims) (*schemas.User, bool, error) {
	sub, _ := claims["sub"].(string)
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return nil, false, fmt.Errorf("missing subject")
	}

	// Returning principal?
	if fi, err := h.StorageProvider.GetFederatedIdentity(ctx, flow.OrgID, flow.ExpectedIssuer, sub); err == nil && fi != nil {
		user, err := h.StorageProvider.GetUserByID(ctx, fi.UserID)
		if err != nil || user == nil {
			return nil, false, fmt.Errorf("federated identity references an unknown user")
		}
		if user.RevokedTimestamp != nil {
			return nil, false, fmt.Errorf("user access has been revoked")
		}
		return user, false, nil
	}

	if !h.Config.EnableSignup {
		return nil, false, fmt.Errorf("signup is disabled for this instance")
	}

	email := strings.TrimSpace(claimString(claims, "email"))
	// Account-takeover defense: never link to an existing account by email.
	if email != "" {
		if existing, err := h.StorageProvider.GetUserByEmail(ctx, email); err == nil && existing != nil {
			return nil, false, fmt.Errorf("an account with this email already exists")
		}
	}

	now := time.Now().Unix()
	user := &schemas.User{
		SignupMethods: constants.AuthRecipeMethodSSO,
		Roles:         strings.Join(h.Config.DefaultRoles, ","),
	}
	if email != "" {
		user.Email = refs.NewStringRef(email)
		if emailVerifiedClaim(claims) {
			user.EmailVerifiedAt = &now
		}
	}
	if v := claimString(claims, "given_name"); v != "" {
		user.GivenName = refs.NewStringRef(v)
	}
	if v := claimString(claims, "family_name"); v != "" {
		user.FamilyName = refs.NewStringRef(v)
	}
	if v := claimString(claims, "name"); v != "" {
		user.Nickname = refs.NewStringRef(v)
	}
	if v := claimString(claims, "picture"); v != "" {
		user.Picture = refs.NewStringRef(v)
	}

	user, err := h.StorageProvider.AddUser(ctx, user)
	if err != nil {
		return nil, false, fmt.Errorf("failed to provision user")
	}
	if _, err := h.StorageProvider.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
		OrgID:   flow.OrgID,
		Issuer:  flow.ExpectedIssuer,
		Subject: sub,
		UserID:  user.ID,
	}); err != nil {
		// Compensating action (LOW-1): the user we just created now has an email on
		// record but NO federated-identity mapping. Left in place, the next login
		// would hit the email-collision guard and lock this principal out forever,
		// and the orphan would pollute email lookups. Delete it so a retry is clean.
		if delErr := h.StorageProvider.DeleteUser(ctx, user); delErr != nil {
			h.Log.Debug().Err(delErr).Msg("failed to delete orphaned user after federated-identity insert failure")
		}
		return nil, false, fmt.Errorf("failed to record federated identity")
	}
	// Best-effort org membership: the (org_id, user_id) uniqueness guard tolerates
	// a pre-existing row. A failure here must not block the login.
	if _, err := h.StorageProvider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  flow.OrgID,
		UserID: user.ID,
	}); err != nil {
		h.Log.Debug().Err(err).Msg("failed to add org membership (non-fatal)")
	}
	return user, true, nil
}

// issueSSOSession mints the Authorizer session/tokens, sets the session cookie,
// records the session, and redirects to the app's redirect_uri.
func (h *httpProvider) issueSSOSession(c *gin.Context, flow *ssoFlowState, user *schemas.User, isSignUp bool) error {
	hostname := parsers.GetHost(c)
	roles := splitRoles(user.Roles)
	authToken, err := h.TokenProvider.CreateAuthToken(c, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: constants.AuthRecipeMethodSSO,
		Nonce:       flow.Nonce,
		HostName:    hostname,
	})
	if err != nil {
		return err
	}

	sessionKey := constants.AuthRecipeMethodSSO + ":" + user.ID
	cookie.SetSession(c, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
	_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)
	if authToken.RefreshToken != nil {
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	bgCtx := context.WithoutCancel(c.Request.Context())
	userAgent := utils.GetUserAgent(c.Request)
	ip := utils.GetIP(c.Request)
	go func() {
		if isSignUp {
			_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodSSO, user)
		}
		_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodSSO, user)
		if err := h.StorageProvider.AddSession(bgCtx, &schemas.Session{UserID: user.ID, UserAgent: userAgent, IP: ip}); err != nil {
			h.Log.Debug().Err(err).Msg("failed to add session")
		}
	}()

	params := "state=" + url.QueryEscape(flow.AppState)
	redirectURL := flow.AppRedirect
	if strings.Contains(redirectURL, "?") {
		redirectURL = redirectURL + "&" + params
	} else {
		redirectURL = redirectURL + "?" + params
	}
	metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusSuccess)
	metrics.ActiveSessions.Inc()
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditSSOCallbackSuccessEvent,
		ActorID:      user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   user.ID,
		Metadata:     flow.OrgSlug,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
	c.Redirect(http.StatusFound, redirectURL)
	return nil
}

// ssoFail writes a uniform OAuth-style error response and records the failure.
func ssoFail(c *gin.Context, log *zerolog.Logger, code, desc string) {
	metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
	log.Debug().Str("error", code).Msg("sso callback failed")
	c.JSON(http.StatusBadRequest, gin.H{"error": code, "error_description": desc})
}

// ssoSafeGet performs an SSRF-hardened GET (host-pinned, redirects refused,
// size-capped) against an issuer-controlled URL.
func ssoSafeGet(ctx context.Context, rawURL string) ([]byte, error) {
	client, err := validators.SafeHTTPClient(ctx, rawURL, ssoHTTPTimeout)
	if err != nil {
		return nil, err
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(io.LimitReader(resp.Body, ssoMaxRespBytes))
}

// randURLString returns n bytes of crypto-random data, base64url-encoded.
func randURLString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// pkceS256 derives the RFC 7636 S256 code challenge from a verifier.
func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func ssoIsAllowedAlg(alg string) bool {
	for _, a := range ssoAllowedAlgs {
		if a == alg {
			return true
		}
	}
	return false
}

func ssoAudienceContains(aud interface{}, expected string) bool {
	switch v := aud.(type) {
	case string:
		return v == expected
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == expected {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == expected {
				return true
			}
		}
	}
	return false
}

// ssoAudMultiValued reports whether the aud claim carries more than one audience.
func ssoAudMultiValued(aud interface{}) bool {
	switch v := aud.(type) {
	case []interface{}:
		return len(v) > 1
	case []string:
		return len(v) > 1
	}
	return false
}

func ssoClaimInt64(claims jwt.MapClaims, key string) (int64, bool) {
	switch v := claims[key].(type) {
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case int64:
		return v, true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	}
	return 0, false
}

func claimString(claims jwt.MapClaims, key string) string {
	s, _ := claims[key].(string)
	return s
}

// splitRoles parses a comma-separated role string, trimming and dropping empties.
func splitRoles(roles string) []string {
	out := []string{}
	for _, r := range strings.Split(roles, ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// emailVerifiedClaim reads the OIDC email_verified claim (bool or "true").
func emailVerifiedClaim(claims jwt.MapClaims) bool {
	switch v := claims["email_verified"].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	}
	return false
}
