// Package clientauth resolves and authenticates the OAuth client presented at
// the token endpoint (RFC 6749 §2.3). It is the single source of truth for
// client-secret verification so every transport (token, and — in later PRs —
// introspect/revoke) authenticates clients identically.
package clientauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Sentinel errors callers map to the RFC 6749 §5.2 token-endpoint responses.
var (
	// ErrMultipleAuthMethods is returned when a request presents more than one
	// client-authentication method (RFC 6749 §2.3: "The client MUST NOT use more
	// than one authentication method in each request."). Map to invalid_request.
	ErrMultipleAuthMethods = errors.New("clientauth: multiple authentication methods presented")

	// ErrMissingClientID is returned when no client_id can be extracted from the
	// request. Map to invalid_request.
	ErrMissingClientID = errors.New("clientauth: client_id is required")

	// ErrInvalidClient is returned when the client is unknown, inactive, or the
	// presented secret does not match. Map to invalid_client. When the client was
	// resolved (a known client with a wrong secret or an inactive account) the
	// returned *schemas.Client is non-nil so the caller can attribute an audit
	// event to it; on an unknown client it is nil.
	ErrInvalidClient = errors.New("clientauth: client authentication failed")

	// ErrUnauthorizedClient is returned when a resolved client is not permitted to
	// use the requested grant — e.g. an interactive client attempting
	// client_credentials (design §4.1: client_credentials is machine-only). It is
	// returned BEFORE the secret is verified, so a correct and an incorrect secret
	// produce the identical response — no secret-confirmation oracle. Map to
	// unauthorized_client (RFC 6749 §5.2).
	ErrUnauthorizedClient = errors.New("clientauth: client not authorized for this grant")

	// ErrUnsupportedAssertionType is returned when a client_assertion is presented
	// with a missing or unsupported client_assertion_type. Only
	// urn:ietf:params:oauth:client-assertion-type:jwt-bearer is supported here
	// (the jwt-spiffe type is a follow-up PR). Map to invalid_request.
	ErrUnsupportedAssertionType = errors.New("clientauth: unsupported client_assertion_type")
)

// dummySecretCost mirrors the bcrypt cost real client secrets are hashed with
// (admin_clients.go clientSecretCost == 12, and the reserved-client seed). A
// dummy compare for an unknown client MUST take the same time as a real compare
// or timing reveals whether the client exists.
const dummySecretCost = 12

var (
	dummyHash []byte
	dummyOnce sync.Once
)

// performDummyCompare runs a constant-cost bcrypt comparison whose result is
// discarded, equalising the unknown-client path with a real secret verification.
func performDummyCompare(secret string) {
	dummyOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), dummySecretCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(secret))
}

// ResolveParams carries the client credentials extracted from a token request.
// The transport layer owns extraction (it has the *http.Request); this keeps the
// resolver transport-agnostic and unit-testable.
type ResolveParams struct {
	// BodyClientID / BodySecret are client_id and client_secret from the request
	// body (client_secret_post, or a public client sending only client_id).
	BodyClientID string
	BodySecret   string

	// BasicClientID / BasicSecret / HasBasicAuth carry the HTTP Basic
	// (client_secret_basic) credential. HasBasicAuth is true when a well-formed
	// Authorization: Basic header was present.
	BasicClientID string
	BasicSecret   string
	HasBasicAuth  bool

	// RequireSecret verifies the presented secret even when it is empty
	// (client_credentials — a machine identity always authenticates with a
	// secret). Implies VerifyPresentedSecret.
	RequireSecret bool

	// VerifyPresentedSecret verifies the secret only when one is presented, and
	// treats a missing secret as "no secret" (authorization_code — the caller's
	// PKCE checks gate a secret-less request). When both flags are false
	// (refresh_token) the secret is ignored entirely, only the client_id is
	// authenticated — reproducing the pre-registry refresh_token behavior.
	VerifyPresentedSecret bool

	// RequireServiceAccountKind rejects a resolved client whose Kind is not
	// service_account with ErrUnauthorizedClient, BEFORE the secret is verified.
	// Set for client_credentials (a machine-only grant, design §4.1). Rejecting
	// pre-verification is what keeps the response identical for a correct and an
	// incorrect secret, so the interactive reserved client_id cannot be used as a
	// secret-confirmation oracle on this grant.
	RequireServiceAccountKind bool

	// ClientAssertion / ClientAssertionType carry the RFC 7523 JWT-bearer client
	// credential. When ClientAssertion is non-empty the resolver authenticates the
	// client by verifying the assertion against a registered TrustedIssuer instead
	// of a secret. ClientAssertionType MUST be
	// urn:ietf:params:oauth:client-assertion-type:jwt-bearer.
	ClientAssertion     string
	ClientAssertionType string
}

