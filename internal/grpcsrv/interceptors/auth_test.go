package interceptors

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/token"
)

type stubTokenProvider struct {
	token.Provider

	superAdmin  bool
	tokenData   *token.SessionOrAccessTokenData
	tokenErr    error
	sessionData *token.SessionData
	sessionErr  error

	superAdminChecks int
	userChecks       int
	sessionChecks    int
}

func (s *stubTokenProvider) IsSuperAdmin(_ *gin.Context) bool {
	s.superAdminChecks++
	return s.superAdmin
}

func (s *stubTokenProvider) GetUserIDFromSessionOrAccessToken(_ *gin.Context) (*token.SessionOrAccessTokenData, error) {
	s.userChecks++
	if s.tokenErr != nil {
		return nil, s.tokenErr
	}
	return s.tokenData, nil
}

func (s *stubTokenProvider) ValidateBrowserSession(_ *gin.Context, encryptedSession string) (*token.SessionData, error) {
	s.sessionChecks++
	if s.sessionErr != nil {
		return nil, s.sessionErr
	}
	if s.sessionData == nil || encryptedSession == "" {
		return nil, status.Error(codes.Unauthenticated, "bad session")
	}
	return s.sessionData, nil
}

func TestAuth_PublicMethodPassesThrough(t *testing.T) {
	stub := &stubTokenProvider{}
	mw := Auth(stub, nil, nil)

	called := false
	_, err := mw(context.Background(), &authorizerv1.MetaRequest{}, info(authorizerv1.AuthorizerService_Meta_FullMethodName), func(ctx context.Context, _ any) (any, error) {
		called = true
		_, ok := authctx.FromContext(ctx)
		assert.False(t, ok, "public methods should not attach principal")
		return &authorizerv1.Meta{}, nil
	})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, 0, stub.superAdminChecks)
	assert.Equal(t, 0, stub.userChecks)
}

