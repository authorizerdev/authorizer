package cmd

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// reservedClientSecretCost is the bcrypt cost for the reserved client's secret.
// It mirrors the cost the admin client API commits to (see
// internal/service/admin_clients.go clientSecretCost) — MUST stay 12.
const reservedClientSecretCost = 12

// seedReservedClient idempotently upserts the deployment's reserved interactive
// client into the client registry, keyed on ClientID == Config.ClientID (BC1).
//
// The row makes the deployment's global OAuth client a first-class registry
// record so later PRs can resolve client authentication from storage. This PR
// only makes the row EXIST; it does NOT rewire any handler — existing
// Config.ClientID comparisons stay as-is.
//
// BC2: Config.ClientSecret remains the session-cookie AES key untouched. We only
// ADD a bcrypt hash of it into the client row; cookie crypto is not changed.
//
// The seed is idempotent (skip-if-present, keyed on ClientID). On a read-only /
// no-write-path storage instance the AddClient error is logged and skipped
// rather than fatal, so a read replica can still boot.
func seedReservedClient(ctx context.Context, storageProvider storage.Provider, cfg *config.Config, logger *zerolog.Logger) {
	log := logger.With().Str("func", "seedReservedClient").Logger()

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		log.Warn().Msg("Config.ClientID is empty; skipping reserved client seed")
		return
	}

	if existing, err := storageProvider.GetClientByClientID(ctx, clientID); err == nil && existing != nil {
		log.Debug().Msg("reserved client already present; seed is a no-op")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.ClientSecret), reservedClientSecretCost)
	if err != nil {
		log.Error().Err(err).Msg("failed to bcrypt Config.ClientSecret; skipping reserved client seed")
		return
	}

	if _, err := storageProvider.AddClient(ctx, &schemas.Client{
		ClientID:                clientID,
		Kind:                    constants.ClientKindInteractive,
		Name:                    "Reserved Interactive Client",
		ClientSecret:            string(hash),
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodClientSecretBasic,
		GrantTypes:              constants.GrantTypeAuthorizationCode + "," + constants.GrantTypeRefreshToken,
		IsActive:                true,
	}); err != nil {
		// Read-only / no-write-path instance, or a concurrent seed that lost the
		// unique-index race: log and skip rather than fatal.
		log.Warn().Err(err).Msg("failed to seed reserved client (read-only instance or concurrent seed); continuing")
		return
	}

	log.Info().Str("client_id", clientID).Msg("seeded reserved interactive client")
}
