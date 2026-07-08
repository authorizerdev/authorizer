package clientauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const (
	testIssuerURL   = "https://issuer.test.example.com"
	testExpectedAud = "https://authorizer.example.com/oauth/token"
	testSubject     = "system:serviceaccount:prod:payments"
	testKID         = "test-key-1"
	testSAClientPK  = "sa-pk-1"
)

// assertionStore overrides only the three methods the client_assertion path
// touches; everything else panics via the embedded nil interface.
type assertionStore struct {
	storage.Provider
	clientsByID map[string]*schemas.Client
	issuers     map[string]*schemas.TrustedIssuer
}

func (s *assertionStore) GetClientByID(_ context.Context, id string) (*schemas.Client, error) {
	c, ok := s.clientsByID[id]
	if !ok {
		return nil, errors.New("client not found")
	}
	return c, nil
}

func (s *assertionStore) GetTrustedIssuerByIssuerURL(_ context.Context, url string) (*schemas.TrustedIssuer, error) {
	iss, ok := s.issuers[url]
	if !ok {
		return nil, errors.New("issuer not found")
	}
	return iss, nil
}

// fakeMemStore implements the SetCache/GetCache subset used by the resolver.
type fakeMemStore struct {
	memory_store.Provider
	mu    sync.Mutex
	cache map[string]string
}

func newFakeMemStore() *fakeMemStore { return &fakeMemStore{cache: map[string]string{}} }

func (m *fakeMemStore) SetCache(key, value string, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *fakeMemStore) GetCache(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cache[key], nil
}

// --- test fixtures ---

func genKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	// A 2048-bit key is standard; generation is the slow part but runs once per test.
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

func jwksBytes(t *testing.T, pub *rsa.PublicKey, kid string) []byte {
	t.Helper()
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: pub, KeyID: kid, Algorithm: "RS256", Use: "sig"}}}
	b, err := json.Marshal(set)
	require.NoError(t, err)
	return b
}

func signRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString(key)
	require.NoError(t, err)
	return s
}

func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": testIssuerURL,
		"sub": testSubject,
		"aud": testExpectedAud,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"jti": "jti-" + now.Format("150405.000000000"),
	}
}

// buildResolver wires a resolver whose JWKS fetch is stubbed to serve jwks, and
// registers one active service_account client + one active trusted issuer with
// the given allowedSubjects.
func buildResolver(t *testing.T, jwks []byte, allowedSubjects string) *provider {
	t.Helper()
	logger := zerolog.Nop()
	store := &assertionStore{
		clientsByID: map[string]*schemas.Client{
			testSAClientPK: {
				ID:       testSAClientPK,
				ClientID: "payments-sa",
				Kind:     constants.ClientKindServiceAccount,
				IsActive: true,
			},
		},
		issuers: map[string]*schemas.TrustedIssuer{
			testIssuerURL: {
				ID:              "issuer-row-1",
				ClientID:        testSAClientPK,
				IssuerURL:       testIssuerURL,
				KeySourceType:   constants.KeySourceStaticJWKSURL,
				JWKSUrl:         refString("https://issuer.test.example.com/jwks.json"),
				ExpectedAud:     testExpectedAud,
				SubjectClaim:    "sub",
				AllowedSubjects: allowedSubjects,
				IsActive:        true,
			},
		},
	}
	p := New(
		&config.Config{ClientID: "reserved", ClientSecret: "reserved-secret"},
		&Dependencies{Log: &logger, StorageProvider: store, MemoryStoreProvider: newFakeMemStore()},
	).(*provider)
	p.fetchURL = func(_ context.Context, _ string) ([]byte, error) { return jwks, nil }
	return p
}

func refString(s string) *string { return &s }

func assertionParams(assertion string) ResolveParams {
	return ResolveParams{
		ClientAssertion:           assertion,
		ClientAssertionType:       constants.ClientAssertionTypeJWTBearer,
		RequireServiceAccountKind: true,
	}
}

// --- tests ---

func TestClientAssertion_ValidAuthenticates(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	client, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, testSAClientPK, client.ID)
}

func TestClientAssertion_ValidWithArrayAudience(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	claims := validClaims()
	claims["aud"] = []string{"some-other-aud", testExpectedAud}
	client, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	require.NoError(t, err)
	assert.Equal(t, testSAClientPK, client.ID)
}

