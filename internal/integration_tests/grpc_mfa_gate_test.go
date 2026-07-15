package integration_tests

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// bootPublicClientForConfig is newPublicClient's sibling for tests that need
// to seed cfg (e.g. EnableMFA) before initTestSetup boots the service
// provider; newPublicClient always calls getTestConfig() internally so it
// can't be reused here.
func bootPublicClientForConfig(t *testing.T, cfg *config.Config) (authorizerv1.AuthorizerServiceClient, *testSetup) {
	t.Helper()
	ts := initTestSetup(t, cfg)

	srv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          cfg,
		ServiceProvider: ts.ServiceProvider,
		TokenProvider:   ts.TokenProvider,
	})
	require.NoError(t, err)

	lis := bufconn.Listen(1 << 20)
	t.Cleanup(func() { _ = lis.Close() })
	go func() { _ = srv.GRPCServer().Serve(lis) }()
	t.Cleanup(srv.GRPCServer().GracefulStop)

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return authorizerv1.NewAuthorizerServiceClient(conn), ts
}

// mfaSessionCookieCtx builds an outgoing gRPC context carrying the mfa
// session cookie directly (no "cookie:" prefix parsing needed since the
// value already is "name=value").
func mfaSessionCookieCtx(session string) context.Context {
	return metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("cookie", constants.MfaCookieName+"_session="+session))
}

// findSetCookie returns the first Set-Cookie header value (as captured via
// grpc.Header) whose cookie name matches, or "" if none match. Mirrors how
// TestSessionGRPCRequiresCookieOnly extracts the session cookie from gRPC
// response metadata.
func findSetCookie(cookies []string, name string) string {
	prefix := name + "="
	for _, c := range cookies {
		if len(c) >= len(prefix) && c[:len(prefix)] == prefix {
			return c
		}
	}
	return ""
}

