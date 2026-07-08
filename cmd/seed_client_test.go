package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage"
)

func newSeedTestProvider(t *testing.T) (storage.Provider, *config.Config) {
	t.Helper()
	cfg := &config.Config{
		DatabaseType: constants.DbTypeSqlite,
		DatabaseURL:  filepath.Join(t.TempDir(), "seed_client_test.db"),
		DatabaseName: "authorizer_test",
		ClientID:     "reserved-client-" + t.Name(),
		ClientSecret: "super-secret-cookie-key",
	}
	logger := zerolog.Nop()
	sp, err := storage.New(cfg, &storage.Dependencies{Log: &logger})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sp.Close() })
	return sp, cfg
}

// TestSeedReservedClient_IsIdempotent verifies the boot seed inserts exactly one
// reserved client, keyed on Config.ClientID, and is a no-op on repeat calls.
func TestSeedReservedClient_IsIdempotent(t *testing.T) {
	sp, cfg := newSeedTestProvider(t)
	logger := zerolog.Nop()
	ctx := context.Background()

	seedReservedClient(ctx, sp, cfg, &logger)
	seedReservedClient(ctx, sp, cfg, &logger)

	// Exactly one row carries the reserved client_id.
	clients, _, err := sp.ListClients(ctx, &model.Pagination{Limit: 100, Offset: 0})
	require.NoError(t, err)
	count := 0
	for _, c := range clients {
		if c.ClientID == cfg.ClientID {
			count++
		}
	}
	assert.Equal(t, 1, count, "seeding twice must yield exactly one reserved client row")
}

// TestSeedReservedClient_RowShape verifies the seeded row's identity, kind, and
// that its stored secret is a bcrypt hash verifying against Config.ClientSecret
// (BC1: client_id == Config.ClientID).
func TestSeedReservedClient_RowShape(t *testing.T) {
	sp, cfg := newSeedTestProvider(t)
	logger := zerolog.Nop()
	ctx := context.Background()

	seedReservedClient(ctx, sp, cfg, &logger)

	row, err := sp.GetClientByClientID(ctx, cfg.ClientID)
	require.NoError(t, err)
	require.NotNil(t, row)

	assert.Equal(t, cfg.ClientID, row.ClientID, "BC1: seeded client_id must equal Config.ClientID")
	assert.Equal(t, constants.ClientKindInteractive, row.Kind)
	assert.Equal(t, constants.TokenEndpointAuthMethodClientSecretBasic, row.TokenEndpointAuthMethod)
	assert.True(t, row.IsActive)
	assert.Contains(t, row.GrantTypes, constants.GrantTypeAuthorizationCode)
	assert.Contains(t, row.GrantTypes, constants.GrantTypeRefreshToken)

	// The stored secret is a bcrypt hash of Config.ClientSecret, never plaintext.
	assert.NotEqual(t, cfg.ClientSecret, row.ClientSecret, "stored secret must be hashed, not plaintext")
	assert.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(row.ClientSecret), []byte(cfg.ClientSecret)),
		"stored bcrypt hash must verify against Config.ClientSecret",
	)
}

// TestSeedReservedClient_EmptyClientIDSkips verifies an empty Config.ClientID is
// a safe no-op (no row, no panic).
func TestSeedReservedClient_EmptyClientIDSkips(t *testing.T) {
	sp, cfg := newSeedTestProvider(t)
	cfg.ClientID = ""
	logger := zerolog.Nop()
	ctx := context.Background()

	seedReservedClient(ctx, sp, cfg, &logger)

	clients, _, err := sp.ListClients(ctx, &model.Pagination{Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.Empty(t, clients, "empty Config.ClientID must seed nothing")
}
