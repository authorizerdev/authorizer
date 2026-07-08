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

// createClient provisions a service account over gRPC and returns the
// response (which carries the plaintext client secret exactly once). Used by the
// admin service-account and trusted-issuer RPC tests.
func createClient(t *testing.T, client authorizerv1.AuthorizerAdminServiceClient, ctx context.Context) *authorizerv1.CreateClientResponse {
	t.Helper()
	resp, err := client.CreateClient(ctx, &authorizerv1.CreateClientRequest{
		Name:          "sa-" + uuid.New().String(),
		AllowedScopes: []string{"openid", "profile"},
	})
	require.NoError(t, err)
	return resp
}

// TestAdminCreateClientGRPC exercises
// AuthorizerAdminService.CreateClient over gRPC: the fail-closed contract
// (no secret → Unauthenticated), buf.validate rejection of missing fields, and
// the happy path returning the client secret exactly once.
func TestAdminCreateClientGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.CreateClient(context.Background(), &authorizerv1.CreateClientRequest{
			Name:          "sa-" + uuid.New().String(),
			AllowedScopes: []string{"openid"},
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("empty allowed scopes is rejected", func(t *testing.T) {
		_, err := client.CreateClient(adminCtx(cfg.AdminSecret), &authorizerv1.CreateClientRequest{
			Name: "sa-" + uuid.New().String(),
		})
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("creates and returns the secret once", func(t *testing.T) {
		resp := createClient(t, client, adminCtx(cfg.AdminSecret))
		require.NotNil(t, resp.Client)
		require.NotEmpty(t, resp.Client.Id)
		require.NotEmpty(t, resp.ClientSecret, "create must surface the plaintext secret")
		require.True(t, resp.Client.IsActive)
		require.ElementsMatch(t, []string{"openid", "profile"}, resp.Client.AllowedScopes)

		// The secret is returned only by CreateClientResponse. A subsequent
		// Get returns a Client message that has NO field to carry it — this
		// is enforced by the proto schema itself (GetClientResponse.
		// Client has no client_secret), so the round-trip cannot leak it.
		got, err := client.GetClient(adminCtx(cfg.AdminSecret), &authorizerv1.GetClientRequest{
			Id: resp.Client.Id,
		})
		require.NoError(t, err)
		require.Equal(t, resp.Client.Id, got.Client.Id)
	})
}

// TestAdminUpdateClientGRPC exercises
// AuthorizerAdminService.UpdateClient over gRPC: the fail-closed contract
// and a happy-path field update.
func TestAdminUpdateClientGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createClient(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		name := "renamed"
		_, err := client.UpdateClient(context.Background(), &authorizerv1.UpdateClientRequest{
			Id:   created.Client.Id,
			Name: &name,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("updates name and active state", func(t *testing.T) {
		name := "renamed-" + uuid.New().String()
		active := false
		resp, err := client.UpdateClient(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateClientRequest{
			Id:       created.Client.Id,
			Name:     &name,
			IsActive: &active,
		})
		require.NoError(t, err)
		require.Equal(t, name, resp.Client.Name)
		require.False(t, resp.Client.IsActive)
	})

	t.Run("updating unknown account is an error", func(t *testing.T) {
		name := "x"
		_, err := client.UpdateClient(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateClientRequest{
			Id:   uuid.New().String(),
			Name: &name,
		})
		require.Error(t, err)
	})
}

// TestAdminRotateClientSecretGRPC exercises
// AuthorizerAdminService.RotateClientSecret over gRPC: the fail-closed
// contract and a happy-path rotation that yields a fresh secret.
func TestAdminRotateClientSecretGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createClient(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.RotateClientSecret(context.Background(), &authorizerv1.RotateClientSecretRequest{
			Id: created.Client.Id,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("rotates to a new secret", func(t *testing.T) {
		resp, err := client.RotateClientSecret(adminCtx(cfg.AdminSecret), &authorizerv1.RotateClientSecretRequest{
			Id: created.Client.Id,
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.ClientSecret)
		require.NotEqual(t, created.ClientSecret, resp.ClientSecret, "rotation must yield a new secret")
		require.Equal(t, created.Client.Id, resp.Client.Id)
	})

	t.Run("rotating unknown account is an error", func(t *testing.T) {
		_, err := client.RotateClientSecret(adminCtx(cfg.AdminSecret), &authorizerv1.RotateClientSecretRequest{
			Id: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

// TestAdminDeleteClientGRPC exercises
// AuthorizerAdminService.DeleteClient over gRPC: the fail-closed contract
// and a happy-path delete that cascades to the account's trusted issuers.
func TestAdminDeleteClientGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createClient(t, client, adminCtx(cfg.AdminSecret))
	saID := created.Client.Id

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
		_, err := client.DeleteClient(context.Background(), &authorizerv1.DeleteClientRequest{Id: saID})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("deletes account and cascades to trusted issuers", func(t *testing.T) {
		resp, err := client.DeleteClient(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteClientRequest{Id: saID})
		require.NoError(t, err)
		require.Equal(t, "Service account deleted successfully", resp.Message)

		// The account is gone.
		_, err = client.GetClient(adminCtx(cfg.AdminSecret), &authorizerv1.GetClientRequest{Id: saID})
		require.Error(t, err)

		// Its trusted issuers cascaded away.
		issuers, err := client.TrustedIssuers(adminCtx(cfg.AdminSecret), &authorizerv1.TrustedIssuersRequest{
			ServiceAccountId: &saID,
		})
		require.NoError(t, err)
		require.Empty(t, issuers.TrustedIssuers, "deleting the account must cascade to its trusted issuers")
	})
}

// TestAdminClientsGRPC exercises AuthorizerAdminService.Clients
// over gRPC: the fail-closed contract and the happy path with a seeded account in
// the paginated page.
func TestAdminClientsGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	created := createClient(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.Clients(context.Background(), &authorizerv1.ClientsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns paginated accounts with admin secret", func(t *testing.T) {
		resp, err := client.Clients(adminCtx(cfg.AdminSecret), &authorizerv1.ClientsRequest{
			Pagination: &authorizerv1.PaginationRequest{Page: 1, Limit: 10},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		var found bool
		for _, sa := range resp.Clients {
			if sa.Id == created.Client.Id {
				found = true
				break
			}
		}
		require.True(t, found, "created service account should appear in the page")
	})
}
