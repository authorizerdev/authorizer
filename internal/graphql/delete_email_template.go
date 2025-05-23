package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeleteEmailTemplate is the method to delete an email template.
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteEmailTemplate(ctx context.Context, params *model.DeleteEmailTemplateRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "DeleteEmailTemplate").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if strings.TrimSpace(params.ID) == "" {
		return nil, fmt.Errorf("email template ID required")
	}

	log = log.With().Str("emailTemplateID", params.ID).Logger()

	emailTemplate, err := g.StorageProvider.GetEmailTemplateByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get email template by id")
		return nil, err
	}

	err = g.StorageProvider.DeleteEmailTemplate(ctx, emailTemplate)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete email template")
		return nil, err
	}

	return &model.Response{
		Message: "Email templated deleted successfully",
	}, nil
}
