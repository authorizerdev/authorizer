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
	"github.com/authorizerdev/authorizer/internal/delegatedscope"
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

// TokenResolver turns a request into the caller's identity, or an error when the
// request carries no credential this surface accepts. It is the single point at
// which a gRPC server decides WHICH tokens authenticate it.
//
// The default (GetUserIDFromSessionOrAccessToken) accepts a browser session
// cookie or a first-party bearer token, and rejects every resource-bound
// audience. The MCP surface overrides it with MCPTokenResolver, which does the
// exact opposite on the audience and drops the cookie path entirely. Because the
// override is per-server rather than per-request, no token can cross between the
// two surfaces — see MCPTokenResolver.
type TokenResolver func(gc *gin.Context) (*token.SessionOrAccessTokenData, error)

// Auth returns a unary interceptor that enforces proto-declared auth policy.
// log may be nil (rejections are then only counted, not logged).
//
// resolve may be nil, in which case the caller's identity is resolved with
// tp.GetUserIDFromSessionOrAccessToken and the cookie-based paths below stay
// active — the behaviour every TCP-listening server uses.
//
// A non-nil resolve is the SOLE authority for that server. It replaces the
// identity-resolution site on the public path, refuses the AuthorizerAdminService
// outright, and disables the Session RPC's cookie-only branch.
//
// Narrowing that far is the point, not a side effect. A surface that declares its
// own token rule — MCP, whose rule is "the audience must name this MCP server" —
// must not be reachable with a credential that rule never saw. Leaving the other
// paths active meant the boundary held only because no cookie-authenticated
// method happened to be mcp_tool-exposed, and transport.MetaFromGRPC
// reconstructs cookies from gRPC metadata, so a bridge that forwarded headers
// wholesale would have made a browser session authenticate a tool call on an
// internet-facing, CSRF-exempt endpoint.
//
// The admin service is refused wholesale rather than merely skipping its
// super-admin check, because service.requireSuperAdmin re-derives super-admin
// from meta.Request on its own — skipping the check here would move it one layer
// down, not remove it.
func Auth(tp token.Provider, log *zerolog.Logger, resolve TokenResolver) grpc.UnaryServerInterceptor {
	resolverIsSoleAuthority := resolve != nil
	if resolve == nil {
		resolve = func(gc *gin.Context) (*token.SessionOrAccessTokenData, error) {
			return tp.GetUserIDFromSessionOrAccessToken(gc)
		}
	}
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
			// A resolver-governed surface does not serve the admin API at all.
			//
			// Skipping the IsSuperAdmin check here is NOT enough on its own:
			// service.requireSuperAdmin re-derives super-admin from meta.Request
			// (admin_provider.go), reading the admin cookie or the
			// x-authorizer-admin-secret header that transport.MetaFromGRPC
			// reconstructs from gRPC metadata. Disabling the check at this layer
			// would only move it one layer down, so a caller holding a valid
			// MCP-audience token plus an admin credential would still reach
			// platform-wide operations on an internet-facing, CSRF-exempt
			// surface. Refusing the whole service is the only version of this
			// guard that actually holds, and it costs nothing: no admin RPC is
			// mcp_tool-exposed, so nothing legitimate is being turned off.
			if resolverIsSoleAuthority {
				return nil, status.Error(codes.Unauthenticated, "unauthorized")
			}
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
			tokenData, err := resolve(gc)
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
				ActorID:     tokenData.ActorID,
				Scope:       tokenData.Scope,
			})
			if err := enforceDelegatedScope(tokenData, info.FullMethod); err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}

		// Session rotates the browser session cookie only; bearer tokens are ignored.
		// Guard on publicServiceName to prevent a future method named "Session" on
		// another service from inheriting cookie-only auth. Skipped entirely when a
		// resolver is the sole authority: a cookie is not a credential such a
		// surface accepts, so Session falls through to the resolver and is rejected
		// like any other unauthenticated call.
		if !resolverIsSoleAuthority && serviceName == publicServiceName && string(methodDesc.Name()) == sessionMethodName {
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

		tokenData, err := resolve(gc)
		if err != nil || tokenData == nil || tokenData.UserID == "" {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		ctx = authctx.WithPrincipal(ctx, &authctx.Principal{
			UserID:      tokenData.UserID,
			LoginMethod: tokenData.LoginMethod,
			Nonce:       tokenData.Nonce,
			ActorID:     tokenData.ActorID,
			Scope:       tokenData.Scope,
		})
		if err := enforceDelegatedScope(tokenData, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// enforceDelegatedScope gates a DELEGATED caller on the scope its token
// actually carries. First-party callers are untouched — see the
// delegatedscope package comment for why that asymmetry is deliberate.
//
// This is the choke point for gRPC, the REST gateway (which dispatches through
// these same methods) and the MCP server (which serves over an in-process
// bufconn). GraphQL has its own, in http_handlers.
func enforceDelegatedScope(tokenData *token.SessionOrAccessTokenData, fullMethod string) error {
	if tokenData == nil || strings.TrimSpace(tokenData.ActorID) == "" {
		return nil
	}
	required, ok := delegatedscope.RequiredForGRPC(fullMethod)
	if !ok {
		// Fail closed: an operation nobody has cleared for delegated callers is
		// out of reach for an agent, whatever scope it holds.
		return status.Error(codes.PermissionDenied, "insufficient_scope")
	}
	if !delegatedscope.Satisfied(tokenData.Scope, required) {
		return status.Error(codes.PermissionDenied, "insufficient_scope")
	}
	return nil
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
