package clientauth

// RFC 7523 client_assertion (JWT-bearer) client authentication — the secretless
// workload-identity path for Kubernetes projected ServiceAccount tokens and
// other trusted-issuer JWTs. All validation is fail-closed; every rejection
// returns the generic ErrInvalidClient so no check leaks which one failed.
//
// SECURITY MODEL (design §4.3 / §5.2):
//   - Algorithm allow-list, asymmetric only; alg:none and HS* rejected (RFC 8725 §3.1, G8).
//   - Issuer resolved by the token's `iss` against a registered TrustedIssuer.
//   - JWKS fetched SSRF-hardened, cached in the shared store keyed by the trust
//     row's identity (id), not the issuer URL alone (H7).
//   - `aud` MUST contain the issuer's ExpectedAud exactly (a generic aud is rejected).
//   - Subject pin (C1): the SubjectClaim value MUST be in AllowedSubjects (exact).
//     Empty AllowedSubjects is DENY-ALL.
//   - Replay (C2/H4): single-use `jti`, or (iss,sub,iat,exp) when `jti` is absent,
//     held in the shared store until the token's exp.
//   - Lifetime (H4): exp − iat MUST be ≤ a short ceiling.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/validators"
)

const (
	// defaultMaxClientAssertionLifetime bounds exp − iat (§5.2 H4). RFC 7523 §3
	// recommends short-lived, single-use assertions; this default matches the
	// Kubernetes projected-SA-token default lifetime (3600s) so workloads that
	// mint a fresh per-request token are accepted while abnormally long-lived
	// assertions (which would widen the replay window) are rejected.
	// ponytail: constant, not a CLI flag yet — raise it here if a deployment mints
	// longer projected tokens; wire to config when an operator actually needs it.
	defaultMaxClientAssertionLifetime = time.Hour

	// jwksCacheTTLSeconds bounds how long a fetched JWKS is trusted before a
	// re-fetch. Short enough to pick up an issuer key rotation, long enough to
	// keep the token endpoint off the network on the hot path.
	jwksCacheTTLSeconds = 600

	// assertionClockSkew tolerates minor clock drift on exp/nbf comparisons.
	assertionClockSkew = 60 * time.Second

	// httpFetchTimeout caps a single JWKS / discovery fetch.
	httpFetchTimeout = 10 * time.Second

	// maxJWKSBytes caps the response body read from an issuer-controlled endpoint.
	maxJWKSBytes = 1 << 20 // 1 MiB

	// tokenReviewPath is the Kubernetes TokenReview subresource (authentication.k8s.io/v1).
	tokenReviewPath = "/apis/authentication.k8s.io/v1/tokenreviews"

	// inClusterSATokenPath is the standard mount for Authorizer's OWN projected
	// ServiceAccount token. TokenReview requires the caller (Authorizer) to
	// authenticate to the apiserver with a token that carries the
	// system:auth-delegator RBAC; when running in-cluster this file is present.
	// Read best-effort — absent means the Authorization header is omitted.
	inClusterSATokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// allowedAssertionAlgs is the asymmetric signature allow-list (RFC 8725 §3.1).
// alg:none and every symmetric (HS*) algorithm are absent by construction — a
// symmetric alg would let anyone who can read the JWKS forge a token.
var allowedAssertionAlgs = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
}

