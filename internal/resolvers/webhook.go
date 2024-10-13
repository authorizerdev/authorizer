package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	log "github.com/sirupsen/logrus"
)

// WebhookResolver resolver for getting webhook by identifier
func WebhookResolver(ctx context.Context, params model.WebhookRequest) (*model.Webhook, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	webhook, err := db.Provider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug("error getting webhook: ", err)
		return nil, err
	}
	return webhook, nil
}