// Dependencies for the clientauth provider.
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
	// MemoryStoreProvider backs the shared, cross-instance caches used by the
	// client_assertion path: the single-use jti/replay markers and the per-trust-
	// row JWKS cache. Optional for the secret-only paths.
	MemoryStoreProvider memory_store.Provider
}

// Provider resolves and authenticates the OAuth client for a token request.
type Provider interface {
	// ResolveClient extracts the client credential from params, enforces the
	// single-auth-method rule (RFC 6749 §2.3), looks the client up by its public
	// client_id, and verifies the secret. See the sentinel errors for the caller
	// contract.
	ResolveClient(ctx context.Context, params ResolveParams) (*schemas.Client, error)
}

type provider struct {
	*config.Config
	Dependencies

	// maxAssertionLifetime bounds a client_assertion's exp−iat span (§5.2 H4).
	// Defaults to defaultMaxClientAssertionLifetime; in-package tests may override.
	maxAssertionLifetime time.Duration

	// fetchURL is the SSRF-hardened HTTP fetch seam used by the JWKS/OIDC-discovery
	// resolution. Production uses safeFetchURL; in-package tests inject a stub so
	// the resolver is exercised without a real network round-trip (loopback is
	// deliberately blocked by the SSRF guard, so httptest cannot be reached).
	fetchURL func(ctx context.Context, rawURL string) ([]byte, error)
}

var _ Provider = &provider{}

// New constructs a clientauth provider. cfg supplies the bootstrap
// ClientID/ClientSecret fallback that keeps a deployment from being locked out
// when the reserved-client row is absent (read-only replica).
func New(cfg *config.Config, deps *Dependencies) Provider {
	p := &provider{
		Config:               cfg,
		Dependencies:         *deps,
		maxAssertionLifetime: defaultMaxClientAssertionLifetime,
	}
	p.fetchURL = p.safeFetchURL
	return p
}

