package http_handlers

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/service"
)

// publicErrorMessage returns the text that is safe to put in a client-facing
// response (JSON body or redirect URL) for err. A storage-originated error the
// service did not classify is replaced with a generic message carrying an
// error_id, and the original is logged under that id — the same rule the
// GraphQL presenter and gRPC ErrorMap apply. Every other error's text is
// returned unchanged, so deliberate user-facing messages stay intact.
func publicErrorMessage(log *zerolog.Logger, err error) string {
	msg, errorID, masked := service.MaskInternalError(err)
	if !masked {
		return err.Error()
	}
	service.MaskedErrorEvent(log, err, errorID).Msg("http: masked internal error")
	return msg
}
