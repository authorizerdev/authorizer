//go:build k8s

// Package integration_tests, k8s build tag: the Kubernetes workload-identity
// suite. Excluded from every normal `go test` run because it needs a real
// cluster; driven by scripts/k8s-e2e.sh (`make test-k8s`), which creates a kind
// cluster, exports the facts below and tears it down afterwards.
//
// # WHY THIS SUITE EXISTS
//
// The Kubernetes workload-identity path — a projected ServiceAccount token
// presented as an RFC 7523 client_assertion, with keys fetched from the cluster
// and optionally a TokenReview call — had NO test against a real cluster. Every
// existing test substitutes an httptest server for the cluster, which silently
// replaces the one property that actually decides whether the feature works:
// the ADDRESS the cluster publishes.
//
// A real cluster publishes:
//
//	issuer    https://kubernetes.default.svc.cluster.local  (ClusterIP, private)
//	jwks_uri  https://<apiserver-ip>:6443/openid/v1/jwks     (node IP, private)
//	apiserver https://<private-or-loopback>:<port>
//
// and validators.SafeHTTPClient rejects every private, loopback and link-local
// address unconditionally. An httptest server on 127.0.0.1 is refused for the
// same reason, which is why the existing tests inject a plain client instead —
// and so never exercise the guard the real deployment hits first.
//
// These tests therefore assert the CURRENT, documented behaviour against real
// cluster addresses. They are a pin, not an aspiration: if someone changes the
// SSRF policy, this suite tells them exactly which Kubernetes behaviour they
// changed. See the KNOWN LIMITATION block on performTokenReview.
package integration_tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// Facts exported by scripts/k8s-e2e.sh from the live cluster.
type clusterFacts struct {
	issuer    string // .well-known/openid-configuration "issuer"
	jwksURI   string // .well-known/openid-configuration "jwks_uri"
	apiServer string // kubeconfig cluster.server
	saToken   string // kubectl create token, audience-bound
	saSubject string // system:serviceaccount:<ns>:<name>
	audience  string // the --audience the token was minted with
}

func k8sFacts(t *testing.T) clusterFacts {
	t.Helper()
	f := clusterFacts{
		issuer:    os.Getenv("K8S_ISSUER"),
		jwksURI:   os.Getenv("K8S_JWKS_URI"),
		apiServer: os.Getenv("K8S_APISERVER"),
		saToken:   os.Getenv("K8S_SA_TOKEN"),
		saSubject: os.Getenv("K8S_SA_SUBJECT"),
		audience:  os.Getenv("K8S_AUDIENCE"),
	}
	if f.issuer == "" || f.jwksURI == "" || f.saToken == "" {
		t.Skip("no cluster facts in the environment; run via `make test-k8s`")
	}
	return f
}

// TestK8sClusterAddressesAreRefusedBySSRFGuard is the core finding, pinned.
//
// Every address a standard cluster publishes for OIDC discovery, JWKS and the
// apiserver falls in a range validators.SafeHTTPClient rejects. This is not a
// kind artefact: kubernetes.default.svc is ALWAYS a ClusterIP (10.0.0.0/8 or
// 172.16.0.0/12), and jwks_uri always points at the apiserver's internal
// address. Managed clusters with a public API endpoint are the exception the
// feature currently depends on, not the norm it was designed around.
func TestK8sClusterAddressesAreRefusedBySSRFGuard(t *testing.T) {
	f := k8sFacts(t)
	ctx := context.Background()

	for name, raw := range map[string]string{
		"jwks_uri":  f.jwksURI,
		"apiserver": f.apiServer,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := validators.SafeHTTPClient(ctx, raw, 2*time.Second)
			require.Error(t, err,
				"%s (%s) is expected to be refused; if this now succeeds the SSRF policy changed "+
					"and the Kubernetes story changed with it — update the KNOWN LIMITATION on "+
					"performTokenReview and this test together", name, raw)
			assert.Contains(t, err.Error(), "private/internal networks are not allowed",
				"the refusal must come from the SSRF guard specifically, not from a TLS or DNS "+
					"failure that would mask it")
		})
	}
}

// TestK8sProjectedTokenIsWellFormed isolates the failure.
//
// It proves the TOKEN side of workload identity is fine — the projected token
// carries exactly the issuer, subject and audience Authorizer expects — so the
// end-to-end failure below is attributable to the key-fetch address and nothing
// else. Without this, a reader could reasonably assume the token was the problem.
func TestK8sProjectedTokenIsWellFormed(t *testing.T) {
	f := k8sFacts(t)

	parts := strings.Split(f.saToken, ".")
	require.Len(t, parts, 3, "a projected SA token must be a well-formed JWT")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &claims))

	assert.Equal(t, f.issuer, claims["iss"],
		"the token issuer must match the cluster's published issuer")
	assert.Equal(t, f.saSubject, claims["sub"],
		"sub must be the system:serviceaccount:<ns>:<name> Authorizer pins against allowed_subjects")

	auds, _ := claims["aud"].([]interface{})
	require.NotEmpty(t, auds, "an audience-bound projected token must carry aud")
	var found bool
	for _, a := range auds {
		if s, _ := a.(string); s == f.audience {
			found = true
		}
	}
	assert.True(t, found, "aud must contain the audience the token was minted for (%s); got %v",
		f.audience, auds)
}

// TestK8sWorkloadAuthenticationEndToEnd is the operator-visible symptom.
//
// It configures the trusted issuer exactly as the docs describe for a private
// cluster — key_source_type=static_jwks_url pointed at the cluster's own JWKS —
// and presents the real projected token at /oauth/token. The request fails, and
// this test records that it fails so the gap is a checked fact rather than a
// paragraph in a code comment.
//
// When the address problem is solved (a scoped SSRF exemption for an
// operator-declared apiserver/JWKS host with CA pinning, or an in-cluster
// transport that bypasses the guard deliberately), invert the assertion here:
// this is the test that should start demanding a 200.
func TestK8sWorkloadAuthenticationEndToEnd(t *testing.T) {
	f := k8sFacts(t)
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	setAdminCookie(t, ts)

	sa, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
		Name:          "k8s-workload-" + uuid.NewString(),
		AllowedScopes: []string{"openid"},
	})
	require.NoError(t, err)

	_, err = ts.GraphQLProvider.AddTrustedIssuer(ctx, &model.AddTrustedIssuerRequest{
		ServiceAccountID: sa.Client.ID,
		Name:             "kind-cluster",
		IssuerURL:        f.issuer,
		KeySourceType:    constants.KeySourceStaticJWKSURL,
		JwksURL:          &f.jwksURI,
		ExpectedAud:      f.audience,
		IssuerType:       constants.IssuerTypeKubernetesSA,
		AllowedSubjects:  &f.saSubject,
	})
	require.NoError(t, err, "the trusted issuer must be REGISTRABLE — the gap is at fetch time, not config time")

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	form := url.Values{}
	form.Set("grant_type", constants.GrantTypeClientCredentials)
	form.Set("client_assertion_type", constants.ClientAssertionTypeJWTBearer)
	form.Set("client_assertion", f.saToken)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code,
		"KNOWN LIMITATION pinned: a real cluster's JWKS address is private, so SafeHTTPClient "+
			"refuses the fetch and the workload cannot authenticate. If this now returns 200 the "+
			"limitation is fixed — invert this assertion and update README + performTokenReview's "+
			"doc comment. Body: %s", w.Body.String())
	t.Logf("operator-visible response: %d %s", w.Code, w.Body.String())
}
