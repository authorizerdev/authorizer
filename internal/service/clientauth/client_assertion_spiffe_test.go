package clientauth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// SPIFFE fixtures. The SPIRE server (iss) is deliberately NOT the workload's
// SPIFFE ID (sub): for a JWT-SVID iss ≠ sub is expected, and the resolver keys
// the trust-row lookup on iss while pinning sub against AllowedSubjects.
const (
	testSpireIssuerURL = "https://spire-server.test.example.com"
	testSpiffeSubject  = "spiffe://example.org/ns/prod/sa/payments"
)

// buildSpiffeResolver mirrors buildResolver but marks the trust row as a SPIFFE
// issuer (IssuerType = spiffe_jwt) and keys it on the SPIRE-server issuer URL.
func buildSpiffeResolver(t *testing.T, jwks []byte, allowedSubjects string) *provider {
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
			testSpireIssuerURL: {
				ID:              "spiffe-row-1",
				ClientID:        testSAClientPK,
				IssuerURL:       testSpireIssuerURL,
				IssuerType:      constants.IssuerTypeSPIFFEJWT,
				KeySourceType:   constants.KeySourceStaticJWKSURL,
				JWKSUrl:         refString("https://spire-server.test.example.com/keys"),
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

func spiffeClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": testSpireIssuerURL,
		"sub": testSpiffeSubject,
		"aud": testExpectedAud,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"jti": "spiffe-jti-" + now.Format("150405.000000000"),
	}
}

func spiffeParams(assertion string) ResolveParams {
	return ResolveParams{
		ClientAssertion:           assertion,
		ClientAssertionType:       constants.ClientAssertionTypeJWTSPIFFE,
		RequireServiceAccountKind: true,
	}
}

// TestSpiffeAssertion_ValidAuthenticates: a JWT-SVID with iss = SPIRE server and
// sub = a spiffe:// ID in AllowedSubjects authenticates. Proves iss ≠ sub is
// handled (the SPIRE iss locates the row; the SPIFFE-ID sub is pinned).
func TestSpiffeAssertion_ValidAuthenticates(t *testing.T) {
	key := genKey(t)
	r := buildSpiffeResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSpiffeSubject)

	client, err := r.ResolveClient(context.Background(), spiffeParams(signRS256(t, key, testKID, spiffeClaims())))
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, testSAClientPK, client.ID)
}

// TestSpiffeAssertion_WrongSubjectRejected: a valid JWT-SVID whose spiffe:// sub
// is not in AllowedSubjects is rejected.
func TestSpiffeAssertion_WrongSubjectRejected(t *testing.T) {
	key := genKey(t)
	r := buildSpiffeResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSpiffeSubject)
	claims := spiffeClaims()
	claims["sub"] = "spiffe://example.org/ns/prod/sa/attacker"
	_, err := r.ResolveClient(context.Background(), spiffeParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

// TestSpiffeAssertion_NonSpiffeSubjectRejected: a subject that is NOT a spiffe://
// URI is rejected by the SPIFFE-ID format gate even when it appears verbatim in
// AllowedSubjects — a spiffe row only authenticates SPIFFE IDs.
func TestSpiffeAssertion_NonSpiffeSubjectRejected(t *testing.T) {
	key := genKey(t)
	nonSpiffe := "system:serviceaccount:prod:payments"
	r := buildSpiffeResolver(t, jwksBytes(t, &key.PublicKey, testKID), nonSpiffe)
	claims := spiffeClaims()
	claims["sub"] = nonSpiffe
	_, err := r.ResolveClient(context.Background(), spiffeParams(signRS256(t, key, testKID, claims)))
	assert.ErrorIs(t, err, ErrInvalidClient, "a non-SPIFFE subject must be rejected on the jwt-spiffe path")
}

// TestSpiffeAssertion_EmptyAllowedSubjectsDenyAll: an empty AllowedSubjects on a
// spiffe row authenticates nobody.
func TestSpiffeAssertion_EmptyAllowedSubjectsDenyAll(t *testing.T) {
	key := genKey(t)
	r := buildSpiffeResolver(t, jwksBytes(t, &key.PublicKey, testKID), "")
	_, err := r.ResolveClient(context.Background(), spiffeParams(signRS256(t, key, testKID, spiffeClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient, "empty AllowedSubjects must deny-all on the jwt-spiffe path")
}

// TestSpiffeAssertion_AlgNoneRejected: alg:none is rejected on the SPIFFE path.
func TestSpiffeAssertion_AlgNoneRejected(t *testing.T) {
	key := genKey(t)
	r := buildSpiffeResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSpiffeSubject)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, spiffeClaims())
	unsigned, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = r.ResolveClient(context.Background(), spiffeParams(unsigned))
	assert.ErrorIs(t, err, ErrInvalidClient, "alg:none must be rejected on the jwt-spiffe path")
}

// TestSpiffeAssertion_BearerTypeAgainstSpiffeRowRejected: presenting the plain
// jwt-bearer type against a spiffe_jwt row is rejected (type ↔ row mismatch) —
// this stops routing a SPIFFE row through the bearer profile to skip the
// spiffe:// subject-format gate.
func TestSpiffeAssertion_BearerTypeAgainstSpiffeRowRejected(t *testing.T) {
	key := genKey(t)
	r := buildSpiffeResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSpiffeSubject)
	params := spiffeParams(signRS256(t, key, testKID, spiffeClaims()))
	params.ClientAssertionType = constants.ClientAssertionTypeJWTBearer
	_, err := r.ResolveClient(context.Background(), params)
	assert.ErrorIs(t, err, ErrInvalidClient)
}

// TestSpiffeAssertion_SpiffeTypeAgainstBearerRowRejected: presenting the
// jwt-spiffe type against a non-SPIFFE (jwt-bearer) row is rejected — the other
// half of the type ↔ row consistency guard. buildResolver's row is a generic
// (empty issuer_type) client_assertion row.
func TestSpiffeAssertion_SpiffeTypeAgainstBearerRowRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	params := assertionParams(signRS256(t, key, testKID, validClaims()))
	params.ClientAssertionType = constants.ClientAssertionTypeJWTSPIFFE
	_, err := r.ResolveClient(context.Background(), params)
	assert.ErrorIs(t, err, ErrInvalidClient)
}
