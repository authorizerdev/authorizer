package clientauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// The JWKS-mirror pattern is the documented answer for clusters left on the
// default --service-account-issuer (kubeadm, kind), whose issuer and jwks_uri
// are private addresses the SSRF guard refuses.
//
// It works because of one non-obvious property: under static_jwks_url the
// issuer_url is a MATCHING KEY, never an address. It is looked up
// (GetTrustedIssuerByIssuerURL) and compared against the assertion's `iss`; the
// only code path that fetches it is the oidc_discovery branch. So an operator
// can register issuer_url = https://kubernetes.default.svc — unroutable from
// outside the cluster, and never dialed — while pointing jwks_url at a
// reachable mirror of /openid/v1/jwks.
//
// That property is load-bearing for the docs and invisible in the type system:
// nothing stops a future change from resolving issuer_url "for validation" and
// silently breaking every private-issuer deployment. This test is the guard.
func TestPrivateIssuerURLIsNeverDialedUnderStaticJWKS(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	// The address a default-issuer cluster stamps into every projected token.
	const clusterIssuer = "https://kubernetes.default.svc.cluster.local"

	store := r.StorageProvider.(*assertionStore)
	row := store.issuers[testIssuerURL]
	delete(store.issuers, testIssuerURL)
	row.IssuerURL = clusterIssuer
	row.KeySourceType = constants.KeySourceStaticJWKSURL
	row.JWKSUrl = refString("https://jwks-mirror.example.com/openid/v1/jwks")
	store.issuers[clusterIssuer] = row

	// Record every URL the resolver actually fetches. Asserting on the RESOLVED
	// OUTCOME alone would not guard this: the fetch seam returns the same JWKS
	// whatever it is handed, so a future change that dialed issuer_url would
	// still authenticate here and the regression would ship silently.
	var fetched []string
	jwks := jwksBytes(t, &key.PublicKey, testKID)
	r.fetchURL = func(_ context.Context, url string) ([]byte, error) {
		fetched = append(fetched, url)
		return jwks, nil
	}

	claims := validClaims()
	claims["iss"] = clusterIssuer

	client, err := r.ResolveClient(context.Background(),
		assertionParams(signRS256(t, key, testKID, claims)))

	require.NoError(t, err,
		"a private issuer_url must never be dialed under static_jwks_url — this is what makes "+
			"the documented JWKS-mirror setup work for default-issuer clusters")
	assert.Equal(t, testSAClientPK, client.ID)

	require.Equal(t, []string{"https://jwks-mirror.example.com/openid/v1/jwks"}, fetched,
		"the mirror must be the ONLY thing fetched; dialing issuer_url (or its "+
			"/.well-known/openid-configuration) would break every default-issuer cluster, "+
			"which is precisely the setup the docs tell those operators to use")
}
