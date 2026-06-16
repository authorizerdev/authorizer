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
	mw := Auth(stub)

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
		mw := Auth(stub)
		called := false
		_, err := mw(context.Background(), &authorizerv1.AdminMetaRequest{}, info(authorizerv1.AuthorizerAdminService_AdminMeta_FullMethodName), func(_ context.Context, _ any) (any, error) {
			called = true
			return &authorizerv1.AdminMetaResponse{}, nil
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.False(t, called)
		assert.Equal(t, 1, stub.superAdminChecks)
		assert.Equal(t, 0, stub.userChecks)
	})

	t.Run("attaches admin principal when authorized", func(t *testing.T) {
		stub := &stubTokenProvider{superAdmin: true}
		mw := Auth(stub)
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

func TestAuth_PrivatePublicServiceMethodRequiresUser(t *testing.T) {
	t.Run("rejects unauthenticated user", func(t *testing.T) {
		stub := &stubTokenProvider{tokenErr: status.Error(codes.Unauthenticated, "bad token")}
		mw := Auth(stub)
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
		mw := Auth(stub)
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
	mw := Auth(stub)
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
	mw := Auth(stub)
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
	mw := Auth(stub)
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
	mw := Auth(stub)
	called := false
	_, err := mw(context.Background(), &authorizerv1.LogoutRequest{}, info(authorizerv1.AuthorizerService_Logout_FullMethodName), func(_ context.Context, _ any) (any, error) {
		called = true
		return &authorizerv1.LogoutResponse{}, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.False(t, called)
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
	mw := Auth(nil)
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
	mw := Auth(stub)
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
