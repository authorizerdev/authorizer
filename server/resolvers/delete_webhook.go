package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	log "github.com/sirupsen/logrus"
)

// DeleteWebhookResolver resolver to delete webhook and its relevant logs
func DeleteWebhookResolver(ctx context.Context, params model.WebhookRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if params.ID == "" {
		log.Debug("webhookID is required")
		return nil, fmt.Errorf("webhook ID required")
	}

	log := log.WithField("webhook_id", params.ID)

	webhook, err := db.Provider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug("failed to get webhook: ", err)
		return nil, err
	}

	err = db.Provider.DeleteWebhook(ctx, webhook)
	if err != nil {
		log.Debug("failed to delete webhook: ", err)
		return nil, err
	}

	return &model.Response{
		Message: "Webhook deleted successfully",
	}, nil
}
