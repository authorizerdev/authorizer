package integration_tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyEmail tests the verify email functionality
// using the GraphQL API.
// It creates a user, verifies the email, and checks
// the changes in the database.
func TestVerifyEmail(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "verify_email_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, signupRes)
	// Expect the user to be nil, as the email is not verified yet
	assert.Nil(t, signupRes.User)

	t.Run("should fail for invalid token", func(t *testing.T) {
		verificationReq := &model.VerifyEmailRequest{
			Token: "invalid-token",
		}
		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should verify email and use correct login method for basic auth", func(t *testing.T) {
		basicAuthEmail := "verify_email_basic_" + uuid.New().String() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &basicAuthEmail,
			Password:        password,
			ConfirmPassword: password,
		})
		assert.NoError(t, err)

		vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, basicAuthEmail, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, vreq)
		// Identifier should be basic_auth_signup, not magic_link_login
		assert.Equal(t, constants.VerificationTypeBasicAuthSignup, vreq.Identifier)

		verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: vreq.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes)
		assert.NotEmpty(t, verifyRes.AccessToken)
		assert.NotNil(t, verifyRes.User)
		assert.True(t, verifyRes.User.EmailVerified)
	})
	t.Run("should fail for revoked user", func(t *testing.T) {
		revokedEmail := "verify_email_revoked_" + uuid.New().String() + "@authorizer.dev"
		revokedSignupReq := &model.SignUpRequest{
			Email:           &revokedEmail,
			Password:        password,
			ConfirmPassword: password,
		}
		_, err := ts.GraphQLProvider.SignUp(ctx, revokedSignupReq)
		require.NoError(t, err)

		// Get verification token
		vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, revokedEmail, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		require.NotNil(t, vreq)

		// Revoke the user
		user, err := ts.StorageProvider.GetUserByEmail(ctx, revokedEmail)
		require.NoError(t, err)
		now := time.Now().Unix()
		user.RevokedTimestamp = &now
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// Try to verify email - should fail
		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: vreq.Token,
		})
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
		assert.Contains(t, err.Error(), "revoked")
	})

	t.Run("should verify email", func(t *testing.T) {
		// Get the verification token from db
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.NotEmpty(t, request.Token)

		// Verify email with an invalid token
		verificationReq := &model.VerifyEmailRequest{
			Token: request.Token,
		}

		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, verificationReq)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRes)
		assert.NotEmpty(t, verificationRes.AccessToken)
	})
}

// TestVerifyEmailRESTEndpointMFAGate guards GET /verify_email - the actual
// URL the "verify your email" / magic-link-login email sends users to. It is
// a separate implementation from the GraphQL verify_email mutation (which
// TestVerifyEmail and TestMagicLinkLoginMFAGate exercise), and previously
// issued a full session unconditionally with no MFA gate check and no
// RevokedTimestamp check at all - a user could complete signup email
// verification or magic-link login and be handed working tokens regardless
// of MFA enrollment status or account revocation, entirely bypassing
// resolveMFAGate. Exercises the real HTTP handler, not the service layer
// directly, since that's exactly the boundary the bug lived at.
func TestVerifyEmailRESTEndpointMFAGate(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.EnableMFA = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	password := "Password@123"

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	signupAndGetToken := func(t *testing.T, email string) string {
		t.Helper()
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		require.NotNil(t, vreq)
		return vreq.Token
	}

	hitVerifyEmail := func(t *testing.T, token string) *http.Response {
		t.Helper()
		reqURL := ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(token) +
			"&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback")
		resp, err := httpClient.Get(reqURL)
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	t.Run("MFA gate withholds tokens, matching the GraphQL mutation", func(t *testing.T) {
		email := "verify_email_rest_offer_" + uuid.New().String() + "@authorizer.dev"
		resp := hitVerifyEmail(t, signupAndGetToken(t, email))

		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		location := resp.Header.Get("Location")
		assert.NotContains(t, location, "access_token=")
		assert.Contains(t, location, "mfa_required=1")
		assert.Contains(t, location, "mfa_methods=")
	})

	t.Run("revoked user is rejected, not issued a session", func(t *testing.T) {
		email := "verify_email_rest_revoked_" + uuid.New().String() + "@authorizer.dev"
		token := signupAndGetToken(t, email)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		now := time.Now().Unix()
		user.RevokedTimestamp = &now
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		resp := hitVerifyEmail(t, token)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		location := resp.Header.Get("Location")
		assert.NotContains(t, location, "access_token=")
		assert.Contains(t, location, "error=")
	})

	t.Run("no MFA configured still completes normally with real tokens", func(t *testing.T) {
		cfgNoMFA := getTestConfig()
		cfgNoMFA.IsEmailServiceEnabled = true
		cfgNoMFA.EnableEmailVerification = true
		tsNoMFA := initTestSetup(t, cfgNoMFA)
		_, ctxNoMFA := createContext(tsNoMFA)

		email := "verify_email_rest_nomfa_" + uuid.New().String() + "@authorizer.dev"
		_, err := tsNoMFA.GraphQLProvider.SignUp(ctxNoMFA, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		vreq, err := tsNoMFA.StorageProvider.GetVerificationRequestByEmail(ctxNoMFA, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)

		reqURL := tsNoMFA.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
			"&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback")
		resp, err := httpClient.Get(reqURL)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Location"), "access_token=")
	})
}
