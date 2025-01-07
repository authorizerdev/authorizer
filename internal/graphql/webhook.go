package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Webhook is the method to get webhook details
// Permission: authorizer:admin
func (g *graphqlProvider) Webhook(ctx context.Context, params *model.WebhookRequest) (*model.Webhook, error) {
	log := g.Log.With().Str("func", "Webhook").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	webhook, err := g.StorageProvider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetWebhookByID")
		return nil, err
	}
	return webhook, nil
}