// resolveViaClientAssertion authenticates a client by verifying a client
// assertion against a registered TrustedIssuer. Two assertion types are handled:
//
//   - jwt-bearer (RFC 7523): a generic trusted-issuer JWT (K8s SA token, cloud
//     OIDC token). The trust row is a non-SPIFFE issuer_type.
//   - jwt-spiffe (draft-ietf-oauth-spiffe-client-auth): a SPIFFE JWT-SVID whose
//     `sub` is a SPIFFE ID (spiffe://…). The trust row is issuer_type spiffe_jwt.
//     The token's `iss` is the SPIRE server, NOT the subject — iss ≠ sub is
//     expected and correct, which is exactly why this is a distinct type: `iss`
//     locates the trust row and its bundle, while `sub` (the SPIFFE ID) is pinned
//     against AllowedSubjects. No private_key_jwt iss==sub==client_id assumption
//     is (or ever was) applied on this path.
//
// On success it returns the active Client the trust row is bound to. All
// validation is fail-closed; every rejection returns the generic ErrInvalidClient.
func (p *provider) resolveViaClientAssertion(ctx context.Context, params ResolveParams) (*schemas.Client, error) {
	log := p.Log.With().Str("func", "resolveViaClientAssertion").Logger()

	// Select the assertion profile. A missing/unknown type is a malformed request
	// (invalid_request); spiffe toggles the SPIFFE-specific checks below.
	var spiffe bool
	switch params.ClientAssertionType {
	case constants.ClientAssertionTypeJWTBearer:
		spiffe = false
	case constants.ClientAssertionTypeJWTSPIFFE:
		spiffe = true
	default:
		log.Debug().Str("client_assertion_type", params.ClientAssertionType).Msg("unsupported client_assertion_type")
		return nil, ErrUnsupportedAssertionType
	}

	// Replay protection is mandatory for this path; without the shared store we
	// cannot guarantee single-use, so fail closed rather than accept.
	if p.MemoryStoreProvider == nil {
		log.Debug().Msg("memory store unavailable; cannot enforce assertion replay protection")
		return nil, ErrInvalidClient
	}

	// 1. Parse WITHOUT verifying to read `iss` and the header alg — we need the
	//    issuer to locate the trust row (and its keys) before we can verify.
	unverifiedClaims := jwt.MapClaims{}
	parsed, _, err := jwt.NewParser().ParseUnverified(params.ClientAssertion, unverifiedClaims)
	if err != nil {
		log.Debug().Err(err).Msg("client_assertion is not a parseable JWT")
		return nil, ErrInvalidClient
	}

	// 2. Enforce the algorithm allow-list up front (defence in depth; the verified
	//    parse below re-enforces it via WithValidMethods). Rejects alg:none, HS*.
	headerAlg, _ := parsed.Header["alg"].(string)
	if !isAllowedAlg(headerAlg) {
		log.Debug().Str("alg", headerAlg).Msg("client_assertion uses a disallowed algorithm")
		return nil, ErrInvalidClient
	}

	iss, _ := unverifiedClaims["iss"].(string)
	iss = strings.TrimSpace(iss)
	if iss == "" {
		log.Debug().Msg("client_assertion missing iss")
		return nil, ErrInvalidClient
	}

	// 3. Resolve the trust row by iss. Unknown or inactive issuer → invalid_client.
	issuer, err := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, iss)
	if err != nil || issuer == nil {
		log.Debug().Err(err).Str("iss", iss).Msg("no trusted issuer for iss")
		return nil, ErrInvalidClient
	}
	// SECURITY (design §5.2 CR1 — kind confusion): the unified trusted_issuers
	// table also holds per-org SSO connections (kind=sso_oidc/sso_saml), which have
	// NO subject pin and federate end users, not clients. Only a
	// client_assertion_trust row with an empty OrgID may authenticate an OAuth
	// client. issuer_url is globally unique so this lookup returns at most one row;
	// reject it here if it is anything other than an instance-global
	// client_assertion_trust row. A pre-existing row (empty Kind) is treated as
	// client_assertion_trust by EffectiveKind, so upgrades are unaffected.
	if issuer.EffectiveKind() != constants.TrustKindClientAssertion || strings.TrimSpace(issuer.OrgID) != "" {
		log.Debug().Str("iss", iss).Str("kind", issuer.EffectiveKind()).Msg("trusted issuer is not a client_assertion trust row")
		return nil, ErrInvalidClient
	}
	if !issuer.IsActive {
		log.Debug().Str("iss", iss).Msg("trusted issuer is inactive")
		return nil, ErrInvalidClient
	}

	// SECURITY: the assertion type MUST match the trust row's issuer_type. A
	// spiffe_jwt row (whose AllowedSubjects hold spiffe:// IDs and whose key set is
	// a SPIFFE trust bundle) is reachable ONLY via the jwt-spiffe type; a generic
	// jwt-bearer row is reachable ONLY via jwt-bearer. Enforcing the match both
	// ways stops a caller from routing a SPIFFE row through the bearer profile
	// (which would skip the spiffe:// subject-format check) or vice versa.
	rowIsSpiffe := issuer.IssuerType == constants.IssuerTypeSPIFFEJWT
	if spiffe != rowIsSpiffe {
		log.Debug().Str("iss", iss).Str("issuer_type", issuer.IssuerType).Bool("assertion_is_spiffe", spiffe).Msg("client_assertion type does not match trusted issuer type")
		return nil, ErrInvalidClient
	}

	// 4. Fetch the issuer's JWKS (cached in the shared store, keyed by the trust
	//    row identity — H7). 5. Verify the signature against it.
	keyfunc, err := p.assertionKeyfunc(ctx, issuer)
	if err != nil {
		log.Debug().Err(err).Msg("failed to build JWKS keyfunc for issuer")
		return nil, ErrInvalidClient
	}
	// WithoutClaimsValidation: the library's default time checks use a zero clock
	// skew and would reject a token whose iat is even a second ahead of us (issuer
	// clock drift is normal in federation). We do all time-claim validation below
	// with an explicit skew, so here the parser only verifies the signature and
	// re-enforces the algorithm allow-list.
	claims := jwt.MapClaims{}
	if _, err := jwt.NewParser(jwt.WithValidMethods(allowedAssertionAlgs), jwt.WithoutClaimsValidation()).ParseWithClaims(params.ClientAssertion, claims, keyfunc); err != nil {
		log.Debug().Err(err).Msg("client_assertion signature validation failed")
		return nil, ErrInvalidClient
	}

	// 6. Claim checks: exp/iat present, not expired, nbf honoured, exp−iat ≤ ceiling.
	now := time.Now()
	exp, ok := claimInt64(claims, "exp")
	if !ok {
		log.Debug().Msg("client_assertion missing exp")
		return nil, ErrInvalidClient
	}
	iat, ok := claimInt64(claims, "iat")
	if !ok {
		log.Debug().Msg("client_assertion missing iat")
		return nil, ErrInvalidClient
	}
	if now.Unix() > exp+int64(assertionClockSkew.Seconds()) {
		log.Debug().Msg("client_assertion is expired")
		return nil, ErrInvalidClient
	}
	// Reject a clearly bogus future issuance (beyond tolerated skew) while still
	// accepting the normal issuer-clock-ahead-of-us case.
	if iat > now.Unix()+int64(assertionClockSkew.Seconds()) {
		log.Debug().Msg("client_assertion iat is in the future")
		return nil, ErrInvalidClient
	}
	if nbf, ok := claimInt64(claims, "nbf"); ok {
		if now.Unix() < nbf-int64(assertionClockSkew.Seconds()) {
			log.Debug().Msg("client_assertion is not yet valid (nbf)")
			return nil, ErrInvalidClient
		}
	}
	// Measure the token's declared lifetime (exp − iat), NOT exp − now: a token
	// minted with an abnormally long life is rejected regardless of when presented.
	if exp-iat > int64(p.maxAssertionLifetime.Seconds()) {
		log.Debug().Int64("exp_minus_iat", exp-iat).Msg("client_assertion lifetime exceeds the ceiling")
		return nil, ErrInvalidClient
	}

	// aud MUST contain the issuer's ExpectedAud exactly — which the admin sets to
	// this Authorizer's token endpoint, so a token minted for another audience
	// cannot be replayed here.
	if strings.TrimSpace(issuer.ExpectedAud) == "" || !audienceContains(claims["aud"], issuer.ExpectedAud) {
		log.Debug().Msg("client_assertion aud does not match the issuer's expected_aud")
		return nil, ErrInvalidClient
	}

	// 7. Subject pin (C1): the SubjectClaim value must be an exact member of the
	//    row's AllowedSubjects. Empty AllowedSubjects is DENY-ALL.
	subjectClaim := issuer.SubjectClaim
	if subjectClaim == "" {
		subjectClaim = "sub"
	}
	subject, _ := claims[subjectClaim].(string)
	subject = strings.TrimSpace(subject)
	// SPIFFE (draft-ietf-oauth-spiffe-client-auth): the subject MUST be a SPIFFE ID
	// (a spiffe:// URI). AllowedSubjects on a spiffe_jwt row hold spiffe:// IDs, so
	// requiring the scheme here rejects a non-SPIFFE subject before the exact-match
	// and keeps the allow-list semantics unambiguous.
	if spiffe && !strings.HasPrefix(subject, "spiffe://") {
		log.Debug().Str("subject_claim", subjectClaim).Msg("spiffe client_assertion subject is not a SPIFFE ID")
		return nil, ErrInvalidClient
	}
	allowed := issuer.ParsedAllowedSubjects()
	if subject == "" || !contains(allowed, subject) {
		log.Debug().Str("subject_claim", subjectClaim).Msg("client_assertion subject not in the allow-list")
		return nil, ErrInvalidClient
	}

	// 8. Replay (C2/H4): single-use per issuer. Prefer jti; fall back to a hash of
	//    (iss,sub,iat,exp) when the token carries no jti (K8s SA tokens have none).
	//    Held until the token's exp so a captured token cannot be re-presented.
	replayKey := assertionReplayKey(issuer.ID, claims, iss, subject, iat, exp)
	if seen, _ := p.MemoryStoreProvider.GetCache(replayKey); seen != "" {
		log.Debug().Msg("client_assertion replay detected")
		return nil, ErrInvalidClient
	}

	// 8b. Kubernetes TokenReview (S7): offline JWKS validation proves the token was
	//     signed by the cluster, but NOT that the bound Pod/ServiceAccount still
	//     exists — a deleted pod's not-yet-expired token would otherwise pass. When
	//     the trust row opts in, call the cluster's TokenReview API to confirm the
	//     presented token is still authenticated online, and fail closed otherwise.
	//     Done BEFORE the replay marker is set so a transient apiserver failure does
	//     not permanently burn a legitimate token (the token is retryable), and
	//     AFTER the replay CHECK so a replay is rejected without any apiserver call.
	if issuer.EnableTokenReview {
		apiServerURL := ""
		if issuer.KubernetesAPIServerURL != nil {
			apiServerURL = strings.TrimSpace(*issuer.KubernetesAPIServerURL)
		}
		if apiServerURL == "" {
			log.Debug().Msg("enable_token_review is set but kubernetes_api_server_url is empty")
			return nil, ErrInvalidClient
		}
		if err := p.performTokenReview(ctx, apiServerURL, params.ClientAssertion, issuer.ExpectedAud); err != nil {
			log.Debug().Err(err).Msg("kubernetes TokenReview rejected the client_assertion")
			return nil, ErrInvalidClient
		}
	}

	replayTTL := exp - now.Unix()
	if replayTTL < 1 {
		replayTTL = 1
	}
	// ponytail: best-effort check-then-set (memory_store has no atomic SetNX);
	// a sub-second cross-instance race could let two simultaneous replays through.
	// Acceptable given the short window; upgrade to SetNX if it ever matters.
	if err := p.MemoryStoreProvider.SetCache(replayKey, "1", replayTTL); err != nil {
		log.Debug().Err(err).Msg("failed to persist assertion replay marker")
		return nil, ErrInvalidClient
	}

	// 9. Resolve the Client the trust row authenticates (stored as the surrogate
	//    PK). Must exist and be active.
	client, err := p.StorageProvider.GetClientByID(ctx, issuer.ClientID)
	if err != nil || client == nil {
		log.Debug().Err(err).Msg("trusted issuer references an unknown client")
		return nil, ErrInvalidClient
	}
	if params.RequireServiceAccountKind && client.Kind != constants.ClientKindServiceAccount {
		log.Debug().Str("kind", client.Kind).Msg("client not authorized for this grant")
		return client, ErrUnauthorizedClient
	}
	if !client.IsActive {
		log.Debug().Msg("client is inactive")
		return client, ErrInvalidClient
	}
	return client, nil
}

