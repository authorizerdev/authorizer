package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// EmailTemplates is the method to get all email templates.
// Permissions: authorizer:admin
func (g *graphqlProvider) EmailTemplates(ctx context.Context, params *model.PaginatedInput) (*model.EmailTemplates, error) {
	log := g.Log.With().Str("func", "EmailTemplates").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)
	emailTemplates, err := g.StorageProvider.ListEmailTemplate(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get email templates")
		return nil, err
	}
	return emailTemplates, nil
}
