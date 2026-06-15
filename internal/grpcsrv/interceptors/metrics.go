package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// Metrics returns a unary interceptor that records
// authorizer_api_operations_total for every RPC, labeled by transport protocol,
// the short method name, and ok|error status. REST calls arrive through the
// grpc-gateway, which sets x-authorizer-transport=rest; absent that marker the
// call came over pure gRPC. This single interceptor therefore covers both the
// gRPC and REST surfaces.
func Metrics() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)

		protocol := constants.ProtocolGRPC
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-authorizer-transport"); len(vals) > 0 && vals[0] == constants.ProtocolREST {
				protocol = constants.ProtocolREST
			}
		}
		st := metrics.OperationStatusOK
		if status.Code(err) != codes.OK {
			st = metrics.OperationStatusError
		}
		metrics.RecordAPIOperation(protocol, shortMethod(info.FullMethod), st)
		return resp, err
	}
}

// shortMethod extracts the trailing method name from a gRPC FullMethod, e.g.
// "/authorizer.v1.AuthorizerAdminService/AdminMeta" -> "AdminMeta".
func shortMethod(fullMethod string) string {
	if i := strings.LastIndex(fullMethod, "/"); i >= 0 {
		return fullMethod[i+1:]
	}
	return fullMethod
}
