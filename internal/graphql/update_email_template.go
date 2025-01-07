package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// UpdateEmailTemplate  for update email template mutation
// Permission: authorizer:admin
func (g *graphqlProvider) UpdateEmailTemplate(ctx context.Context, params *model.UpdateEmailTemplateRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "UpdateEmailTemplate").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	emailTemplate, err := g.StorageProvider.GetEmailTemplateByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetEmailTemplateByID")
		return nil, err
	}

	emailTemplateDetails := &schemas.EmailTemplate{
		ID:        emailTemplate.ID,
		Key:       emailTemplate.ID,
		EventName: emailTemplate.EventName,
		CreatedAt: refs.Int64Value(emailTemplate.CreatedAt),
	}

	if params.EventName != nil && emailTemplateDetails.EventName != refs.StringValue(params.EventName) {
		if isValid := validators.IsValidEmailTemplateEventName(refs.StringValue(params.EventName)); !isValid {
			log.Debug().Str("event_name", refs.StringValue(params.EventName)).Msg("invalid event name")
			return nil, fmt.Errorf("invalid event name %s", refs.StringValue(params.EventName))
		}
		emailTemplateDetails.EventName = refs.StringValue(params.EventName)
	}

	if params.Subject != nil && emailTemplateDetails.Subject != refs.StringValue(params.Subject) {
		if strings.TrimSpace(refs.StringValue(params.Subject)) == "" {
			log.Debug().Msg("empty subject not allowed")
			return nil, fmt.Errorf("empty subject not allowed")
		}
		emailTemplateDetails.Subject = refs.StringValue(params.Subject)
	}

	if params.Template != nil && emailTemplateDetails.Template != refs.StringValue(params.Template) {
		if strings.TrimSpace(refs.StringValue(params.Template)) == "" {
			log.Debug().Msg("empty template not allowed")
			return nil, fmt.Errorf("empty template not allowed")
		}
		emailTemplateDetails.Template = refs.StringValue(params.Template)
	}

	if params.Design != nil && emailTemplateDetails.Design != refs.StringValue(params.Design) {
		if strings.TrimSpace(refs.StringValue(params.Design)) == "" {
			log.Debug().Msg("empty design not allowed")
			return nil, fmt.Errorf("empty design not allowed")
		}
		emailTemplateDetails.Design = refs.StringValue(params.Design)
	}

	_, err = g.StorageProvider.UpdateEmailTemplate(ctx, emailTemplateDetails)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateEmailTemplate")
		return nil, err
	}

	return &model.Response{
		Message: `Email template updated successfully.`,
	}, nil
}
