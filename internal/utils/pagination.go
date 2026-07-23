package utils

import (
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// GetPagination helps getting pagination data from a pagination input
// also returns default limit and offset if pagination data is not present
func GetPagination(pagination *model.PaginationRequest) *model.Pagination {
	limit := int64(constants.DefaultLimit)
	page := int64(1)
	if pagination != nil {
		if pagination.Limit != nil {
			limit = *pagination.Limit
		}

		if pagination.Page != nil {
			page = *pagination.Page
		}
	}

	return &model.Pagination{
		Limit:  limit,
		Offset: (page - 1) * limit,
		Page:   page,
	}
}
