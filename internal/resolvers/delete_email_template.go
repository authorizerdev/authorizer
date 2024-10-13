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

// DeleteEmailTemplateResolver resolver to delete email template and its relevant logs
func DeleteEmailTemplateResolver(ctx context.Context, params model.DeleteEmailTemplateRequest) (*model.Response, error) {
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
		log.Debug("email template is required")
		return nil, fmt.Errorf("email template ID required")
	}

	log := log.WithField("email_template_id", params.ID)

	emailTemplate, err := db.Provider.GetEmailTemplateByID(ctx, params.ID)
	if err != nil {
		log.Debug("failed to get email template: ", err)
		return nil, err
	}

	err = db.Provider.DeleteEmailTemplate(ctx, emailTemplate)
	if err != nil {
		log.Debug("failed to delete email template: ", err)
		return nil, err
	}

	return &model.Response{
		Message: "Email templated deleted successfully",
	}, nil
}
