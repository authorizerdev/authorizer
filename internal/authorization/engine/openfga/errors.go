package openfga

import (
	"errors"
	"fmt"

	serverErrors "github.com/openfga/openfga/pkg/server/errors"

	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// errorSource is the internalerr source recorded for OpenFGA engine faults.
const errorSource = "openfga"

// opErr prefixes an error returned by the embedded OpenFGA server with the
// engine operation, and marks it internal when OpenFGA itself classified it as
// an internal fault (serverErrors.InternalError: a datastore failure or any
// other unexpected server error). Those are masked by the transports and
// logged with an error_id.
//
// Validation errors (unknown type/relation, invalid tuple, invalid model) are
// left unmarked: they describe the caller's own input and are deliberately
// shown to the admin who sent it.
func opErr(op string, err error) error {
	wrapped := fmt.Errorf("%s: %w", op, err)
	var ie serverErrors.InternalError
	if errors.As(err, &ie) {
		return internalerr.Wrap(errorSource, wrapped)
	}
	return wrapped
}
