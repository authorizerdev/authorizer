package resolvers

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
	log "github.com/sirupsen/logrus"
)

// AddEmailTemplateResolver resolver for add email template mutation
func AddEmailTemplateResolver(ctx context.Context, params model.AddEmailTemplateRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if !validators.IsValidEmailTemplateEventName(params.EventName) {
		log.Debug("Invalid Event Name: ", params.EventName)
		return nil, fmt.Errorf("invalid event name %s", params.EventName)
	}

	if strings.TrimSpace(params.Subject) == "" {
		return nil, fmt.Errorf("empty subject not allowed")
	}

	if strings.TrimSpace(params.Template) == "" {
		return nil, fmt.Errorf("empty template not allowed")
	}

	var design string

	if params.Design == nil {
		design = ""
	}

	_, err = db.Provider.AddEmailTemplate(ctx, models.EmailTemplate{
		EventName: params.EventName,
		Template:  params.Template,
		Subject:   params.Subject,
		Design:    design,
	})
	if err != nil {
		log.Debug("Failed to add email template: ", err)
		return nil, err
	}

	return &model.Response{
		Message: `Email template added successfully`,
	}, nil
}
