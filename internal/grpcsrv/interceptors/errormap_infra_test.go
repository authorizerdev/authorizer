package interceptors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	serverErrors "github.com/openfga/openfga/pkg/server/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// TestErrorMap_MasksInfraProviderErrors extends the issue #356 guard beyond
// storage: a Redis/memory-store fault, an OpenFGA datastore fault (which
// itself carries a gRPC status that must NOT be passed through), a JWT
// signing-key fault and an SMTP fault are all masked with an error_id that is
// logged next to the original.
func TestErrorMap_MasksInfraProviderErrors(t *testing.T) {
	cases := map[string]error{
		"memory_store": internalerr.Wrap("memory_store", errors.New("dial tcp 10.0.0.5:6379: connect: connection refused")),
		"openfga":      internalerr.Wrap("openfga", fmt.Errorf("openfga.Check: %w", serverErrors.NewInternalError("", errors.New(`pq: relation "tuple" does not exist`)))),
		"token":        internalerr.Wrap("token", errors.New("asn1: structure error: tags don't match")),
		"email":        internalerr.Wrap("email", errors.New("535 5.7.8 auth failed for smtp.internal:587")),
	}
	for source, handlerErr := range cases {
		t.Run(source, func(t *testing.T) {
			var buf bytes.Buffer
			log := zerolog.New(&buf)
			_, err := ErrorMap(&log)(context.Background(), nil, info("/authorizer.v1.AuthorizerService/Login"), func(_ context.Context, _ any) (any, error) {
				return nil, fmt.Errorf("login: %w", handlerErr)
			})
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.Internal, st.Code())
			assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, st.Message())

			id := st.Message()[len("internal server error (error_id: ") : len(st.Message())-1]
			logged := buf.String()
			assert.Contains(t, logged, `"error_id":"`+id+`"`)
			assert.Contains(t, logged, `"error_source":"`+source+`"`)
			assert.Contains(t, logged, "/authorizer.v1.AuthorizerService/Login")
		})
	}
}

// TestErrorMap_DeliberateUntypedMessageUnchanged: unmarked untyped errors are
// deliberate user-facing messages and keep their text.
func TestErrorMap_DeliberateUntypedMessageUnchanged(t *testing.T) {
	err := runErrorMap(t, errors.New("unauthorized"))
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "unauthorized", status.Convert(err).Message())
}
