package clientauth

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// fakeStore implements only GetClientByClientID; every other storage.Provider
// method is promoted from the embedded nil interface and would panic if called.
// The resolver only ever calls GetClientByClientID, so this stays valid.
type fakeStore struct {
	storage.Provider
	clients map[string]*schemas.Client
}

func (f *fakeStore) GetClientByClientID(_ context.Context, clientID string) (*schemas.Client, error) {
	c, ok := f.clients[clientID]
	if !ok {
		return nil, errors.New("client not found")
	}
	return c, nil
}

const (
	testClientID     = "worker-1"
	testClientSecret = "s3cr3t-plaintext"
	testConfigID     = "reserved-client"
	testConfigSecret = "config-plaintext-secret"
)

func newResolver(t *testing.T, clients map[string]*schemas.Client) Provider {
	t.Helper()
	logger := zerolog.Nop()
	return New(
		&config.Config{ClientID: testConfigID, ClientSecret: testConfigSecret},
		&Dependencies{Log: &logger, StorageProvider: &fakeStore{clients: clients}},
	)
}

func hashSecret(t *testing.T, secret string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(secret), dummySecretCost)
	require.NoError(t, err)
	return string(h)
}

func confidentialClient(t *testing.T) *schemas.Client {
	return &schemas.Client{
		ID:                      "id-worker-1",
		ClientID:                testClientID,
		Kind:                    constants.ClientKindServiceAccount,
		ClientSecret:            hashSecret(t, testClientSecret),
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodClientSecretBasic,
		IsActive:                true,
	}
}

func TestResolveClient_ClientSecretBasic(t *testing.T) {
	r := newResolver(t, map[string]*schemas.Client{testClientID: confidentialClient(t)})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BasicClientID: testClientID,
		BasicSecret:   testClientSecret,
		HasBasicAuth:  true,
		RequireSecret: true,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, testClientID, got.ClientID)
}

func TestResolveClient_ClientSecretPost(t *testing.T) {
	r := newResolver(t, map[string]*schemas.Client{testClientID: confidentialClient(t)})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:  testClientID,
		BodySecret:    testClientSecret,
		RequireSecret: true,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, testClientID, got.ClientID)
}

func TestResolveClient_DualAuthMethodsRejected(t *testing.T) {
	r := newResolver(t, map[string]*schemas.Client{testClientID: confidentialClient(t)})
	// A Basic credential AND a body client_secret in one request (RFC 6749 §2.3).
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:  testClientID,
		BodySecret:    testClientSecret,
		BasicClientID: testClientID,
		BasicSecret:   testClientSecret,
		HasBasicAuth:  true,
		RequireSecret: true,
	})
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrMultipleAuthMethods)
}

func TestResolveClient_WrongSecret(t *testing.T) {
	r := newResolver(t, map[string]*schemas.Client{testClientID: confidentialClient(t)})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:  testClientID,
		BodySecret:    "wrong-secret",
		RequireSecret: true,
	})
	assert.ErrorIs(t, err, ErrInvalidClient)
	// The resolved client is returned so the caller can attribute an audit event.
	require.NotNil(t, got)
	assert.Equal(t, "id-worker-1", got.ID)
}

func TestResolveClient_UnknownClient(t *testing.T) {
	r := newResolver(t, map[string]*schemas.Client{})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:  "does-not-exist",
		BodySecret:    "whatever",
		RequireSecret: true,
	})
	assert.ErrorIs(t, err, ErrInvalidClient)
	// Unknown client returns a nil client (nothing to attribute an audit to); the
	// dummy bcrypt compare inside keeps timing indistinguishable from wrong-secret.
	assert.Nil(t, got)
}

func TestResolveClient_MissingClientID(t *testing.T) {
	r := newResolver(t, map[string]*schemas.Client{})
	got, err := r.ResolveClient(context.Background(), ResolveParams{BodySecret: "x"})
	assert.ErrorIs(t, err, ErrMissingClientID)
	assert.Nil(t, got)
}

