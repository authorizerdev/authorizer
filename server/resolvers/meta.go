package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Meta(ctx context.Context) (*model.Meta, error) {
	metaInfo := utils.GetMetaInfo()
	return &metaInfo, nil
}
