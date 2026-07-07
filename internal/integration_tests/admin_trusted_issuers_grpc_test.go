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

// addTrustedIssuer registers a trusted issuer for the given service account over
// gRPC and returns the created issuer. Used by the trusted-issuer RPC tests.
func addTrustedIssuer(t *testing.T, client authorizerv1.AuthorizerAdminServiceClient, ctx context.Context, serviceAccountID string) *authorizerv1.TrustedIssuer {
	t.Helper()
	resp, err := client.AddTrustedIssuer(ctx, &authorizerv1.AddTrustedIssuerRequest{
		ServiceAccountId: serviceAccountID,
		Name:             "issuer-" + uuid.New().String(),
		IssuerUrl:        "https://issuer.example/" + uuid.New().String(),
		KeySourceType:    "oidc_discovery",
		ExpectedAud:      "authorizer",
		IssuerType:       "oidc",
	})
	require.NoError(t, err)
	return resp.TrustedIssuer
}

// TestAdminAddTrustedIssuerGRPC exercises AuthorizerAdminService.AddTrustedIssuer
// over gRPC: the fail-closed contract, buf.validate rejection of missing fields,
// the parent-must-exist guard, and the happy path (subject_claim defaults to
// "sub").
func TestAdminAddTrustedIssuerGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	sa := createClient(t, client, adminCtx(cfg.AdminSecret))

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.AddTrustedIssuer(context.Background(), &authorizerv1.AddTrustedIssuerRequest{
			ServiceAccountId: sa.Client.Id,
			Name:             "issuer",
			IssuerUrl:        "https://issuer.example/x",
			KeySourceType:    "oidc_discovery",
			ExpectedAud:      "authorizer",
			IssuerType:       "oidc",
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("missing required fields are rejected", func(t *testing.T) {
		_, err := client.AddTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.AddTrustedIssuerRequest{
			ServiceAccountId: sa.Client.Id,
		})
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("unknown service account is an error", func(t *testing.T) {
		_, err := client.AddTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.AddTrustedIssuerRequest{
			ServiceAccountId: uuid.New().String(),
			Name:             "issuer",
			IssuerUrl:        "https://issuer.example/" + uuid.New().String(),
			KeySourceType:    "oidc_discovery",
			ExpectedAud:      "authorizer",
			IssuerType:       "oidc",
		})
		require.Error(t, err)
	})

	t.Run("adds issuer and defaults subject_claim to sub", func(t *testing.T) {
		issuer := addTrustedIssuer(t, client, adminCtx(cfg.AdminSecret), sa.Client.Id)
		require.NotEmpty(t, issuer.Id)
		require.Equal(t, sa.Client.Id, issuer.ServiceAccountId)
		require.Equal(t, "sub", issuer.SubjectClaim)
		require.True(t, issuer.IsActive)
	})
}

// TestAdminUpdateTrustedIssuerGRPC exercises
// AuthorizerAdminService.UpdateTrustedIssuer over gRPC: the fail-closed contract
// and a happy-path field update.
func TestAdminUpdateTrustedIssuerGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	sa := createClient(t, client, adminCtx(cfg.AdminSecret))
	issuer := addTrustedIssuer(t, client, adminCtx(cfg.AdminSecret), sa.Client.Id)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		name := "renamed"
		_, err := client.UpdateTrustedIssuer(context.Background(), &authorizerv1.UpdateTrustedIssuerRequest{
			Id:   issuer.Id,
			Name: &name,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("updates name and expected aud", func(t *testing.T) {
		name := "renamed-" + uuid.New().String()
		aud := "authorizer-v2"
		resp, err := client.UpdateTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateTrustedIssuerRequest{
			Id:          issuer.Id,
			Name:        &name,
			ExpectedAud: &aud,
		})
		require.NoError(t, err)
		require.Equal(t, name, resp.TrustedIssuer.Name)
		require.Equal(t, aud, resp.TrustedIssuer.ExpectedAud)
	})

	t.Run("updating unknown issuer is an error", func(t *testing.T) {
		name := "x"
		_, err := client.UpdateTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateTrustedIssuerRequest{
			Id:   uuid.New().String(),
			Name: &name,
		})
		require.Error(t, err)
	})
}

// TestAdminDeleteTrustedIssuerGRPC exercises
// AuthorizerAdminService.DeleteTrustedIssuer over gRPC: the fail-closed contract
// and a happy-path delete.
func TestAdminDeleteTrustedIssuerGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	sa := createClient(t, client, adminCtx(cfg.AdminSecret))
	issuer := addTrustedIssuer(t, client, adminCtx(cfg.AdminSecret), sa.Client.Id)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.DeleteTrustedIssuer(context.Background(), &authorizerv1.DeleteTrustedIssuerRequest{Id: issuer.Id})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("deletes issuer with admin secret", func(t *testing.T) {
		resp, err := client.DeleteTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteTrustedIssuerRequest{Id: issuer.Id})
		require.NoError(t, err)
		require.Equal(t, "Trusted issuer deleted successfully", resp.Message)

		_, err = client.GetTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.GetTrustedIssuerRequest{Id: issuer.Id})
		require.Error(t, err)
	})

	t.Run("deleting unknown issuer is an error", func(t *testing.T) {
		_, err := client.DeleteTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteTrustedIssuerRequest{Id: uuid.New().String()})
		require.Error(t, err)
	})
}

// TestAdminGetTrustedIssuerGRPC exercises AuthorizerAdminService.GetTrustedIssuer
// over gRPC: the fail-closed contract and the happy path against a seeded issuer.
func TestAdminGetTrustedIssuerGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	sa := createClient(t, client, adminCtx(cfg.AdminSecret))
	issuer := addTrustedIssuer(t, client, adminCtx(cfg.AdminSecret), sa.Client.Id)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.GetTrustedIssuer(context.Background(), &authorizerv1.GetTrustedIssuerRequest{Id: issuer.Id})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns issuer with admin secret", func(t *testing.T) {
		resp, err := client.GetTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.GetTrustedIssuerRequest{Id: issuer.Id})
		require.NoError(t, err)
		require.Equal(t, issuer.Id, resp.TrustedIssuer.Id)
		require.Equal(t, sa.Client.Id, resp.TrustedIssuer.ServiceAccountId)
	})

	t.Run("unknown issuer is an error", func(t *testing.T) {
		_, err := client.GetTrustedIssuer(adminCtx(cfg.AdminSecret), &authorizerv1.GetTrustedIssuerRequest{Id: uuid.New().String()})
		require.Error(t, err)
	})
}

// TestAdminTrustedIssuersGRPC exercises AuthorizerAdminService.TrustedIssuers over
// gRPC: the fail-closed contract, the happy path with a seeded issuer present,
// and the service_account_id filter.
func TestAdminTrustedIssuersGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	sa := createClient(t, client, adminCtx(cfg.AdminSecret))
	issuer := addTrustedIssuer(t, client, adminCtx(cfg.AdminSecret), sa.Client.Id)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.TrustedIssuers(context.Background(), &authorizerv1.TrustedIssuersRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns paginated issuers filtered by service account", func(t *testing.T) {
		saID := sa.Client.Id
		resp, err := client.TrustedIssuers(adminCtx(cfg.AdminSecret), &authorizerv1.TrustedIssuersRequest{
			ServiceAccountId: &saID,
			Pagination:       &authorizerv1.PaginationRequest{Page: 1, Limit: 10},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		require.Len(t, resp.TrustedIssuers, 1)
		require.Equal(t, issuer.Id, resp.TrustedIssuers[0].Id)
	})
}