// TestSkipMfaSetupGRPC exercises the new SkipMfaSetup RPC end-to-end over
// gRPC: sign up + enable MFA, log in (token withheld behind an MFA session
// cookie), then prove skip_mfa_setup issues the withheld token and persists
// HasSkippedMfaSetupAt. Closes the REST/gRPC parity gap for the GraphQL-only
// skip_mfa_setup mutation added in PR #686.
func TestSkipMfaSetupGRPC(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	c, ts := bootPublicClientForConfig(t, cfg)
	ctx := context.Background()

	email := "grpc_skip_mfa_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	_, err := c.Signup(ctx, &authorizerv1.SignupRequest{
		Email: email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	_, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)

	var header metadata.MD
	loginResp, err := c.Login(ctx, &authorizerv1.LoginRequest{Email: email, Password: password}, grpc.Header(&header))
	require.NoError(t, err)
	require.Empty(t, loginResp.AccessToken, "first login with optional MFA and no prior enrollment must withhold the token")
	require.True(t, loginResp.ShouldShowTotpScreen)

	mfaCookie := findSetCookie(header.Get("Set-Cookie"), constants.MfaCookieName+"_session")
	require.NotEmpty(t, mfaCookie, "login must set an mfa session cookie via gRPC response metadata")
	mfaCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("cookie", mfaCookie))

	t.Run("skip issues the withheld token and persists HasSkippedMfaSetupAt", func(t *testing.T) {
		resp, err := c.SkipMfaSetup(mfaCtx, &authorizerv1.SkipMfaSetupRequest{Email: email})
		require.NoError(t, err)
		require.NotEmpty(t, resp.AccessToken, "skip must issue the token withheld at login")

		updated, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, updated.HasSkippedMFASetupAt)
	})

	t.Run("without a valid mfa session cookie it is rejected, not Unimplemented", func(t *testing.T) {
		_, err := c.SkipMfaSetup(context.Background(), &authorizerv1.SkipMfaSetupRequest{Email: "nobody@authorizer.dev"})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}

// TestLockMfaGRPC exercises the new LockMfa RPC end-to-end over gRPC: a
// caller with a valid mfa session cookie can lock their account (no verified
// OTP fallback enrolled), and a caller without one is rejected.
func TestLockMfaGRPC(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	c, ts := bootPublicClientForConfig(t, cfg)
	ctx := context.Background()

	email := "grpc_lock_mfa_" + uuid.New().String() + "@authorizer.dev"
	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:                    refs.NewStringRef(email),
		EmailVerifiedAt:          &now,
		SignupMethods:            constants.AuthRecipeMethodBasicAuth,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
	})
	require.NoError(t, err)

	mfaSession := uuid.NewString()
	require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, time.Now().Add(5*time.Minute).Unix()))

	t.Run("locks the account with a valid mfa session", func(t *testing.T) {
		resp, err := c.LockMfa(mfaSessionCookieCtx(mfaSession), &authorizerv1.LockMfaRequest{Email: email})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Message)

		updated, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, updated.MFALockedAt)
	})

	t.Run("without a valid mfa session it is rejected, not Unimplemented", func(t *testing.T) {
		_, err := c.LockMfa(context.Background(), &authorizerv1.LockMfaRequest{Email: email})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}

// TestEmailOtpMfaSetupGRPC exercises the new EmailOtpMfaSetup RPC end-to-end
// over gRPC, proving the transport correctly reaches both auth modes
// resolveOTPSetupCaller supports (already proven at the GraphQL layer by
// TestEmailOTPMFASetupViaMfaSessionCookie / TestOTPMFASetupRejectsUnauthenticatedCaller
// in otp_mfa_setup_test.go): the MFA-session-cookie + email fallback for a
// caller in the withheld first-time-offer state, and the ordinary bearer
// token used by an already-authenticated caller adding a second factor.
func TestEmailOtpMfaSetupGRPC(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	cfg.EnableEmailOTP = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	c, ts := bootPublicClientForConfig(t, cfg)
	ctx := context.Background()
	password := "Password@123"

	t.Run("cookie mode: mfa session cookie + email, no bearer token", func(t *testing.T) {
		email := "grpc_email_otp_cookie_" + uuid.New().String() + "@authorizer.dev"

		_, err := c.Signup(ctx, &authorizerv1.SignupRequest{
			Email: email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		var header metadata.MD
		loginResp, err := c.Login(ctx, &authorizerv1.LoginRequest{Email: email, Password: password}, grpc.Header(&header))
		require.NoError(t, err)
		require.Empty(t, loginResp.AccessToken, "first login with optional MFA and no prior enrollment must withhold the token")

		mfaCookie := findSetCookie(header.Get("Set-Cookie"), constants.MfaCookieName+"_session")
		require.NotEmpty(t, mfaCookie, "login must set an mfa session cookie via gRPC response metadata")
		mfaCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("cookie", mfaCookie))

		resp, err := c.EmailOtpMfaSetup(mfaCtx, &authorizerv1.EmailOtpMfaSetupRequest{Email: email})
		require.NoError(t, err, "email_otp_mfa_setup must be reachable via cookie+email with no bearer token")
		require.NotEmpty(t, resp.Message)

		authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
		require.NoError(t, err)
		require.NotNil(t, authenticator)
		assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")
	})

	t.Run("bearer-token mode: already authenticated caller, no email param", func(t *testing.T) {
		email := "grpc_email_otp_bearer_" + uuid.New().String() + "@authorizer.dev"

		// Signup omits is_multi_factor_auth_enabled, which the gRPC handler
		// still forwards explicitly as false (see AuthorizerHandler.Signup),
		// so the MFA gate never engages here and the token is issued
		// directly -- this caller reaches EmailOtpMfaSetup the ordinary
		// already-logged-in "add a second factor from settings" way.
		signupResp, err := c.Signup(ctx, &authorizerv1.SignupRequest{
			Email: email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotEmpty(t, signupResp.AccessToken, "no MFA enrolled -> token issued directly")

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)

		resp, err := c.EmailOtpMfaSetup(bearerCtx(signupResp.AccessToken), &authorizerv1.EmailOtpMfaSetupRequest{})
		require.NoError(t, err, "email_otp_mfa_setup must be reachable via bearer token with no email param")
		require.NotEmpty(t, resp.Message)

		authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
		require.NoError(t, err)
		require.NotNil(t, authenticator)
		assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")
	})

	t.Run("without a valid token or cookie+email it is rejected, not Unimplemented", func(t *testing.T) {
		_, err := c.EmailOtpMfaSetup(context.Background(), &authorizerv1.EmailOtpMfaSetupRequest{})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}

// TestSmsOtpMfaSetupGRPC is TestEmailOtpMfaSetupGRPC's SMS twin -- same two
// auth modes and rejection case, keyed by phone_number instead of email.
func TestSmsOtpMfaSetupGRPC(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableSMSOTP = true
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	cfg.TwilioAPISecret = "test-twilio-api-secret"
	cfg.TwilioAPIKey = "test-twilio-api-key"
	cfg.TwilioAccountSID = "test-twilio-account-sid"
	cfg.TwilioSender = "test-twilio-sender"
	c, ts := bootPublicClientForConfig(t, cfg)
	ctx := context.Background()
	password := "Password@123"

	t.Run("cookie mode: mfa session cookie + phone_number, no bearer token", func(t *testing.T) {
		mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)

		_, err := c.Signup(ctx, &authorizerv1.SignupRequest{
			PhoneNumber: mobile, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
		require.NoError(t, err)
		// Signup's own phone-verification OTP is irrelevant here; mark the
		// phone verified directly so login reaches the MFA gate instead of
		// the phone-verification challenge, same as the GraphQL-layer twin.
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		var header metadata.MD
		loginResp, err := c.Login(ctx, &authorizerv1.LoginRequest{PhoneNumber: mobile, Password: password}, grpc.Header(&header))
		require.NoError(t, err)
		require.Empty(t, loginResp.AccessToken, "first login with optional MFA and no prior enrollment must withhold the token")

		mfaCookie := findSetCookie(header.Get("Set-Cookie"), constants.MfaCookieName+"_session")
		require.NotEmpty(t, mfaCookie, "login must set an mfa session cookie via gRPC response metadata")
		mfaCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("cookie", mfaCookie))

		resp, err := c.SmsOtpMfaSetup(mfaCtx, &authorizerv1.SmsOtpMfaSetupRequest{PhoneNumber: mobile})
		require.NoError(t, err, "sms_otp_mfa_setup must be reachable via cookie+phone_number with no bearer token")
		require.NotEmpty(t, resp.Message)

		authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
		require.NoError(t, err)
		require.NotNil(t, authenticator)
		assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")
	})

	t.Run("bearer-token mode: already authenticated caller, no phone_number param", func(t *testing.T) {
		mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)

		_, err := c.Signup(ctx, &authorizerv1.SignupRequest{
			PhoneNumber: mobile, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
		require.NoError(t, err)
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// Same reasoning as the email twin: is_multi_factor_auth_enabled is
		// always forwarded explicitly (false here), so login issues the
		// token directly instead of hitting the MFA gate.
		loginResp, err := c.Login(ctx, &authorizerv1.LoginRequest{PhoneNumber: mobile, Password: password})
		require.NoError(t, err)
		require.NotEmpty(t, loginResp.AccessToken, "no MFA enrolled -> token issued directly")

		resp, err := c.SmsOtpMfaSetup(bearerCtx(loginResp.AccessToken), &authorizerv1.SmsOtpMfaSetupRequest{})
		require.NoError(t, err, "sms_otp_mfa_setup must be reachable via bearer token with no phone_number param")
		require.NotEmpty(t, resp.Message)

		authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
		require.NoError(t, err)
		require.NotNil(t, authenticator)
		assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")
	})

	t.Run("without a valid token or cookie+phone_number it is rejected, not Unimplemented", func(t *testing.T) {
		_, err := c.SmsOtpMfaSetup(context.Background(), &authorizerv1.SmsOtpMfaSetupRequest{})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}
