package interceptors

import (
	"context"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
	"github.com/authorizerdev/authorizer/internal/token"
)

const (
	adminServiceName  = "authorizer.v1.AuthorizerAdminService"
	publicServiceName = "authorizer.v1.AuthorizerService"
	sessionMethodName = "Session"
	// adminLoginMethodName is the ONLY AuthorizerAdminService RPC allowed to be
	// reached without super-admin auth: it establishes that auth. Every other
	// admin RPC marked `public` is a mistake and must still require super-admin
	// (see the bypass guard in Auth).
	adminLoginMethodName = "AdminLogin"
)

// infrastructureServices are gRPC surfaces registered alongside Authorizer that
// must not go through Authorizer auth (k8s probes, reflection).
var infrastructureServices = map[string]struct{}{
	"grpc.health.v1.Health":                    {},
	"grpc.reflection.v1alpha.ServerReflection": {},
	"grpc.reflection.v1.ServerReflection":      {},
}

var methodDescCache sync.Map // map[string]protoreflect.MethodDescriptor

// Auth returns a unary interceptor that enforces proto-declared auth policy.
func Auth(tp token.Provider) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		methodDesc, ok := methodDescriptor(info.FullMethod)
		if !ok {
			// No registered proto descriptor (unknown path) — not an Authorizer RPC.
			return handler(ctx, req)
		}
		serviceName := string(methodDesc.Parent().FullName())
		if _, infra := infrastructureServices[serviceName]; infra {
			return handler(ctx, req)
		}
		if shouldRejectUnlistedService(serviceName) {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		// The `public` proto annotation grants no-auth access. Honor it for the
		// public service's methods and for the admin auth-bootstrap RPC
		// (AdminLogin), which must be reachable before super-admin auth exists.
		// Any OTHER AuthorizerAdminService method mistakenly marked `public`
		// must NOT skip admin auth — it falls through to the super-admin check
		// below (mirroring the Session RPC's explicit service guard, closing the
		// latent footgun where a future admin RPC is accidentally made public).
		if isPublicMethod(methodDesc) &&
			(serviceName == publicServiceName ||
				(serviceName == adminServiceName && string(methodDesc.Name()) == adminLoginMethodName)) {
			return handler(ctx, req)
		}
		if tp == nil {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}

		meta := transport.MetaFromGRPC(ctx)
		gc := &gin.Context{Request: meta.Request}

		if serviceName == adminServiceName {
			if !tp.IsSuperAdmin(gc) {
				return nil, status.Error(codes.Unauthenticated, "unauthorized")
			}
			ctx = authctx.WithPrincipal(ctx, &authctx.Principal{IsSuperAdmin: true})
			return handler(ctx, req)
		}

		// Session rotates the browser session cookie only; bearer tokens are ignored.
		// Guard on publicServiceName to prevent a future method named "Session" on
		// another service from inheriting cookie-only auth.
		if serviceName == publicServiceName && string(methodDesc.Name()) == sessionMethodName {
			sessionToken, err := cookie.GetSession(gc)
			if err != nil || sessionToken == "" {
				return nil, status.Error(codes.Unauthenticated, "unauthorized")
			}
			claims, err := tp.ValidateBrowserSession(gc, sessionToken)
			if err != nil || claims == nil || claims.Subject == "" {
				return nil, status.Error(codes.Unauthenticated, "unauthorized")
			}
			ctx = authctx.WithPrincipal(ctx, &authctx.Principal{
				UserID:      claims.Subject,
				LoginMethod: claims.LoginMethod,
				Nonce:       claims.Nonce,
			})
			return handler(ctx, req)
		}

		tokenData, err := tp.GetUserIDFromSessionOrAccessToken(gc)
		if err != nil || tokenData == nil || tokenData.UserID == "" {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		ctx = authctx.WithPrincipal(ctx, &authctx.Principal{
			UserID:      tokenData.UserID,
			LoginMethod: tokenData.LoginMethod,
			Nonce:       tokenData.Nonce,
		})
		return handler(ctx, req)
	}
}

func methodDescriptor(fullMethod string) (protoreflect.MethodDescriptor, bool) {
	if cached, ok := methodDescCache.Load(fullMethod); ok {
		if cached == nil {
			return nil, false
		}
		return cached.(protoreflect.MethodDescriptor), true
	}
	desc, ok := lookupMethodDescriptor(fullMethod)
	if ok {
		methodDescCache.Store(fullMethod, desc)
	} else {
		methodDescCache.Store(fullMethod, nil)
	}
	return desc, ok
}

func lookupMethodDescriptor(fullMethod string) (protoreflect.MethodDescriptor, bool) {
	parts := strings.Split(strings.TrimPrefix(fullMethod, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, false
	}
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(parts[0]))
	if err != nil {
		return nil, false
	}
	svcDesc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, false
	}
	methods := svcDesc.Methods()
	name := protoreflect.Name(parts[1])
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		if m.Name() == name {
			return m, true
		}
	}
	return nil, false
}

func isPublicMethod(method protoreflect.MethodDescriptor) bool {
	opts := method.Options()
	if opts == nil {
		return false
	}
	// proto.GetExtension may surface bool extensions as either bool or *bool
	// depending on code generation; handle both.
	publicOpt := proto.GetExtension(opts, authorizerv1.E_Public)
	switch v := publicOpt.(type) {
	case bool:
		return v
	case *bool:
		return v != nil && *v
	default:
		return false
	}
}

func shouldRejectUnlistedService(serviceName string) bool {
	if _, infra := infrastructureServices[serviceName]; infra {
		return false
	}
	return serviceName != publicServiceName && serviceName != adminServiceName
}
