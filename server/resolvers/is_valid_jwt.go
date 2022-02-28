package resolvers

import (
	"context"
	"errors"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	tokenHelper "github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// IsValidJwtResolver resolver to return if given jwt is valid
func IsValidJwtResolver(ctx context.Context, params *model.IsValidJWTQueryInput) (*model.ValidJWTResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	token, err := token.GetAccessToken(gc)

	if token == "" || err != nil {
		if params != nil && *params.Jwt != "" {
			token = *params.Jwt
		} else {
			return nil, errors.New("no jwt provided via cookie / header / params")
		}
	}

	claims, err := tokenHelper.ParseJWTToken(token)
	if err != nil {
		return nil, err
	}

	claimRoleInterface := claims[envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtRoleClaim)].([]interface{})
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
