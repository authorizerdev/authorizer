package interceptors

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/service"
)

// kindToCode maps a transport-neutral service.ErrorKind onto the gRPC status
// code. grpc-gateway then derives the REST HTTP status from this code
// (InvalidArgument->400, Unauthenticated->401, PermissionDenied->403,
// NotFound->404, FailedPrecondition->400, AlreadyExists->409, Internal->500).
func kindToCode(kind service.ErrorKind) codes.Code {
	switch kind {
	case service.KindInvalidArgument:
		return codes.InvalidArgument
	case service.KindUnauthenticated:
		return codes.Unauthenticated
	case service.KindPermissionDenied:
		return codes.PermissionDenied
	case service.KindNotFound:
		return codes.NotFound
	case service.KindFailedPrecondition:
		return codes.FailedPrecondition
	case service.KindTooManyRequests:
		return codes.ResourceExhausted
	case service.KindAlreadyExists:
		return codes.AlreadyExists
	default:
		return codes.Internal
	}
}

// ErrorMap is the innermost unary interceptor. It translates errors returned by
// the handler into proper gRPC status errors so both gRPC and REST clients see
// a meaningful status code instead of codes.Unknown / HTTP 500 for everything.
//
// Placement: this MUST be the last (innermost) interceptor in the chain so it
// wraps the handler directly. Validation failures produced by the protovalidate
// interceptor live one level out and already carry InvalidArgument, so they
// never reach here. Any error that is already a gRPC status (e.g. the
// Unimplemented stubs) is passed through untouched; typed service.Error values
// are mapped by Kind; everything else is treated as Internal.
func ErrorMap() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		// Typed service error: map its Kind onto a gRPC code, preserving the
		// message the service chose to surface.
		var se *service.Error
		if errors.As(err, &se) {
			return resp, status.Error(kindToCode(se.Kind), se.Error())
		}

		// Already a gRPC status error (Unimplemented stubs, or a status
		// produced upstream): leave it as-is.
		if _, ok := status.FromError(err); ok {
			return resp, err
		}

		// Unclassified error: treat as internal. The handler already decided
		// what message text is safe to surface, so we forward it verbatim.
		return resp, status.Error(codes.Internal, err.Error())
	}
}
