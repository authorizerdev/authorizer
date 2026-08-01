package interceptors

import (
	"context"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
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
	"github.com/authorizerdev/authorizer/internal/metrics"
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
// log may be nil (rejections are then only counted, not logged).
func Auth(tp token.Provider, log *zerolog.Logger) grpc.UnaryServerInterceptor {
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
			// Platform super-admin: unchanged, and still the only identity that
			// reaches the platform-wide operations.
			if tp.IsSuperAdmin(gc) {
				ctx = authctx.WithPrincipal(ctx, &authctx.Principal{IsSuperAdmin: true})
				return handler(ctx, req)
			}

			// Not a super-admin — but super-admin is NOT the only identity the admin
			// surface accepts. The service layer authorizes org-scoped operations
			// with requireOrgAdmin, which passes for a super-admin OR for that
			// org's own org-admin (constants.OrgRoleAdmin). Rejecting every
			// non-super-admin here meant those ops were reachable over GraphQL but
			// not over gRPC/REST: an org-admin listing their own org's members got
			// `<nil>` on GraphQL and Unauthenticated on the other two.
			//
			// So attach the caller's identity and let the service layer decide, the
			// same way the public service below does. This is only safe because
			// EVERY admin method except AdminLogin (the bootstrap, handled above)
			// opens with requireSuperAdmin or requireOrgAdmin — verified across all
			// 78 AdminProvider methods, each gate at the top level of the function,
			// never inside a branch. A new admin method that forgets its gate would
			// now be reachable by any authenticated user, which is what
			// TestAdminMethodsAreGated exists to prevent.
			tokenData, err := tp.GetUserIDFromSessionOrAccessToken(gc)
			if err != nil || tokenData == nil || tokenData.UserID == "" {
				// No usable credential at all — reject before reaching a handler.
				if isPublicMethod(methodDesc) {
					// This method carries the `public` proto annotation but isn't
					// AdminLogin, so the bypass above deliberately did not honor it —
					// it fell through here and is still being blocked. That is a
					// mis-annotated admin RPC (a real footgun: it means someone marked
					// a sensitive admin method public by mistake), worth alerting on
					// distinctly from an ordinary unauthenticated caller.
					metrics.RecordSecurityEvent("admin_public_bypass_blocked", "grpc_auth")
					if log != nil {
						log.Warn().Str("method", string(methodDesc.Name())).Msg("admin RPC marked public but is not AdminLogin — bypass denied, authentication still required")
					}
				}
				return nil, status.Error(codes.Unauthenticated, "unauthorized")
			}
			// Authenticated, not super-admin. The handler's service call decides:
			// requireSuperAdmin still rejects (the principal carries no
			// IsSuperAdmin and the request carries no admin secret), while
			// requireOrgAdmin can now resolve this caller's org membership.
			ctx = authctx.WithPrincipal(ctx, &authctx.Principal{
				UserID:      tokenData.UserID,
				LoginMethod: tokenData.LoginMethod,
				Nonce:       tokenData.Nonce,
			})
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
