package integration_tests

import (
	"net/url"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestForgotPassword tests the forgot password functionality
func TestForgotPassword(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "forgot_password_test_" + uuid.New().String() + "@authorizer.dev"
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
	assert.NotNil(t, signupRes.User)

	// To prevent user enumeration, ForgotPassword returns the same generic
	// success response whether or not the account exists. The previous
	// behaviour of returning an error for unknown emails was a leak.
	t.Run("should return generic success for unknown email", func(t *testing.T) {
		forgotPasswordReq := &model.ForgotPasswordRequest{
			Email: refs.NewStringRef("invalid-email@gmail.com"),
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ForgotPassword(ctx, forgotPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, forgotPasswordRes)
	})

	t.Run("should send forgot password email", func(t *testing.T) {
		forgotPasswordReq := &model.ForgotPasswordRequest{
			Email: refs.NewStringRef(email),
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ForgotPassword(ctx, forgotPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, forgotPasswordRes)
		assert.NotEmpty(t, forgotPasswordRes.Message)

		// Validate if the entry is created in db
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.NotEmpty(t, request.Token)
		assert.Equal(t, email, request.Email)
	})

	// Issue #344: a user who reaches the login UI from an OIDC /authorize
	// redirect and then resets their password must be able to resume that
	// authorization request. The login UI forwards the original /authorize
	// parameters on the reset-password redirect_uri; the server must keep them
	// and build a reset link that still carries them alongside the token.
	t.Run("should preserve authorize params on reset link", func(t *testing.T) {
		authorizeParams := url.Values{}
		authorizeParams.Set("client_id", cfg.ClientID)
		authorizeParams.Set("redirect_uri", "http://localhost:3000/callback")
		authorizeParams.Set("response_type", "code")
		authorizeParams.Set("response_mode", "query")
		authorizeParams.Set("scope", "openid profile email")
		authorizeParams.Set("state", "issue-344-state")
		authorizeParams.Set("code_challenge", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")
		authorizeParams.Set("code_challenge_method", "S256")
		// The test config's only allowed origin is http://localhost:3000.
		redirectURI := "http://localhost:3000/app/reset-password?" + authorizeParams.Encode()

		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			State:       refs.NewStringRef("issue-344-state"),
			RedirectURI: refs.NewStringRef(redirectURI),
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)
		require.NotNil(t, request)
		assert.Equal(t, redirectURI, request.RedirectURI)

		// The link mailed to the user (same builder ForgotPassword uses).
		link, err := url.Parse(utils.GetForgotPasswordURL(request.Token, request.RedirectURI))
		require.NoError(t, err)
		assert.Equal(t, "/app/reset-password", link.Path)
		q := link.Query()
		assert.Equal(t, request.Token, q.Get("token"))
		for k := range authorizeParams {
			assert.Equal(t, authorizeParams.Get(k), q.Get(k), "authorize param %q lost from reset link", k)
		}

		// The token from that link still resets the password, and the user can
		// then log in with the new one (which is what resumes /authorize in
		// the login UI).
		newPassword := "NewPassword@344"
		resetRes, err := ts.GraphQLProvider.ResetPassword(ctx, &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(q.Get("token")),
			Password:        newPassword,
			ConfirmPassword: newPassword,
		})
		require.NoError(t, err)
		require.NotNil(t, resetRes)
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    refs.NewStringRef(email),
			Password: newPassword,
		})
		require.NoError(t, err)
		require.NotNil(t, loginRes)
	})

	// Carrying authorize params must not loosen redirect validation: the
	// reset-password redirect_uri itself is still checked against allowed
	// origins, whatever its query string says.
	t.Run("should reject untrusted redirect uri carrying authorize params", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			State:       refs.NewStringRef("issue-344-state"),
			RedirectURI: refs.NewStringRef("https://evil.example.com/app/reset-password?client_id=x&state=y&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback")),
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
