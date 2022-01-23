package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	tokenHelper "github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// IsValidJwtResolver resolver to return if given jwt is valid
func IsValidJwtResolver(ctx context.Context, params *model.IsValidJWTQueryInput) (*model.ValidJWTResponse, error) {
	claims, err := tokenHelper.VerifyJWTToken(params.Jwt)
	if err != nil {
		return nil, err
	}

	claimRoleInterface := claims[envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtRoleClaim)].([]interface{})
	claimRoles := []string{}
	for _, v := range claimRoleInterface {
		claimRoles = append(claimRoles, v.(string))
	}

	if params != nil && params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				return nil, fmt.Errorf(`unauthorized`)
			}
		}
	}

	return &model.ValidJWTResponse{
		Valid:   true,
		Message: "Valid JWT",
	}, nil
}
