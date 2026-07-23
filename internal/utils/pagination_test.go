package utils

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

func TestGetPagination(t *testing.T) {
	limit10 := int64(10)
	page2 := int64(2)

	cases := []struct {
		name       string
		pagination *model.PaginationRequest
		wantLimit  int64
		wantPage   int64
		wantOffset int64
	}{
		{"nil pagination uses defaults", nil, int64(constants.DefaultLimit), 1, 0},
		{"empty struct uses defaults", &model.PaginationRequest{}, int64(constants.DefaultLimit), 1, 0},
		{"explicit limit and page", &model.PaginationRequest{Limit: &limit10, Page: &page2}, 10, 2, 10},
		{"limit only, default page", &model.PaginationRequest{Limit: &limit10}, 10, 1, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := GetPagination(c.pagination)
			if got.Limit != c.wantLimit || got.Page != c.wantPage || got.Offset != c.wantOffset {
				t.Errorf("GetPagination() = {Limit:%d Page:%d Offset:%d}, want {Limit:%d Page:%d Offset:%d}",
					got.Limit, got.Page, got.Offset, c.wantLimit, c.wantPage, c.wantOffset)
			}
		})
	}
}
