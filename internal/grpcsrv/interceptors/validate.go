package interceptors

import (
	"context"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Validate returns a unary interceptor that runs protovalidate on every
// inbound request that's a proto.Message. Failures convert to
// codes.InvalidArgument with a human-readable detail.
//
// The validator is built once at startup and shared across requests
// (protovalidate.Validator is safe for concurrent use).
func Validate() (grpc.UnaryServerInterceptor, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if msg, ok := req.(proto.Message); ok {
			if err := v.Validate(msg); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "%v", err)
			}
		}
		return handler(ctx, req)
	}, nil
}