func TestClientAssertion_ReplayRejected_WithJTI(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	assertion := signRS256(t, key, testKID, validClaims())

	_, err := r.ResolveClient(context.Background(), assertionParams(assertion))
	require.NoError(t, err)
	_, err = r.ResolveClient(context.Background(), assertionParams(assertion))
	assert.ErrorIs(t, err, ErrInvalidClient, "replayed jti must be rejected")
}

func TestClientAssertion_ReplayRejected_NoJTI(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	claims := validClaims()
	delete(claims, "jti") // K8s SA tokens carry no jti — (iss,sub,iat,exp) is the key.
	assertion := signRS256(t, key, testKID, claims)

	_, err := r.ResolveClient(context.Background(), assertionParams(assertion))
	require.NoError(t, err)
	_, err = r.ResolveClient(context.Background(), assertionParams(assertion))
	assert.ErrorIs(t, err, ErrInvalidClient, "replayed no-jti assertion must be rejected on (iss,sub,iat,exp)")
}

func TestClientAssertion_WrongSubjectRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	claims := validClaims()
	claims["sub"] = "system:serviceaccount:prod:attacker"
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_SubjectPrefixRejected(t *testing.T) {
	key := genKey(t)
	// Pin "prod"; a prefix-adjacent subject "prod-evil" must NOT match (H3, exact).
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), "prod")
	claims := validClaims()
	claims["sub"] = "prod-evil"
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_EmptyAllowedSubjectsDenyAll(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), "") // deny-all
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient, "empty AllowedSubjects must authenticate nobody")
}

func TestClientAssertion_HS256WithPublicKeyRejected(t *testing.T) {
	// JWKS-confusion attack: an attacker who knows the issuer's PUBLIC key (it's
	// published in the JWKS) forges an HS256 token using that public key's PEM as
	// the HMAC secret. It MUST be rejected — the asymmetric-only algorithm
	// allow-list means HS256 is never accepted, so a public key is never treated as
	// a symmetric secret. This locks the property against future refactors of the
	// parse options.
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
	tok.Header["kid"] = testKID
	forged, err := tok.SignedString(pubPEM)
	require.NoError(t, err)

	_, err = r.ResolveClient(context.Background(), assertionParams(forged))
	assert.ErrorIs(t, err, ErrInvalidClient, "HS256 signed with the JWKS public key must be rejected")
}

func TestClientAssertion_AlgNoneRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims())
	unsigned, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = r.ResolveClient(context.Background(), assertionParams(unsigned))
	assert.ErrorIs(t, err, ErrInvalidClient, "alg:none must be rejected")
}

func TestClientAssertion_MismatchedAudRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	claims := validClaims()
	claims["aud"] = "https://some-other-service.example.com"
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_LifetimeOverCeilingRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	now := time.Now()
	claims := validClaims()
	// exp − iat = 2h, over the 1h default ceiling — rejected even though not expired.
	claims["iat"] = now.Add(-90 * time.Minute).Unix()
	claims["exp"] = now.Add(30 * time.Minute).Unix()
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_ExpiredRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	now := time.Now()
	claims := validClaims()
	claims["iat"] = now.Add(-10 * time.Minute).Unix()
	claims["exp"] = now.Add(-5 * time.Minute).Unix()
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_WrongSignatureRejected(t *testing.T) {
	signKey := genKey(t)
	otherKey := genKey(t)
	// JWKS advertises otherKey; the token is signed by signKey → signature fails.
	r := buildResolver(t, jwksBytes(t, &otherKey.PublicKey, testKID), testSubject)
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, signKey, testKID, validClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_UnknownIssuerRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	claims := validClaims()
	claims["iss"] = "https://unregistered.example.com"
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_InactiveIssuerRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	r.StorageProvider.(*assertionStore).issuers[testIssuerURL].IsActive = false
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestClientAssertion_UnsupportedTypeRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	params := assertionParams(signRS256(t, key, testKID, validClaims()))
	params.ClientAssertionType = "urn:ietf:params:oauth:client-assertion-type:saml2-bearer" // not supported
	_, err := r.ResolveClient(context.Background(), params)
	assert.ErrorIs(t, err, ErrUnsupportedAssertionType)
}

func TestClientAssertion_SecretAndAssertionRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	params := assertionParams(signRS256(t, key, testKID, validClaims()))
	params.BodySecret = "a-secret" // presenting two auth methods (RFC 6749 §2.3)
	_, err := r.ResolveClient(context.Background(), params)
	assert.ErrorIs(t, err, ErrMultipleAuthMethods)
}

func TestClientAssertion_InactiveClientRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	r.StorageProvider.(*assertionStore).clientsByID[testSAClientPK].IsActive = false
	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient)
}
