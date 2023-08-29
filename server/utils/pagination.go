package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// GetPagination helps getting pagination data from paginated input
// also returns default limit and offset if pagination data is not present
func GetPagination(paginatedInput *model.PaginatedInput) *model.Pagination {
	limit := int64(constants.DefaultLimit)
	page := int64(1)
	if paginatedInput != nil && paginatedInput.Pagination != nil {
		if paginatedInput.Pagination.Limit != nil {
			limit = *paginatedInput.Pagination.Limit
		}

		if paginatedInput.Pagination.Page != nil {
			page = *paginatedInput.Pagination.Page
		}
	}

	return &model.Pagination{
		Limit:  limit,
		Offset: (page - 1) * limit,
		Page:   page,
	}
}
