package schemas

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/stretchr/testify/assert"
)

// TestAsAPIUserMapsHasSkippedMFASetupAt proves the one piece of real logic
// Task 1 adds: User.HasSkippedMFASetupAt (storage) must pass through to
// model.User.HasSkippedMfaSetupAt (API) unchanged, in both directions —
// nil ("never skipped") and set (explicit skip timestamp).
func TestAsAPIUserMapsHasSkippedMFASetupAt(t *testing.T) {
	skippedAt := refs.NewInt64Ref(1700000000)

	user := &User{ID: "user-1", HasSkippedMFASetupAt: skippedAt}
	assert.Equal(t, skippedAt, user.AsAPIUser().HasSkippedMfaSetupAt)

	neverSkipped := &User{ID: "user-2"}
	assert.Nil(t, neverSkipped.AsAPIUser().HasSkippedMfaSetupAt)
}
