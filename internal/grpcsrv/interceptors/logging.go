package interceptors

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logging returns a unary interceptor that emits one structured log line per
// RPC at info level (or error/warn when the status code reflects failure).
// Method name, duration, and gRPC code are always present.
func Logging(log *zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)

		code := status.Code(err)
		evt := log.Info()
		switch code {
		case codes.OK:
			// stays info
		case codes.Internal, codes.Unknown, codes.DataLoss:
			evt = log.Error()
		default:
			evt = log.Warn()
		}
		evt.
			Str("method", info.FullMethod).
			Str("code", code.String()).
			Dur("duration", dur).
			Err(err).
			Msg("grpc")
		return resp, err
	}
}
