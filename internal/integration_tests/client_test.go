package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

func TestClientAdmin(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should reject non-super-admin caller", func(t *testing.T) {
		// No admin cookie set yet on the fresh request.
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "unauthorized",
			AllowedScopes: []string{"read"},
		})
		require.Error(t, err)
		require.Nil(t, res)
	})

	// Everything below runs as super-admin.
	setAdminCookie(t, ts)

	t.Run("should reject empty/whitespace-only allowed_scopes", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "empty-scopes-" + uuid.NewString(),
			AllowedScopes: []string{"  ", "", "\t"},
		})
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("should create and return the plaintext secret exactly once", func(t *testing.T) {
		name := "payments-worker-" + uuid.NewString()
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:        name,
			Description: refs.NewStringRef("payments background worker"),
			// Duplicates and whitespace are normalized away.
			AllowedScopes: []string{"read", " write ", "read", ""},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Client)

		// Plaintext secret is returned once and is NOT the stored hash.
		require.NotEmpty(t, res.ClientSecret)
		assert.Equal(t, name, res.Client.Name)
		// client_id is exposed and defaults to the internal id on create.
		assert.NotEmpty(t, res.Client.ClientID)
		assert.Equal(t, res.Client.ID, res.Client.ClientID)
		assert.True(t, res.Client.IsActive)
		assert.ElementsMatch(t, []string{"read", "write"}, res.Client.AllowedScopes)

		// Storage holds only a bcrypt hash (cost 12) of the plaintext — never the
		// plaintext itself.
		stored, err := ts.StorageProvider.GetClientByID(ctx, res.Client.ID)
		require.NoError(t, err)
		assert.NotEqual(t, res.ClientSecret, stored.ClientSecret)
		cost, err := bcrypt.Cost([]byte(stored.ClientSecret))
		require.NoError(t, err)
		assert.Equal(t, 12, cost)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.ClientSecret), []byte(res.ClientSecret)))
		assert.Equal(t, "read,write", stored.AllowedScopes)

		// A subsequent fetch returns the account but has no way to surface the
		// secret — the model type has no client_secret field by design.
		fetched, err := ts.GraphQLProvider.Client(ctx, &model.ClientRequest{ID: res.Client.ID})
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, name, fetched.Name)
	})

	t.Run("rotate changes the stored hash and invalidates the old secret", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "rotate-me-" + uuid.NewString(),
			AllowedScopes: []string{"read"},
		})
		require.NoError(t, err)
		oldPlaintext := res.ClientSecret
		id := res.Client.ID

		before, err := ts.StorageProvider.GetClientByID(ctx, id)
		require.NoError(t, err)
		oldHash := before.ClientSecret

		rotated, err := ts.GraphQLProvider.RotateClientSecret(ctx, &model.ClientRequest{ID: id})
		require.NoError(t, err)
		require.NotNil(t, rotated)
		newPlaintext := rotated.ClientSecret
		require.NotEmpty(t, newPlaintext)
		assert.NotEqual(t, oldPlaintext, newPlaintext)

		after, err := ts.StorageProvider.GetClientByID(ctx, id)
		require.NoError(t, err)
		assert.NotEqual(t, oldHash, after.ClientSecret, "stored hash must change on rotation")

		// Old plaintext no longer validates against the new hash; new one does.
		assert.Error(t, bcrypt.CompareHashAndPassword([]byte(after.ClientSecret), []byte(oldPlaintext)))
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(after.ClientSecret), []byte(newPlaintext)))
	})

	t.Run("update mutates only supplied fields and preserves the rest", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "partial-update-" + uuid.NewString(),
			Description:   refs.NewStringRef("original description"),
			AllowedScopes: []string{"read", "write"},
		})
		require.NoError(t, err)
		id := res.Client.ID

		// Update only the name.
		updated, err := ts.GraphQLProvider.UpdateClient(ctx, &model.UpdateClientRequest{
			ID:   id,
			Name: refs.NewStringRef("renamed"),
		})
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "renamed", updated.Name)
		// Untouched columns are preserved.
		require.NotNil(t, updated.Description)
		assert.Equal(t, "original description", *updated.Description)
		assert.ElementsMatch(t, []string{"read", "write"}, updated.AllowedScopes)
		assert.True(t, updated.IsActive)

		// Toggle is_active off without touching anything else.
		deactivated, err := ts.GraphQLProvider.UpdateClient(ctx, &model.UpdateClientRequest{
			ID:       id,
			IsActive: refs.NewBoolRef(false),
		})
		require.NoError(t, err)
		assert.False(t, deactivated.IsActive)
		assert.Equal(t, "renamed", deactivated.Name)
	})

	t.Run("update rejects collapsing allowed_scopes to empty", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "no-empty-update-" + uuid.NewString(),
			AllowedScopes: []string{"read"},
		})
		require.NoError(t, err)

		updated, err := ts.GraphQLProvider.UpdateClient(ctx, &model.UpdateClientRequest{
			ID:            res.Client.ID,
			AllowedScopes: []string{" ", ""},
		})
		require.Error(t, err)
		require.Nil(t, updated)
	})

	t.Run("list returns created accounts without exposing secrets", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Clients(ctx, &model.ListClientsRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.Pagination)
		assert.Greater(t, len(res.Clients), 0)
	})

	t.Run("delete removes the account", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateClient(ctx, &model.CreateClientRequest{
			Name:          "delete-me-" + uuid.NewString(),
			AllowedScopes: []string{"read"},
		})
		require.NoError(t, err)
		id := res.Client.ID

		delRes, err := ts.GraphQLProvider.DeleteClient(ctx, &model.ClientRequest{ID: id})
		require.NoError(t, err)
		require.NotNil(t, delRes)

		_, err = ts.StorageProvider.GetClientByID(ctx, id)
		require.Error(t, err)
	})
}
