package http_handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage"
)

var nopLog = zerolog.Nop()

// TestGraphQLErrorPresenter_MasksStorageError is the regression guard for
// issue #356: raw driver text from the storage layer must never reach a
// GraphQL client. The client gets a generic message with an error_id and
// code INTERNAL; the original error is logged under the same error_id.
func TestGraphQLErrorPresenter_MasksStorageError(t *testing.T) {
	const driverMsg = "mssql: Cannot insert duplicate key row in object 'dbo.authorizer_users' with unique index 'idx_email'"
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	err := fmt.Errorf("signup: %w", &storage.Error{Err: errors.New(driverMsg)})
	gqlErr := graphQLErrorPresenter(&log)(context.Background(), err)
	require.NotNil(t, gqlErr)

	assert.NotContains(t, gqlErr.Message, "mssql")
	assert.NotContains(t, gqlErr.Message, "authorizer_users")
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, gqlErr.Message)
	require.NotNil(t, gqlErr.Extensions)
	assert.Equal(t, "INTERNAL", gqlErr.Extensions["code"])

	logged := buf.String()
	assert.Contains(t, logged, driverMsg, "original error must be logged server-side")
	assert.Contains(t, logged, `"level":"error"`)
	// Correlation: the id the client sees is the one in the log line.
	id := gqlErr.Message[len("internal server error (error_id: ") : len(gqlErr.Message)-1]
	assert.Contains(t, logged, `"error_id":"`+id+`"`)
}

// TestGraphQLErrorPresenter_StorageNotFoundLogsAtWarn: an unclassified
// not-found is still masked (services should classify it), but it is not an
// outage, so it must not page anyone at ERROR.
func TestGraphQLErrorPresenter_StorageNotFoundLogsAtWarn(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	gqlErr := graphQLErrorPresenter(&log)(context.Background(), &storage.Error{Err: gorm.ErrRecordNotFound})
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, gqlErr.Message)
	assert.Contains(t, buf.String(), `"level":"warn"`)
}

// TestGraphQLErrorPresenter_TypedWrappingStorageUsesTypedMessage: GraphQL must
// agree with gRPC's ErrorMap — a typed error that wraps a storage error shows
// only the typed message, never the wrapped driver text.
func TestGraphQLErrorPresenter_TypedWrappingStorageUsesTypedMessage(t *testing.T) {
	err := fmt.Errorf("%w: %w", service.NotFound("user not found"), &storage.Error{Err: errors.New("mongo: no documents in result")})
	gqlErr := graphQLErrorPresenter(&nopLog)(context.Background(), err)
	assert.Equal(t, "user not found", gqlErr.Message)
	assert.Equal(t, "NOT_FOUND", gqlErr.Extensions["code"])
}

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
			gqlErr := graphQLErrorPresenter(&nopLog)(context.Background(), tc.err)
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
	gqlErr := graphQLErrorPresenter(&nopLog)(context.Background(), errors.New("boom"))
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
