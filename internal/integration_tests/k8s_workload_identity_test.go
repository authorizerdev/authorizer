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
	"net"
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

// TestK8sAddressReachabilityFollowsTheAddressClass pins the actual rule, which is
// narrower than "Kubernetes does not work".
//
// Whether this feature works on a given cluster is decided entirely by what that
// cluster publishes as its issuer / jwks_uri / apiserver:
//
//   - Self-managed clusters running the DEFAULT --service-account-issuer
//     (https://kubernetes.default.svc.cluster.local — what kind, and kubeadm
//     without extra flags, produce) publish private addresses. SafeHTTPClient
//     refuses those, so key fetch and TokenReview are both unreachable.
//
//   - EKS, GKE and AKS publish a PUBLIC https issuer with public OIDC discovery
//     by default (https://oidc.eks.<region>.amazonaws.com/id/…,
//     https://container.googleapis.com/v1/projects/…). Those are accepted, and
//     the feature works there with no configuration beyond the trusted issuer.
//
// An earlier version of this test asserted refusal unconditionally, which is a
// false generalisation from kind: it fails on any managed cluster, where the
// correct outcome is that the address is ACCEPTED. Assert the rule instead of
// one cluster's instance of it, so the suite is truthful on whatever cluster it
// is pointed at.
func TestK8sAddressReachabilityFollowsTheAddressClass(t *testing.T) {
	f := k8sFacts(t)
	ctx := context.Background()

	for name, raw := range map[string]string{
		"jwks_uri":  f.jwksURI,
		"apiserver": f.apiServer,
	} {
		t.Run(name, func(t *testing.T) {
			private, why := addressIsPrivate(t, raw)
			_, err := validators.SafeHTTPClient(ctx, raw, 3*time.Second)

			if private {
				require.Error(t, err,
					"%s (%s) resolves to a private address (%s) and MUST be refused; if this now "+
						"succeeds the SSRF policy changed and the Kubernetes story changed with it",
					name, raw, why)
				assert.Contains(t, err.Error(), "private/internal networks are not allowed",
					"the refusal must come from the SSRF guard specifically, not a TLS or DNS "+
						"failure that would mask it")
				t.Logf("%s is private (%s) — unreachable, as expected on a default-issuer cluster", name, why)
				return
			}

			require.NoError(t, err,
				"%s (%s) is publicly routable, so the guard MUST accept it — this is the "+
					"EKS/GKE/AKS case, where the feature works with no extra configuration",
				name, raw)
			t.Logf("%s is public — reachable, feature works on this cluster", name)
		})
	}
}

// addressIsPrivate reports whether a URL's host resolves into a range the SSRF
// guard rejects. It re-resolves rather than reusing the guard's own answer, so
// the test's expectation is derived independently of the code under test.
func addressIsPrivate(t *testing.T, raw string) (bool, string) {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err, "cluster published an unparseable address: %s", raw)

	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast(), ip.String()
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// Unresolvable in-cluster DNS (kubernetes.default.svc from outside the
		// cluster) is the private case by construction.
		return true, "unresolvable: " + host
	}
	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return true, ip.String()
		}
	}
	return false, ips[0].String()
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

	// The expected outcome follows the address class, exactly as above. On a
	// default-issuer cluster the JWKS fetch is refused and the workload cannot
	// authenticate; on EKS/GKE/AKS the JWKS is public and it can.
	if private, why := addressIsPrivate(t, f.jwksURI); private {
		assert.NotEqual(t, http.StatusOK, w.Code,
			"jwks_uri is private (%s), so SafeHTTPClient refuses the fetch and the workload "+
				"cannot authenticate. The operator-visible symptom is this bare invalid_client "+
				"with nothing naming the refused fetch. Body: %s", why, w.Body.String())
		t.Logf("default-issuer cluster: %d %s", w.Code, w.Body.String())
		return
	}
	assert.Equal(t, http.StatusOK, w.Code,
		"jwks_uri is publicly routable, so a projected token MUST authenticate with no "+
			"configuration beyond this trusted issuer. Body: %s", w.Body.String())
	t.Logf("public-issuer cluster: workload authenticated")
}
