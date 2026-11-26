package openfga

import (
	"errors"
	"testing"

	serverErrors "github.com/openfga/openfga/pkg/server/errors"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// TestOpErr: OpenFGA's own internal faults (datastore failures surface as
// serverErrors.InternalError) are marked internal so transports mask them and
// log them with an error_id; validation errors about the caller's input are
// not (issue #356). InternalError.Error() is already OpenFGA's generic
// "Internal Server Error", so the gain is not text redaction: it is that the
// error carries a non-standard gRPC status (code 4000) that ErrorMap would
// otherwise pass straight through to clients.
func TestOpErr(t *testing.T) {
	dsErr := serverErrors.NewInternalError("", errors.New(`pq: relation "tuple" does not exist`))
	err := opErr("openfga.Check", dsErr)
	assert.True(t, internalerr.Is(err))
	assert.Equal(t, "openfga", internalerr.Source(err))
	assert.NotContains(t, err.Error(), "pq:", "OpenFGA already hides the datastore text")

	valErr := serverErrors.TypeNotFound("document")
	err = opErr("openfga.WriteTuples", valErr)
	assert.False(t, internalerr.Is(err))
	assert.Contains(t, err.Error(), "type 'document' not found")
}
