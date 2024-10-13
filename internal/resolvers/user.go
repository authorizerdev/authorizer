package resolvers

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UserResolver is a resolver for user query
// This is admin only query
func UserResolver(ctx context.Context, params model.GetUserRequest) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}
	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return nil, fmt.Errorf("unauthorized")
	}
	// Try getting user by ID
	if params.ID != nil && strings.Trim(*params.ID, " ") != "" {
		res, err := db.Provider.GetUserByID(ctx, *params.ID)
		if err != nil {
			log.Debug("Failed to get users by ID: ", err)
			return nil, err
		}
		return res.AsAPIUser(), nil
	}
	// Try getting user by email
	if params.Email != nil && strings.Trim(*params.Email, " ") != "" {
		res, err := db.Provider.GetUserByEmail(ctx, *params.Email)
		if err != nil {
			log.Debug("Failed to get users by email: ", err)
			return nil, err
		}
		return res.AsAPIUser(), nil
	}
	// Return error if no params are provided
	return nil, fmt.Errorf("invalid params, user id or email is required")
}