func (p *provider) ResolveClient(ctx context.Context, params ResolveParams) (*schemas.Client, error) {
	log := p.Log.With().Str("func", "ResolveClient").Logger()

	// RFC 6749 §2.3: "The client MUST NOT use more than one authentication method
	// in each request." Count the presented methods — HTTP Basic, body
	// client_secret, and the RFC 7523 client_assertion are mutually exclusive.
	methods := 0
	if params.HasBasicAuth {
		methods++
	}
	if params.BodySecret != "" {
		methods++
	}
	if params.ClientAssertion != "" {
		methods++
	}
	if methods > 1 {
		log.Debug().Msg("multiple client authentication methods presented")
		return nil, ErrMultipleAuthMethods
	}

	// RFC 7523 client_assertion path: the client authenticates with a signed JWT
	// issued by a registered TrustedIssuer instead of a shared secret. No
	// client_id parameter is required — the client is derived from the trust row.
	if params.ClientAssertion != "" {
		return p.resolveViaClientAssertion(ctx, params)
	}

	// Select the effective credential. Basic wins when present; otherwise the
	// body carries the client_secret_post / public-client parameters.
	clientID := strings.TrimSpace(params.BodyClientID)
	secret := params.BodySecret
	if params.HasBasicAuth {
		clientID = strings.TrimSpace(params.BasicClientID)
		secret = params.BasicSecret
	}
	if clientID == "" {
		log.Debug().Msg("client_id missing")
		return nil, ErrMissingClientID
	}
	secretPresented := secret != ""
	// doVerify decides whether the secret is checked at all: always for
	// client_credentials (RequireSecret), only-when-present for authorization_code
	// (VerifyPresentedSecret), never for refresh_token (both false).
	doVerify := params.RequireSecret || (params.VerifyPresentedSecret && secretPresented)

	client, err := p.StorageProvider.GetClientByClientID(ctx, clientID)
	if err != nil || client == nil {
		// Availability fallback (BC): the reserved client's row may be absent on a
		// read-only replica where the boot seed was skipped. Fall back to the
		// bootstrap Config credential so login is never locked out. This path
		// reproduces the pre-registry constant-time comparison verbatim.
		if p.Config != nil && clientID == strings.TrimSpace(p.Config.ClientID) {
			// The reserved client is interactive; client_credentials is machine-only.
			// Reject before touching the secret so the response is identical for any
			// secret (no confirmation oracle) — matches the found-client branch below.
			if params.RequireServiceAccountKind {
				log.Debug().Msg("reserved interactive client not authorized for client_credentials grant")
				return nil, ErrUnauthorizedClient
			}
			return p.resolveViaConfig(clientID, secret, doVerify)
		}
		// Unknown client: burn an equivalent bcrypt cost so timing does not
		// distinguish an unknown client from a wrong secret, then reject.
		log.Debug().Err(err).Msg("client not found")
		performDummyCompare(secret)
		return nil, ErrInvalidClient
	}

	// Grant matrix (design §4.1): only a service_account client may use
	// client_credentials. Reject any other kind BEFORE verifying the secret, so a
	// correct and an incorrect secret return the identical unauthorized_client — the
	// interactive reserved client_id cannot confirm a guessed secret on this grant.
	if params.RequireServiceAccountKind && client.Kind != constants.ClientKindServiceAccount {
		log.Debug().Str("kind", client.Kind).Msg("client not authorized for client_credentials grant")
		return client, ErrUnauthorizedClient
	}

	// bcrypt.CompareHashAndPassword is itself constant-time with respect to the
	// secret; running it before the IsActive check keeps a wrong-secret and an
	// inactive-account rejection timing-indistinguishable.
	if doVerify {
		if bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(secret)) != nil {
			log.Debug().Msg("client secret mismatch")
			// Return the resolved client so the caller can attribute an audit event.
			return client, ErrInvalidClient
		}
	}

	if !client.IsActive {
		log.Debug().Msg("client is inactive")
		return client, ErrInvalidClient
	}

	return client, nil
}

// resolveViaConfig authenticates the reserved interactive client against the
// bootstrap Config credential when its registry row is absent. The secret is
// compared constant-time against the plaintext Config.ClientSecret — never a
// stored hash — exactly reproducing the pre-registry token-endpoint behavior.
// On mismatch it still returns the synthesized client alongside ErrInvalidClient
// so the caller can tell "known client_id, bad secret" from an unknown client.
func (p *provider) resolveViaConfig(clientID, secret string, doVerify bool) (*schemas.Client, error) {
	// Synthesize the reserved client. ClientSecret (the bcrypt hash) is left empty
	// on purpose — the secret is verified against Config.ClientSecret, not a hash.
	client := &schemas.Client{
		ClientID:                clientID,
		Kind:                    constants.ClientKindInteractive,
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodClientSecretBasic,
		IsActive:                true,
	}
	if doVerify {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(p.Config.ClientSecret)) != 1 {
			return client, ErrInvalidClient
		}
	}
	return client, nil
}
