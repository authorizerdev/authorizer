// Package interceptors contains the gRPC server interceptors shared across
// Authorizer services. They run in this order (outermost first):
//
//   recovery → logging → validate → ... (auth, permission, audit — added per service in later PRs)
//
// Recovery is outermost so it catches panics raised by anything later, and
// converts them to a clean codes.Internal status instead of crashing the
// server.
package interceptors

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Recovery returns a unary interceptor that converts handler panics into a
// codes.Internal error and logs the stack at error level. The stack stays
// server-side — clients only see a generic "internal error" message.
//
// Security: the panic value is logged as TYPE only, never its full content.
// A handler can panic with a request struct that includes credentials
// (Password, RefreshToken, OTP, AdminSecret, ...); dumping the value via
// .Interface() would write those credentials to the log stream verbatim.
// Logging just the type lets ops correlate the panic with the stack without
// exposing payload fields. (Security audit finding H2.)
func Recovery(log *zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Str("method", info.FullMethod).
					Str("panic_type", fmt.Sprintf("%T", r)).
					Bytes("stack", debug.Stack()).
					Msg("gRPC handler panicked")
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
