package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// MetaResolver is a resolver for meta query
func MetaResolver(ctx context.Context) (*model.Meta, error) {
	metaInfo := utils.GetMetaInfo()
	return &metaInfo, nil
}
