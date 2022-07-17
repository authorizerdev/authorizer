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

// EmailTemplatesResolver resolver for getting the list of email templates based on pagination
func EmailTemplatesResolver(ctx context.Context, params *model.PaginatedInput) (*model.EmailTemplates, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	emailTemplates, err := db.Provider.ListEmailTemplate(ctx, pagination)
	if err != nil {
		log.Debug("failed to get email templates: ", err)
		return nil, err
	}
	return emailTemplates, nil
}
