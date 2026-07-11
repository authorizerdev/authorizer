package http_handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/service"
)

// TestGraphQLErrorPresenter_TypedServiceError guards the contract that
// authorizer-react (via authorizer-js) relies on to detect the TOTP lockout
// without matching on message text: a typed service.Error must surface a
// stable extensions.code, with the message left untouched.
func TestGraphQLErrorPresenter_TypedServiceError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"too many requests", service.TooManyRequests("too many failed attempts, please try again later"), "TOO_MANY_REQUESTS"},
		{"invalid argument", service.InvalidArgument("bad input"), "INVALID_ARGUMENT"},
		{"unauthenticated", service.Unauthenticated("no session"), "UNAUTHENTICATED"},
		{"permission denied", service.PermissionDenied("nope"), "PERMISSION_DENIED"},
		{"not found", service.NotFound("missing"), "NOT_FOUND"},
		{"failed precondition", service.FailedPrecondition("bad state"), "FAILED_PRECONDITION"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gqlErr := graphQLErrorPresenter(context.Background(), tc.err)
			require.NotNil(t, gqlErr)
			assert.Equal(t, tc.err.Error(), gqlErr.Message, "message must be preserved byte-for-byte")
			require.NotNil(t, gqlErr.Extensions)
			assert.Equal(t, tc.code, gqlErr.Extensions["code"])
		})
	}
}

// TestGraphQLErrorPresenter_UntypedError guards that plain errors (storage
// failures, token-creation failures, etc.) keep today's behaviour exactly:
// no extensions.code is attached, so no existing client can be surprised by
// a new field appearing where it never did before.
func TestGraphQLErrorPresenter_UntypedError(t *testing.T) {
	gqlErr := graphQLErrorPresenter(context.Background(), errors.New("boom"))
	require.NotNil(t, gqlErr)
	assert.Equal(t, "boom", gqlErr.Message)
	if gqlErr.Extensions != nil {
		_, hasCode := gqlErr.Extensions["code"]
		assert.False(t, hasCode, "untyped errors must not gain a code extension")
	}
}

func TestKindToGraphQLCode_DefaultsToInternal(t *testing.T) {
	assert.Equal(t, "INTERNAL", kindToGraphQLCode(service.ErrorKind(99)))
}
