package resolvers

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
	log "github.com/sirupsen/logrus"
)

// TODO add template validator

// UpdateEmailTemplateResolver resolver for update email template mutation
func UpdateEmailTemplateResolver(ctx context.Context, params model.UpdateEmailTemplateRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	emailTemplate, err := db.Provider.GetEmailTemplateByID(ctx, params.ID)
	if err != nil {
		log.Debug("failed to get email template: ", err)
		return nil, err
	}

	emailTemplateDetails := models.EmailTemplate{
		ID:        emailTemplate.ID,
		Key:       emailTemplate.ID,
		EventName: emailTemplate.EventName,
		CreatedAt: refs.Int64Value(emailTemplate.CreatedAt),
	}

	if params.EventName != nil && emailTemplateDetails.EventName != refs.StringValue(params.EventName) {
		if isValid := validators.IsValidEmailTemplateEventName(refs.StringValue(params.EventName)); !isValid {
			log.Debug("invalid event name: ", refs.StringValue(params.EventName))
			return nil, fmt.Errorf("invalid event name %s", refs.StringValue(params.EventName))
		}
		emailTemplateDetails.EventName = refs.StringValue(params.EventName)
	}

	if params.Template != nil && emailTemplateDetails.Template != refs.StringValue(params.Template) {
		if strings.TrimSpace(refs.StringValue(params.Template)) == "" {
			log.Debug("empty template not allowed")
			return nil, fmt.Errorf("empty template not allowed")
		}
		emailTemplateDetails.Template = refs.StringValue(params.Template)
	}

	_, err = db.Provider.UpdateEmailTemplate(ctx, emailTemplateDetails)
	if err != nil {
		return nil, err
	}

	return &model.Response{
		Message: `Email template updated successfully.`,
	}, nil
}