func TestAuth_AdminMethodRequiresSuperAdmin(t *testing.T) {
	t.Run("rejects missing admin auth", func(t *testing.T) {
		stub := &stubTokenProvider{superAdmin: false}
		mw := Auth(stub, nil, nil)
		called := false
		_, err := mw(context.Background(), &authorizerv1.AdminMetaRequest{}, info(authorizerv1.AuthorizerAdminService_AdminMeta_FullMethodName), func(_ context.Context, _ any) (any, error) {
			called = true
			return &authorizerv1.AdminMetaResponse{}, nil
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.False(t, called)
		assert.Equal(t, 1, stub.superAdminChecks)
		// A non-super-admin is no longer rejected outright: the caller is
		// resolved so an org-admin can be authorized by the service layer.
		// With no credential at all that lookup yields nothing and the request
		// is still refused here, before any handler runs.
		assert.Equal(t, 1, stub.userChecks)
	})

	t.Run("attaches admin principal when authorized", func(t *testing.T) {
		stub := &stubTokenProvider{superAdmin: true}
		mw := Auth(stub, nil, nil)
		called := false
		_, err := mw(context.Background(), &authorizerv1.AdminMetaRequest{}, info(authorizerv1.AuthorizerAdminService_AdminMeta_FullMethodName), func(ctx context.Context, _ any) (any, error) {
			called = true
			p, ok := authctx.FromContext(ctx)
			require.True(t, ok)
			require.NotNil(t, p)
			assert.True(t, p.IsSuperAdmin)
			assert.Empty(t, p.UserID)
			return &authorizerv1.AdminMetaResponse{}, nil
		})
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, 1, stub.superAdminChecks)
		assert.Equal(t, 0, stub.userChecks)
	})
}

// TestAuth_AdminMethodAllowsAuthenticatedNonSuperAdmin pins the org-admin path
// through the interceptor.
//
// The admin surface accepts two identities: a platform super-admin for
// everything, and an org's own org-admin for that org's scoped operations. The
// interceptor cannot tell them apart — org membership lives in storage, which
// it has no access to — so it resolves the caller, attaches the principal, and
// leaves the decision to the service layer's requireSuperAdmin/requireOrgAdmin.
//
// The safety of that delegation rests on every admin method carrying its own
// gate. That property is asserted separately and exhaustively, by
// service.TestAdminMethodsAreGated (statically) and by
// integration_tests.TestAdminSurfaceDeniesPlainUser (on the real transports).
func TestAuth_AdminMethodAllowsAuthenticatedNonSuperAdmin(t *testing.T) {
	stub := &stubTokenProvider{
		superAdmin: false,
		tokenData: &token.SessionOrAccessTokenData{
			UserID:      "org-admin-1",
			LoginMethod: "basic_auth",
			Nonce:       "nonce-1",
		},
	}
	mw := Auth(stub, nil, nil)
	called := false
	_, err := mw(context.Background(), &authorizerv1.OrgMembersRequest{}, info(authorizerv1.AuthorizerAdminService_OrgMembers_FullMethodName), func(ctx context.Context, _ any) (any, error) {
		called = true
		p, ok := authctx.FromContext(ctx)
		require.True(t, ok, "an authenticated caller must reach the handler with a principal")
		require.NotNil(t, p)
		// Crucially NOT super-admin: the principal must not imply an authority
		// the caller does not hold, or requireSuperAdmin would pass for them.
		assert.False(t, p.IsSuperAdmin)
		assert.Equal(t, "org-admin-1", p.UserID)
		assert.Equal(t, "basic_auth", p.LoginMethod)
		assert.Equal(t, "nonce-1", p.Nonce)
		return &authorizerv1.OrgMembersResponse{}, nil
	})
	require.NoError(t, err)
	assert.True(t, called, "an authenticated non-super-admin must reach the admin handler")
	assert.Equal(t, 1, stub.superAdminChecks)
	assert.Equal(t, 1, stub.userChecks)
}

// TestAuth_AdminMethodRejectsBadCredential covers the other half: a credential
// that fails to resolve is refused at the interceptor, never reaching a handler.
func TestAuth_AdminMethodRejectsBadCredential(t *testing.T) {
	for name, stub := range map[string]*stubTokenProvider{
		"invalid token":   {tokenErr: status.Error(codes.Unauthenticated, "bad token")},
		"no token":        {},
		"empty user id":   {tokenData: &token.SessionOrAccessTokenData{UserID: ""}},
		"blank principal": {tokenData: &token.SessionOrAccessTokenData{LoginMethod: "basic_auth"}},
	} {
		t.Run(name, func(t *testing.T) {
			mw := Auth(stub, nil, nil)
			called := false
			_, err := mw(context.Background(), &authorizerv1.OrgMembersRequest{}, info(authorizerv1.AuthorizerAdminService_OrgMembers_FullMethodName), func(_ context.Context, _ any) (any, error) {
				called = true
				return &authorizerv1.OrgMembersResponse{}, nil
			})
			require.Error(t, err)
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
			assert.False(t, called, "handler must not run without a resolvable caller")
		})
	}
}

func TestAuth_PrivatePublicServiceMethodRequiresUser(t *testing.T) {
	t.Run("rejects unauthenticated user", func(t *testing.T) {
		stub := &stubTokenProvider{tokenErr: status.Error(codes.Unauthenticated, "bad token")}
		mw := Auth(stub, nil, nil)
		called := false
		_, err := mw(context.Background(), &authorizerv1.ProfileRequest{}, info(authorizerv1.AuthorizerService_Profile_FullMethodName), func(_ context.Context, _ any) (any, error) {
			called = true
			return &authorizerv1.User{}, nil
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.False(t, called)
		assert.Equal(t, 0, stub.superAdminChecks)
		assert.Equal(t, 1, stub.userChecks)
	})

	t.Run("attaches user principal", func(t *testing.T) {
		stub := &stubTokenProvider{
			tokenData: &token.SessionOrAccessTokenData{
				UserID:      "user-1",
				LoginMethod: "basic_auth",
				Nonce:       "nonce-1",
			},
		}
		mw := Auth(stub, nil, nil)
		called := false
		_, err := mw(context.Background(), &authorizerv1.ProfileRequest{}, info(authorizerv1.AuthorizerService_Profile_FullMethodName), func(ctx context.Context, _ any) (any, error) {
			called = true
			p, ok := authctx.FromContext(ctx)
			require.True(t, ok)
			require.NotNil(t, p)
			assert.Equal(t, "user-1", p.UserID)
			assert.Equal(t, "basic_auth", p.LoginMethod)
			assert.Equal(t, "nonce-1", p.Nonce)
			assert.False(t, p.IsSuperAdmin)
			return &authorizerv1.User{}, nil
		})
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, 0, stub.superAdminChecks)
		assert.Equal(t, 1, stub.userChecks)
	})
}

func TestAuth_InfrastructureServiceSkipsAuth(t *testing.T) {
	stub := &stubTokenProvider{}
	mw := Auth(stub, nil, nil)
	called := false
	_, err := mw(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, func(_ context.Context, _ any) (any, error) {
		called = true
		return "ok", nil
	})
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, 0, stub.superAdminChecks)
	assert.Equal(t, 0, stub.userChecks)
}

func TestAuth_SessionRequiresCookieRejectsBearer(t *testing.T) {
	stub := &stubTokenProvider{
		tokenData: &token.SessionOrAccessTokenData{UserID: "user-1"},
	}
	mw := Auth(stub, nil, nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer access-token",
	))
	called := false
	_, err := mw(ctx, &authorizerv1.SessionRequest{}, info(authorizerv1.AuthorizerService_Session_FullMethodName), func(_ context.Context, _ any) (any, error) {
		called = true
		return &authorizerv1.AuthResponse{}, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.False(t, called)
	assert.Equal(t, 0, stub.userChecks)
	assert.Equal(t, 0, stub.sessionChecks)
}

func TestAuth_SessionAcceptsCookie(t *testing.T) {
	stub := &stubTokenProvider{
		sessionData: &token.SessionData{
			Subject:     "user-1",
			LoginMethod: "basic_auth",
			Nonce:       "nonce-1",
		},
	}
	mw := Auth(stub, nil, nil)
	cookieName := constants.AppCookieName + "_session"
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"cookie", cookieName+"=sess-token",
	))
	called := false
	_, err := mw(ctx, &authorizerv1.SessionRequest{}, info(authorizerv1.AuthorizerService_Session_FullMethodName), func(ctx context.Context, _ any) (any, error) {
		called = true
		p, ok := authctx.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "user-1", p.UserID)
		return &authorizerv1.AuthResponse{}, nil
	})
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, 1, stub.sessionChecks)
	assert.Equal(t, 0, stub.userChecks)
}