func TestResolveClient_InactiveClient(t *testing.T) {
	c := confidentialClient(t)
	c.IsActive = false
	r := newResolver(t, map[string]*schemas.Client{testClientID: c})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:  testClientID,
		BodySecret:    testClientSecret, // correct secret, but account is inactive
		RequireSecret: true,
	})
	assert.ErrorIs(t, err, ErrInvalidClient)
	require.NotNil(t, got)
}

func TestResolveClient_PublicClientNoSecret(t *testing.T) {
	// A public client (token_endpoint_auth_method == "none") presents no secret;
	// authorization_code sets VerifyPresentedSecret=true but with no secret there
	// is nothing to verify — PKCE (enforced by the caller) is the proof.
	public := &schemas.Client{
		ID:                      "id-public",
		ClientID:                "public-app",
		Kind:                    constants.ClientKindInteractive,
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodNone,
		IsActive:                true,
	}
	r := newResolver(t, map[string]*schemas.Client{"public-app": public})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:          "public-app",
		VerifyPresentedSecret: true,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "public-app", got.ClientID)
}

func TestResolveClient_RefreshTokenIgnoresSecret(t *testing.T) {
	// refresh_token authenticates the client_id only: a presented (wrong) secret
	// is ignored, reproducing the pre-registry behavior.
	r := newResolver(t, map[string]*schemas.Client{testClientID: confidentialClient(t)})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID: testClientID,
		BodySecret:   "wrong-secret-but-ignored",
		// Both RequireSecret and VerifyPresentedSecret are false (refresh_token).
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, testClientID, got.ClientID)
}

func TestResolveClient_ConfigFallbackWhenRowAbsent(t *testing.T) {
	// The reserved client's row is absent (read-only replica); the resolver must
	// fall back to Config.ClientID / Config.ClientSecret so login is never locked
	// out (BC availability fallback).
	r := newResolver(t, map[string]*schemas.Client{})

	t.Run("correct_secret_authenticates", func(t *testing.T) {
		got, err := r.ResolveClient(context.Background(), ResolveParams{
			BodyClientID:          testConfigID,
			BodySecret:            testConfigSecret,
			VerifyPresentedSecret: true,
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, testConfigID, got.ClientID)
		assert.Equal(t, constants.ClientKindInteractive, got.Kind)
	})

	t.Run("no_secret_authenticates_pkce_path", func(t *testing.T) {
		got, err := r.ResolveClient(context.Background(), ResolveParams{
			BodyClientID:          testConfigID,
			VerifyPresentedSecret: true, // authorization_code, but no secret presented
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, testConfigID, got.ClientID)
	})

	t.Run("wrong_secret_returns_client_and_error", func(t *testing.T) {
		got, err := r.ResolveClient(context.Background(), ResolveParams{
			BodyClientID:          testConfigID,
			BodySecret:            "wrong",
			VerifyPresentedSecret: true,
		})
		assert.ErrorIs(t, err, ErrInvalidClient)
		// Non-nil so the caller can distinguish a known client_id from an unknown
		// one (drives the 401-vs-400 status choice at the token endpoint).
		require.NotNil(t, got)
		assert.Empty(t, got.ID, "the synthesized fallback client has no surrogate ID")
	})
}

func TestResolveClient_ClientCredentialsEmptySecretRejected(t *testing.T) {
	// client_credentials always requires a secret; an empty one must be rejected
	// (RequireSecret forces the compare, which fails on the empty secret).
	r := newResolver(t, map[string]*schemas.Client{testClientID: confidentialClient(t)})
	got, err := r.ResolveClient(context.Background(), ResolveParams{
		BodyClientID:  testClientID,
		RequireSecret: true,
	})
	assert.ErrorIs(t, err, ErrInvalidClient)
	require.NotNil(t, got)
}