// assertionKeyfunc returns a golang-jwt keyfunc backed by the issuer's JWKS.
// The JWKS is served from the shared cache (keyed by the trust row id) and
// fetched SSRF-hardened on a miss.
func (p *provider) assertionKeyfunc(ctx context.Context, issuer *schemas.TrustedIssuer) (jwt.Keyfunc, error) {
	jwks, err := p.loadJWKS(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return func(t *jwt.Token) (interface{}, error) {
		// Re-assert the alg allow-list at verification time.
		if !isAllowedAlg(t.Method.Alg()) {
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
		// No kid: only safe when the set has exactly one key.
		if len(jwks.Keys) != 1 {
			return nil, fmt.Errorf("client_assertion has no kid and the JWKS is not single-key")
		}
		return jwks.Keys[0].Key, nil
	}, nil
}

// loadJWKS returns the issuer's JWKS, using the shared cache keyed by the trust
// row identity (H7) so a cached key set can never be attributed to a different
// row that merely shares an issuer URL.
func (p *provider) loadJWKS(ctx context.Context, issuer *schemas.TrustedIssuer) (*jose.JSONWebKeySet, error) {
	cacheKey := "jwks_cache:" + issuer.ID
	if p.MemoryStoreProvider != nil {
		if cached, _ := p.MemoryStoreProvider.GetCache(cacheKey); cached != "" {
			var set jose.JSONWebKeySet
			if err := json.Unmarshal([]byte(cached), &set); err == nil && len(set.Keys) > 0 {
				return &set, nil
			}
		}
	}

	raw, err := p.fetchJWKSBytes(ctx, issuer)
	if err != nil {
		return nil, err
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("malformed JWKS: %w", err)
	}
	if len(set.Keys) == 0 {
		return nil, fmt.Errorf("JWKS contains no keys")
	}
	if p.MemoryStoreProvider != nil {
		// Cache the normalized JWKS JSON so the cached shape is stable.
		if normalized, mErr := json.Marshal(&set); mErr == nil {
			_ = p.MemoryStoreProvider.SetCache(cacheKey, string(normalized), jwksCacheTTLSeconds)
		}
	}
	return &set, nil
}

// fetchJWKSBytes resolves the raw JWKS document per the row's KeySourceType.
// For oidc_discovery the issuer's discovery document is fetched first to learn
// its jwks_uri; both fetches are SSRF-hardened.
func (p *provider) fetchJWKSBytes(ctx context.Context, issuer *schemas.TrustedIssuer) ([]byte, error) {
	switch issuer.KeySourceType {
	case constants.KeySourceStaticJWKSURL:
		if issuer.JWKSUrl == nil || strings.TrimSpace(*issuer.JWKSUrl) == "" {
			return nil, fmt.Errorf("static_jwks_url issuer has no jwks_url")
		}
		return p.fetchURL(ctx, *issuer.JWKSUrl)
	case constants.KeySourceOIDCDiscovery:
		discoveryURL := strings.TrimSuffix(issuer.IssuerURL, "/") + "/.well-known/openid-configuration"
		doc, err := p.fetchURL(ctx, discoveryURL)
		if err != nil {
			return nil, err
		}
		var meta struct {
			JWKSURI string `json:"jwks_uri"`
		}
		if err := json.Unmarshal(doc, &meta); err != nil {
			return nil, fmt.Errorf("malformed OIDC discovery document: %w", err)
		}
		if strings.TrimSpace(meta.JWKSURI) == "" {
			return nil, fmt.Errorf("OIDC discovery document has no jwks_uri")
		}
		return p.fetchURL(ctx, meta.JWKSURI)
	default:
		// spiffe_bundle_endpoint and any other source are out of scope for this PR.
		return nil, fmt.Errorf("unsupported key_source_type %q", issuer.KeySourceType)
	}
}

// safeFetchURL performs an SSRF-hardened GET: the host is resolved once and
// pinned (validators.SafeHTTPClient), redirects are refused (a redirect could
// escape to an internal address), and the body is size-capped.
func (p *provider) safeFetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	client, err := validators.SafeHTTPClient(ctx, rawURL, httpFetchTimeout)
	if err != nil {
		return nil, err
	}
	// The dial IP is pinned to the validated host; a cross-host redirect would
	// misdial anyway. Refuse redirects outright to keep the guarantee explicit.
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
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
	return io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes))
}