func TestAuth_LogoutRequiresAuth(t *testing.T) {
	stub := &stubTokenProvider{tokenErr: status.Error(codes.Unauthenticated, "bad token")}
	mw := Auth(stub, nil, nil)
	called := false
	_, err := mw(context.Background(), &authorizerv1.LogoutRequest{}, info(authorizerv1.AuthorizerService_Logout_FullMethodName), func(_ context.Context, _ any) (any, error) {
		called = true
		return &authorizerv1.LogoutResponse{}, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.False(t, called)
}

// TestAuth_AdminLoginRemainsPublic guards the auth-bootstrap exception: the
// AdminLogin RPC establishes super-admin auth, so it MUST stay reachable without
// it. Regression guard that scoping the `public` bypass did not lock admins out.
func TestAuth_AdminLoginRemainsPublic(t *testing.T) {
	stub := &stubTokenProvider{superAdmin: false}
	mw := Auth(stub, nil, nil)
	called := false
	_, err := mw(context.Background(), &authorizerv1.AdminLoginRequest{}, info(authorizerv1.AuthorizerAdminService_AdminLogin_FullMethodName), func(ctx context.Context, _ any) (any, error) {
		called = true
		_, ok := authctx.FromContext(ctx)
		assert.False(t, ok, "the public bootstrap login must not attach a principal")
		return &authorizerv1.AdminLoginResponse{}, nil
	})
	require.NoError(t, err)
	assert.True(t, called, "AdminLogin must bypass auth (it establishes it)")
	assert.Equal(t, 0, stub.superAdminChecks, "AdminLogin must not require a pre-existing super-admin")
}

// TestAuth_OnlyAdminLoginIsPublicOnAdminService is the two-layer defense against
// the latent footgun: (1) the interceptor guard honors `public` on the admin
// service only for AdminLogin (see Auth), and (2) this invariant ensures no
// OTHER admin RPC carries `public` in the proto. If a future admin mutation is
// accidentally annotated public, this fails — and even if it merged, the
// interceptor guard would still deny it (falling through to the super-admin
// check), because adminLoginMethodName is an exact allowlist of one.
func TestAuth_OnlyAdminLoginIsPublicOnAdminService(t *testing.T) {
	adminSvc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(adminServiceName))
	require.NoError(t, err)
	svc, ok := adminSvc.(protoreflect.ServiceDescriptor)
	require.True(t, ok)
	methods := svc.Methods()
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		if string(m.Name()) == adminLoginMethodName {
			assert.Truef(t, isPublicMethod(m), "%s is expected to be public (auth bootstrap)", m.Name())
			continue
		}
		assert.Falsef(t, isPublicMethod(m),
			"admin RPC %s must never be marked public — only %s may bypass super-admin auth", m.Name(), adminLoginMethodName)
	}
}

func TestShouldRejectUnlistedService(t *testing.T) {
	assert.False(t, shouldRejectUnlistedService(publicServiceName))
	assert.False(t, shouldRejectUnlistedService(adminServiceName))
	assert.False(t, shouldRejectUnlistedService("grpc.health.v1.Health"))
	assert.True(t, shouldRejectUnlistedService("other.v1.UnknownService"))
}

