package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

func TestTrustedIssuerAdmin(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Helper: create a service account as admin and return its id.
	createSA := func(t *testing.T) string {
		t.Helper()
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "issuer-parent-" + uuid.NewString(),
			AllowedScopes: []string{"read"},
		})
		require.NoError(t, err)
		return res.Client.ID
	}

	newIssuerReq := func(saID string) *model.AddTrustedIssuerRequest {
		return &model.AddTrustedIssuerRequest{
			ServiceAccountID: saID,
			Name:             "prod-k8s",
			IssuerURL:        "https://issuer-" + uuid.NewString() + ".example.org",
			KeySourceType:    "oidc_discovery",
			ExpectedAud:      "https://authorizer.example.org",
			IssuerType:       "kubernetes_sa",
		}
	}

	t.Run("should reject non-super-admin caller", func(t *testing.T) {
		// Fresh context without the admin cookie.
		_, freshCtx := createContext(ts)
		res, err := ts.GraphQLProvider.AddTrustedIssuer(freshCtx, newIssuerReq(uuid.NewString()))
		require.Error(t, err)
		require.Nil(t, res)
	})

	setAdminCookie(t, ts)

	t.Run("should reject an issuer bound to a missing service account", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(uuid.NewString()))
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("should add an issuer and default subject_claim to sub", func(t *testing.T) {
		saID := createSA(t)
		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(saID))
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, saID, res.ServiceAccountID)
		assert.Equal(t, "sub", res.SubjectClaim)
		assert.True(t, res.IsActive)

		fetched, err := ts.GraphQLProvider.TrustedIssuer(ctx, &model.TrustedIssuerRequest{ID: res.ID})
		require.NoError(t, err)
		assert.Equal(t, res.ID, fetched.ID)
	})

	t.Run("should honor an explicit subject_claim", func(t *testing.T) {
		saID := createSA(t)
		reqIssuer := newIssuerReq(saID)
		reqIssuer.SubjectClaim = refs.NewStringRef("email")
		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, reqIssuer)
		require.NoError(t, err)
		assert.Equal(t, "email", res.SubjectClaim)
	})

	t.Run("should reject a duplicate issuer_url", func(t *testing.T) {
		saID := createSA(t)
		req := newIssuerReq(saID)
		_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
		require.NoError(t, err)

		// Second issuer with the same issuer_url must be rejected — otherwise
		// GetTrustedIssuerByIssuerURL (used on every client_assertion validation)
		// resolves nondeterministically.
		dup := newIssuerReq(createSA(t))
		dup.IssuerURL = req.IssuerURL
		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, dup)
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("update mutates only supplied fields", func(t *testing.T) {
		saID := createSA(t)
		created, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(saID))
		require.NoError(t, err)

		updated, err := ts.GraphQLProvider.UpdateTrustedIssuer(ctx, &model.UpdateTrustedIssuerRequest{
			ID:       created.ID,
			IsActive: refs.NewBoolRef(false),
		})
		require.NoError(t, err)
		assert.False(t, updated.IsActive)
		// Untouched columns preserved.
		assert.Equal(t, created.IssuerURL, updated.IssuerURL)
		assert.Equal(t, created.ExpectedAud, updated.ExpectedAud)
		assert.Equal(t, saID, updated.ServiceAccountID)
	})

	t.Run("update rejects blanking expected_aud", func(t *testing.T) {
		saID := createSA(t)
		created, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(saID))
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.UpdateTrustedIssuer(ctx, &model.UpdateTrustedIssuerRequest{
			ID:          created.ID,
			ExpectedAud: refs.NewStringRef("  "),
		})
		require.Error(t, err)
	})

	t.Run("add persists and round-trips token review config", func(t *testing.T) {
		saID := createSA(t)
		req := newIssuerReq(saID)
		req.EnableTokenReview = refs.NewBoolRef(true)
		req.KubernetesAPIServerURL = refs.NewStringRef("https://kubernetes.example.com:443")

		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.EnableTokenReview)
		require.NotNil(t, res.KubernetesAPIServerURL)
		assert.Equal(t, "https://kubernetes.example.com:443", *res.KubernetesAPIServerURL)

		fetched, err := ts.GraphQLProvider.TrustedIssuer(ctx, &model.TrustedIssuerRequest{ID: res.ID})
		require.NoError(t, err)
		assert.True(t, fetched.EnableTokenReview)
		require.NotNil(t, fetched.KubernetesAPIServerURL)
		assert.Equal(t, "https://kubernetes.example.com:443", *fetched.KubernetesAPIServerURL)
	})

	t.Run("add rejects enable_token_review without an apiserver url", func(t *testing.T) {
		saID := createSA(t)
		req := newIssuerReq(saID)
		req.EnableTokenReview = refs.NewBoolRef(true)

		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "kubernetes_api_server_url is required")
	})

	t.Run("add rejects a non-https apiserver url", func(t *testing.T) {
		saID := createSA(t)
		req := newIssuerReq(saID)
		req.EnableTokenReview = refs.NewBoolRef(true)
		req.KubernetesAPIServerURL = refs.NewStringRef("http://kubernetes.example.com")

		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "must be an https URL")
	})

	t.Run("add rejects a malformed apiserver url", func(t *testing.T) {
		saID := createSA(t)
		req := newIssuerReq(saID)
		// Non-empty but has no host: fails the well-formed check even with review off.
		req.KubernetesAPIServerURL = refs.NewStringRef("https://")

		res, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "must include a host")
	})

	t.Run("update round-trips token review config and rejects enabling without url", func(t *testing.T) {
		saID := createSA(t)
		created, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(saID))
		require.NoError(t, err)
		assert.False(t, created.EnableTokenReview)

		// Enabling review without ever storing a url must fail (fail-closed).
		_, err = ts.GraphQLProvider.UpdateTrustedIssuer(ctx, &model.UpdateTrustedIssuerRequest{
			ID:                created.ID,
			EnableTokenReview: refs.NewBoolRef(true),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kubernetes_api_server_url is required")

		// Enabling review together with a valid url succeeds and round-trips.
		updated, err := ts.GraphQLProvider.UpdateTrustedIssuer(ctx, &model.UpdateTrustedIssuerRequest{
			ID:                     created.ID,
			EnableTokenReview:      refs.NewBoolRef(true),
			KubernetesAPIServerURL: refs.NewStringRef("https://k8s.internal.example.com"),
		})
		require.NoError(t, err)
		assert.True(t, updated.EnableTokenReview)
		require.NotNil(t, updated.KubernetesAPIServerURL)
		assert.Equal(t, "https://k8s.internal.example.com", *updated.KubernetesAPIServerURL)
	})

	t.Run("list filters by service_account_id", func(t *testing.T) {
		saID := createSA(t)
		_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(saID))
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.TrustedIssuers(ctx, &model.ListTrustedIssuersRequest{
			ServiceAccountID: refs.NewStringRef(saID),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, len(res.TrustedIssuers))
		assert.Equal(t, saID, res.TrustedIssuers[0].ServiceAccountID)
	})

	t.Run("deleting a service account cascades to its trusted issuers", func(t *testing.T) {
		saID := createSA(t)
		issuer, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, newIssuerReq(saID))
		require.NoError(t, err)

		// Sanity: issuer exists before delete.
		_, err = ts.StorageProvider.GetTrustedIssuerByID(ctx, issuer.ID)
		require.NoError(t, err)

		delRes, err := ts.GraphQLProvider.DeleteClient(ctx, &model.ClientRequest{ID: saID})
		require.NoError(t, err)
		require.NotNil(t, delRes)

		// The trusted issuer must be gone — no orphan left behind.
		_, err = ts.StorageProvider.GetTrustedIssuerByID(ctx, issuer.ID)
		require.Error(t, err)

		listRes, _, err := ts.StorageProvider.ListTrustedIssuers(ctx, saID, &model.Pagination{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 0, len(listRes))
	})

	// key_source_type and issuer_type were checked for non-emptiness only, so any
	// string was stored verbatim. The value that mattered was
	// "spiffe_bundle_endpoint": a declared constant with no implementation, which
	// fetchJWKSBytes rejects at authentication time with "unsupported
	// key_source_type". An operator got a 200 and a row that looked configured,
	// and found out it was dead only when the first workload tried to
	// authenticate. A plain typo failed identically, at the same unhelpful moment.
	t.Run("key_source_type is allow-listed against what is implemented", func(t *testing.T) {
		saID := createSA(t)

		t.Run("spiffe_bundle_endpoint is refused as not implemented", func(t *testing.T) {
			req := newIssuerReq(saID)
			req.KeySourceType = constants.KeySourceSPIFFEBundleEndpoint
			_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
			require.Error(t, err, "a key source with no implementation must not be storable")
			assert.Contains(t, err.Error(), "not implemented",
				"the error must say the value is unimplemented, not merely invalid — "+
					"they call for different actions")
		})

		t.Run("a typo is refused", func(t *testing.T) {
			req := newIssuerReq(saID)
			req.KeySourceType = "static_jwks_urls" // trailing s
			_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported key_source_type")
		})

		for _, good := range []string{constants.KeySourceOIDCDiscovery, constants.KeySourceStaticJWKSURL} {
			t.Run("implemented source "+good+" is still accepted", func(t *testing.T) {
				req := newIssuerReq(saID)
				req.KeySourceType = good
				_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
				require.NoError(t, err, "the allow-list must not break a working configuration")
			})
		}
	})

	// issuer_type drives no behaviour today — nothing switches on it — but it is
	// stored, returned by the admin API and rendered in the dashboard. An
	// unconstrained free-text column shaped like an enum diverges across
	// deployments and cannot later be given meaning without a migration.
	t.Run("issuer_type is allow-listed", func(t *testing.T) {
		saID := createSA(t)

		req := newIssuerReq(saID)
		req.IssuerType = "kubernetes_service_account" // plausible, and wrong
		_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported issuer_type")

		for _, good := range []string{
			constants.IssuerTypeKubernetesSA, constants.IssuerTypeSPIFFEJWT,
			constants.IssuerTypeOIDC, constants.IssuerTypeCloudOIDC,
		} {
			ok := newIssuerReq(saID)
			ok.IssuerType = good
			_, err := ts.GraphQLProvider.AddTrustedIssuer(ctx, ok)
			require.NoError(t, err, "declared issuer type %q must be accepted", good)
		}
	})
}
