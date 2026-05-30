// Package gateway translates REST (`/v1/*`) calls into in-process gRPC
// method invocations using grpc-gateway. The resulting http.Handler is
// mounted into the existing Gin router so middleware (CORS, security
// headers, rate limit, logging) is shared.
//
// We deliberately use the *in-process* dialer style: the gateway dials the
// running grpc.Server via bufconn rather than making a real network hop.
// This avoids the latency and TLS-cert plumbing of a loopback gRPC call,
// and removes the need to keep two ports in sync.
package gateway

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// bufconn size; large enough that in-process gateway calls never block.
const bufSize = 1 << 20

// Handler builds an http.Handler that translates `/v1/*` REST calls into
// gRPC calls against the supplied in-process *grpc.Server. Returns the
// handler and a cleanup function the caller invokes at shutdown.
func Handler(ctx context.Context, grpcSrv *grpc.Server) (http.Handler, func(), error) {
	lis := bufconn.Listen(bufSize)

	// Serve gRPC over the bufconn in a goroutine; the existing TCP
	// listener (started by grpcsrv.Server.Run) is the public entry point —
	// this listener only carries in-process gateway traffic.
	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = lis.Close()
		return nil, nil, err
	}

	mux := runtime.NewServeMux(
		// Use snake_case proto field names (UseProtoNames=true) over the
		// camelCase default — keeps payloads aligned with the existing
		// GraphQL surface. DiscardUnknown tolerates older clients sending
		// fields that have since been removed.
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	if err := registerAll(ctx, mux, conn); err != nil {
		_ = conn.Close()
		_ = lis.Close()
		return nil, nil, err
	}

	cleanup := func() {
		_ = conn.Close()
		_ = lis.Close()
	}
	return mux, cleanup, nil
}

func registerAll(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// Single Authorizer service. As more services land (admin-side ones
	// that today stay GraphQL-only), add their registrar here.
	return authorizerv1.RegisterAuthorizerHandler(ctx, mux, conn)
}