// TestAuth_NilTokenProviderFailsClosed asserts the interceptor fails closed when
// no TokenProvider is wired (e.g. during early startup).
func TestAuth_NilTokenProviderFailsClosed(t *testing.T) {
	mw := Auth(nil, nil, nil)
	called := false
	_, err := mw(context.Background(), &authorizerv1.ProfileRequest{}, info(authorizerv1.AuthorizerService_Profile_FullMethodName), func(_ context.Context, _ any) (any, error) {
		called = true
		return &authorizerv1.User{}, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.False(t, called)
}

// TestAuth_SessionOnlyAcceptsPublicService asserts that the cookie-only Session
// path is guarded on publicServiceName so a future method named "Session" on
// another service does not inherit cookie-only auth.
func TestAuth_SessionOnlyAcceptsPublicService(t *testing.T) {
	// Session on the known publicServiceName — cookie path applies.
	stub := &stubTokenProvider{
		sessionData: &token.SessionData{Subject: "user-1", LoginMethod: "basic_auth", Nonce: "n"},
	}
	mw := Auth(stub, nil, nil)
	cookieName := constants.AppCookieName + "_session"
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("cookie", cookieName+"=tok"))
	_, err := mw(ctx, &authorizerv1.SessionRequest{}, info(authorizerv1.AuthorizerService_Session_FullMethodName), func(ctx context.Context, _ any) (any, error) {
		p, ok := authctx.FromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "user-1", p.UserID)
		return &authorizerv1.AuthResponse{}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, stub.sessionChecks)
	assert.Equal(t, 0, stub.userChecks)
}

// TestAuth_TokenResolverOverrideAppliesToBothSites pins the contract Auth's doc
// comment states: a non-nil TokenResolver replaces the default at EVERY
// identity-resolution site, not just one of them.
//
// This is the regression test for a real bug. The override was first wired into
// only the admin fallback, leaving the public path calling
// GetUserIDFromSessionOrAccessToken directly. Nothing caught it — it compiles,
// vets and lints clean, and the surface it breaks is the one that matters:
// MCP tools live on the PUBLIC AuthorizerService, so the MCP transport would
// have authenticated through the default resolver, which rejects the very
// resource-bound audience every MCP token is required to carry. The whole
// surface would have returned 401 with no failing test anywhere.
//
// The stub's counters are what make this airtight: asserting the custom resolver
// ran is not enough, because both could run. The default MUST NOT be consulted
// at all, or a token rejected by the surface's own rule could still be accepted
// by the default one.
func TestAuth_TokenResolverOverrideAppliesToBothSites(t *testing.T) {
	cases := []struct {
		name   string
		method string
	}{
		{"public service", authorizerv1.AuthorizerService_Profile_FullMethodName},
		{"admin service non-super-admin fallback", authorizerv1.AuthorizerAdminService_OrgMembers_FullMethodName},
	}

	for _, tc := range cases {
		t.Run(tc.name+" uses the override", func(t *testing.T) {
			// Default resolver would succeed; the override decides instead.
			stub := &stubTokenProvider{tokenData: &token.SessionOrAccessTokenData{UserID: "default-user"}}
			resolverCalls := 0
			mw := Auth(stub, nil, func(_ *gin.Context) (*token.SessionOrAccessTokenData, error) {
				resolverCalls++
				return &token.SessionOrAccessTokenData{UserID: "override-user"}, nil
			})

			var seen string
			_, err := mw(context.Background(), nil, info(tc.method), func(ctx context.Context, _ any) (any, error) {
				p, ok := authctx.FromContext(ctx)
				require.True(t, ok, "an authenticated call must carry a principal")
				seen = p.UserID
				return nil, nil
			})
			require.NoError(t, err)
			assert.Equal(t, "override-user", seen, "the principal must come from the override, not the default")
			assert.Equal(t, 1, resolverCalls)
			assert.Zero(t, stub.userChecks, "the default resolver must not be consulted when an override is set")
		})

		t.Run(tc.name+" rejects when the override rejects", func(t *testing.T) {
			// The inverse, and the one that actually protects the audience
			// boundary: the default would ACCEPT this caller. If the default
			// were still reachable, a token the surface's own rule rejected
			// would authenticate anyway.
			stub := &stubTokenProvider{tokenData: &token.SessionOrAccessTokenData{UserID: "default-user"}}
			mw := Auth(stub, nil, func(_ *gin.Context) (*token.SessionOrAccessTokenData, error) {
				return nil, status.Error(codes.Unauthenticated, "wrong audience")
			})

			called := false
			_, err := mw(context.Background(), nil, info(tc.method), func(context.Context, any) (any, error) {
				called = true
				return nil, nil
			})
			require.Error(t, err)
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
			assert.False(t, called, "the handler must not run for a caller the surface's resolver rejected")
			assert.Zero(t, stub.userChecks, "a rejection must not fall back to the default resolver")
		})
	}
}