// tokenReviewRequest / tokenReviewResponse are the minimal shapes of the K8s
// authentication.k8s.io/v1 TokenReview API. Declared inline so the resolver does
// not pull in k8s.io/client-go for a single JSON round-trip.
type tokenReviewRequest struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Spec       struct {
		Token     string   `json:"token"`
		Audiences []string `json:"audiences,omitempty"`
	} `json:"spec"`
}

type tokenReviewResponse struct {
	Status struct {
		Authenticated bool   `json:"authenticated"`
		Error         string `json:"error,omitempty"`
	} `json:"status"`
}

// performTokenReview POSTs the presented token to the cluster's TokenReview API
// and returns nil only when the apiserver reports it still authenticated. It
// fails closed on any error (unreachable apiserver, non-2xx, malformed body,
// authenticated=false) so a deleted-pod token whose exp has not yet passed is
// rejected.
//
// SSRF: the admin-supplied apiServerURL is routed through the same SSRF-hardened
// client as JWKS fetches (validators.SafeHTTPClient) — the host is resolved once
// and pinned, and redirects are refused.
//
// KNOWN LIMITATION (reported, not silently worked around): SafeHTTPClient rejects
// private/loopback/link-local IPs, so the common in-cluster apiserver address
// (https://kubernetes.default.svc, a ClusterIP in a private range) is NOT
// reachable through it. TokenReview therefore works today only against a
// publicly-routable apiserver endpoint (e.g. a managed cluster's public API
// endpoint). Reaching a private in-cluster apiserver needs a deliberate SSRF
// exemption + CA-pinning decision, which is intentionally left to the operator/
// security owner rather than weakening the SSRF guard here.
func (p *provider) performTokenReview(ctx context.Context, apiServerURL, token, expectedAud string) error {
	var reqBody tokenReviewRequest
	reqBody.APIVersion = "authentication.k8s.io/v1"
	reqBody.Kind = "TokenReview"
	reqBody.Spec.Token = token
	if strings.TrimSpace(expectedAud) != "" {
		reqBody.Spec.Audiences = []string{expectedAud}
	}
	body, err := json.Marshal(&reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal TokenReview request: %w", err)
	}

	reviewURL := strings.TrimSuffix(apiServerURL, "/") + tokenReviewPath
	client, err := p.safeHTTPClient(ctx, reviewURL, httpFetchTimeout)
	if err != nil {
		return err
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reviewURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Authenticate to the apiserver with Authorizer's own in-cluster SA token when
	// present; a TokenReview call is otherwise anonymous (which most clusters
	// reject — surfaced as authenticated=false / non-2xx and thus fail-closed).
	if saToken, rErr := os.ReadFile(inClusterSATokenPath); rErr == nil {
		if t := strings.TrimSpace(string(saToken)); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("TokenReview API returned status %d", resp.StatusCode)
	}

	// Size-cap the apiserver-controlled body; a truncated read surfaces as a JSON
	// parse failure below → fail-closed.
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes))
	var review tokenReviewResponse
	if err := json.Unmarshal(raw, &review); err != nil {
		return fmt.Errorf("malformed TokenReview response: %w", err)
	}
	if !review.Status.Authenticated {
		return fmt.Errorf("TokenReview reported the token is not authenticated: %s", review.Status.Error)
	}
	return nil
}

