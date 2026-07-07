package integration_tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// createServiceAccount provisions a service account over gRPC and returns the
// response (which carries the plaintext client secret exactly once). Used by the
// admin service-account and trusted-issuer RPC tests.
func createServiceAccount(t *testing.T, client authorizerv1.AuthorizerAdminServiceClient, ctx context.Context) *authorizerv1.CreateServiceAccountResponse {
	t.Helper()
	resp, err := client.CreateServiceAccount(ctx, &authorizerv1.CreateServiceAccountRequest{
		Name:          "sa-" + uuid.New().String(),
		AllowedScopes: []string{"openid", "profile"},
	})
	require.NoError(t, err)
	return resp
}

// TestAdminCreateServiceAccountGRPC exercises
// AuthorizerAdminService.CreateServiceAccount over gRPC: the fail-closed contract
// (no secret → Unauthenticated), buf.validate rejection of missing fields, and
// the happy path returning the client secret exactly once.
func TestAdminCreateServiceAccountGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.CreateServiceAccount(context.Background(), &authorizerv1.CreateServiceAccountRequest{
			Name:          "sa-" + uuid.New().String(),
			AllowedScopes: []string{"openid"},
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("empty allowed scopes is rejected", func(t *testing.T) {
		_, err := client.CreateServiceAccount(adminCtx(cfg.AdminSecret), &authorizerv1.CreateServiceAccountRequest{
			Name: "sa-" + uuid.New().String(),
		})
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("creates and returns the secret once", func(t *testing.T) {
		resp := createServiceAccount(t, client, adminCtx(cfg.AdminSecret))
		require.NotNil(t, resp.ServiceAccount)
		require.NotEmpty(t, resp.ServiceAccount.Id)
		require.NotEmpty(t, resp.ClientSecret, "create must surface the plaintext secret")
		require.True(t, resp.ServiceAccount.IsActive)
		require.ElementsMatch(t, []string{"openid", "profile"}, resp.ServiceAccount.AllowedScopes)

		// The secret is returned only by CreateServiceAccountResponse. A subsequent
		// Get returns a ServiceAccount message that has NO field to carry it — this
		// is enforced by the proto schema itself (GetServiceAccountResponse.
		// ServiceAccount has no client_secret), so the round-trip cannot leak it.
		got, err := client.GetServiceAccount(adminCtx(cfg.AdminSecret), &authorizerv1.GetServiceAccountRequest{
			Id: resp.ServiceAccount.Id,
		})
		require.NoError(t, err)
		require.Equal(t, resp.ServiceAccount.Id, got.ServiceAccount.Id)
	})
}

// TestAdminUpdateServiceAccountGRPC exercises
// AuthorizerAdminService.UpdateServiceAccount over gRPC: the fail-closed contract
// and a happy-path field update.
func TestAdminUpdateServiceAccountGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createServiceAccount(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		name := "renamed"
		_, err := client.UpdateServiceAccount(context.Background(), &authorizerv1.UpdateServiceAccountRequest{
			Id:   created.ServiceAccount.Id,
			Name: &name,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("updates name and active state", func(t *testing.T) {
		name := "renamed-" + uuid.New().String()
		active := false
		resp, err := client.UpdateServiceAccount(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateServiceAccountRequest{
			Id:       created.ServiceAccount.Id,
			Name:     &name,
			IsActive: &active,
		})
		require.NoError(t, err)
		require.Equal(t, name, resp.ServiceAccount.Name)
		require.False(t, resp.ServiceAccount.IsActive)
	})

	t.Run("updating unknown account is an error", func(t *testing.T) {
		name := "x"
		_, err := client.UpdateServiceAccount(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateServiceAccountRequest{
			Id:   uuid.New().String(),
			Name: &name,
		})
		require.Error(t, err)
	})
}

// TestAdminRotateServiceAccountSecretGRPC exercises
// AuthorizerAdminService.RotateServiceAccountSecret over gRPC: the fail-closed
// contract and a happy-path rotation that yields a fresh secret.
func TestAdminRotateServiceAccountSecretGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createServiceAccount(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.RotateServiceAccountSecret(context.Background(), &authorizerv1.RotateServiceAccountSecretRequest{
			Id: created.ServiceAccount.Id,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("rotates to a new secret", func(t *testing.T) {
		resp, err := client.RotateServiceAccountSecret(adminCtx(cfg.AdminSecret), &authorizerv1.RotateServiceAccountSecretRequest{
			Id: created.ServiceAccount.Id,
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.ClientSecret)
		require.NotEqual(t, created.ClientSecret, resp.ClientSecret, "rotation must yield a new secret")
		require.Equal(t, created.ServiceAccount.Id, resp.ServiceAccount.Id)
	})

	t.Run("rotating unknown account is an error", func(t *testing.T) {
		_, err := client.RotateServiceAccountSecret(adminCtx(cfg.AdminSecret), &authorizerv1.RotateServiceAccountSecretRequest{
			Id: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

// TestAdminDeleteServiceAccountGRPC exercises
// AuthorizerAdminService.DeleteServiceAccount over gRPC: the fail-closed contract
// and a happy-path delete that cascades to the account's trusted issuers.
func TestAdminDeleteServiceAccountGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createServiceAccount(t, client, adminCtx(cfg.AdminSecret))
	saID := created.ServiceAccount.Id

	// Attach a trusted issuer so the delete exercises the cascade.
	_, err := client.AddTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.AddTrustedIssuerRequest{
		ServiceAccountId: saID,
		Name:             "issuer-" + uuid.New().String(),
		IssuerUrl:        "https://issuer.example/" + uuid.New().String(),
		KeySourceType:    "oidc_discovery",
		ExpectedAud:      "authorizer",
		IssuerType:       "oidc",
	})
	require.NoError(t, err)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.DeleteServiceAccount(context.Background(), &authorizerv1.DeleteServiceAccountRequest{Id: saID})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("deletes account and cascades to trusted issuers", func(t *testing.T) {
		resp, err := client.DeleteServiceAccount(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteServiceAccountRequest{Id: saID})
		require.NoError(t, err)
		require.Equal(t, "Service account deleted successfully", resp.Message)

		// The account is gone.
		_, err = client.GetServiceAccount(adminCtx(cfg.AdminSecret), &authorizerv1.GetServiceAccountRequest{Id: saID})
		require.Error(t, err)

		// Its trusted issuers cascaded away.
		issuers, err := client.TrustedIssuers(adminCtx(cfg.AdminSecret), &authorizerv1.TrustedIssuersRequest{
			ServiceAccountId: &saID,
		})
		require.NoError(t, err)
		require.Empty(t, issuers.TrustedIssuers, "deleting the account must cascade to its trusted issuers")
	})
}

// TestAdminServiceAccountsGRPC exercises AuthorizerAdminService.ServiceAccounts
// over gRPC: the fail-closed contract and the happy path with a seeded account in
// the paginated page.
func TestAdminServiceAccountsGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createServiceAccount(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.ServiceAccounts(context.Background(), &authorizerv1.ServiceAccountsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns paginated accounts with admin secret", func(t *testing.T) {
		resp, err := client.ServiceAccounts(adminCtx(cfg.AdminSecret), &authorizerv1.ServiceAccountsRequest{
			Pagination: &authorizerv1.PaginationRequest{Page: 1, Limit: 10},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		var found bool
		for _, sa := range resp.ServiceAccounts {
			if sa.Id == created.ServiceAccount.Id {
				found = true
				break
			}
		}
		require.True(t, found, "created service account should appear in the page")
	})
}
