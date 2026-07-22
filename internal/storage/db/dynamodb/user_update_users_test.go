package dynamodb

import (
	"context"
	"errors"
	"testing"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestUpdateUsersRejectsEmptyIDs proves UpdateUsers refuses an empty ids slice
// with schemas.ErrUpdateUsersEmptyIDs instead of silently updating every user
// row. The guard returns before any DB access, so no live connection is required.
func TestUpdateUsersRejectsEmptyIDs(t *testing.T) {
	p := &provider{}
	for _, ids := range [][]string{nil, {}} {
		err := p.UpdateUsers(context.Background(), map[string]interface{}{"given_name": "x"}, ids)
		if !errors.Is(err, schemas.ErrUpdateUsersEmptyIDs) {
			t.Fatalf("UpdateUsers(ids=%v) must return ErrUpdateUsersEmptyIDs, got: %v", ids, err)
		}
	}
}