// assertionReplayKey derives the single-use key. A jti (when present) is the
// canonical single-use handle; otherwise (iss,sub,iat,exp) uniquely identifies
// the token issuance and is hashed to bound the key length.
func assertionReplayKey(issuerID string, claims jwt.MapClaims, iss, sub string, iat, exp int64) string {
	if jti, ok := claims["jti"].(string); ok && strings.TrimSpace(jti) != "" {
		return "assertion_jti:" + issuerID + ":" + strings.TrimSpace(jti)
	}
	h := sha256.Sum256([]byte(iss + "|" + sub + "|" + strconv.FormatInt(iat, 10) + "|" + strconv.FormatInt(exp, 10)))
	return "assertion_replay:" + issuerID + ":" + hex.EncodeToString(h[:])
}

// isAllowedAlg reports whether alg is in the asymmetric allow-list.
func isAllowedAlg(alg string) bool {
	return contains(allowedAssertionAlgs, alg)
}

// audienceContains reports whether the JWT `aud` claim (a string or an array of
// strings per RFC 7519 §4.1.3) contains expected exactly.
func audienceContains(aud interface{}, expected string) bool {
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
		return contains(v, expected)
	}
	return false
}

// claimInt64 reads a numeric JWT claim, tolerating both float64 (encoding/json
// default) and json.Number encodings.
func claimInt64(claims jwt.MapClaims, key string) (int64, bool) {
	switch v := claims[key].(type) {
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case int64:
		return v, true
	}
	return 0, false
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
