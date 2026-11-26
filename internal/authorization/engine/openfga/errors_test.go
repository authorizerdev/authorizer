package openfga

import (
	"errors"
	"testing"

	serverErrors "github.com/openfga/openfga/pkg/server/errors"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// TestOpErr: OpenFGA's own internal faults (datastore failures surface as
// serverErrors.InternalError) are marked internal so transports mask them;
// validation errors about the caller's input are not (issue #356).
func TestOpErr(t *testing.T) {
	dsErr := serverErrors.NewInternalError("", errors.New(`pq: relation "tuple" does not exist`))
	err := opErr("openfga.Check", dsErr)
	assert.True(t, internalerr.Is(err))
	assert.Equal(t, "openfga", internalerr.Source(err))

	valErr := serverErrors.TypeNotFound("document")
	err = opErr("openfga.WriteTuples", valErr)
	assert.False(t, internalerr.Is(err))
	assert.Contains(t, err.Error(), "type 'document' not found")
}
