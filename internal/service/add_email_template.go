package service

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

// AddEmailTemplate is the method to add email template.
// Permissions: authorizer:admin
func (s *service) AddEmailTemplate(ctx context.Context, params *model.AddEmailTemplateRequest) (*model.Response, error) {
	log := s.Log.With().Str("func", "AddEmailTemplate").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if !validators.IsValidEmailTemplateEventName(params.EventName) {
		log.Debug().Str("EventName", params.EventName).Msg("Invalid Event Name")
		return nil, fmt.Errorf("invalid event name %s", params.EventName)
	}

	if strings.TrimSpace(params.Subject) == "" {
		log.Debug().Msg("subject is missing")
		return nil, fmt.Errorf("empty subject not allowed")
	}

	if strings.TrimSpace(params.Template) == "" {
		log.Debug().Msg("template is missing")
		return nil, fmt.Errorf("empty template not allowed")
	}

	var design string

	if params.Design == nil || strings.TrimSpace(refs.StringValue(params.Design)) == "" {
		design = ""
	}

	_, err = s.StorageProvider.AddEmailTemplate(ctx, &schemas.EmailTemplate{
		EventName: params.EventName,
		Template:  params.Template,
		Subject:   params.Subject,
		Design:    design,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add email template in db")
		return nil, err
	}

	return &model.Response{
		Message: `Email template added successfully`,
	}, nil
}
