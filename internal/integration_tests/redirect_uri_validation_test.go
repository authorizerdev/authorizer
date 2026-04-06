package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestRedirectURIRejectsAttacker verifies that forgot_password rejects
// attacker-controlled redirect_uri values with explicit AllowedOrigins.
func TestRedirectURIRejectsAttacker(t *testing.T) {
	cfg := getTestConfig()
	cfg.AllowedOrigins = []string{"http://localhost:3000"}
	cfg.EnableBasicAuthentication = true
	cfg.EnableEmailVerification = false
	cfg.EnableSignup = true

	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "redirect_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	t.Run("rejects attacker redirect_uri", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef("https://attacker.com/steal"),
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid redirect URI")
	})

	t.Run("accepts valid redirect_uri", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef("http://localhost:3000/reset"),
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// TestRedirectURIWildcardOrigins is a regression test for the open redirect
// vulnerability (issue #540). When allowed_origins=["*"] (the default config),
// attacker-controlled redirect_uri values must still be rejected.
func TestRedirectURIWildcardOrigins(t *testing.T) {
	cfg := getTestConfig()
	cfg.AllowedOrigins = []string{"*"}
	cfg.EnableBasicAuthentication = true
	cfg.EnableEmailVerification = false
	cfg.EnableSignup = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "wildcard_redirect_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	t.Run("rejects attacker redirect_uri with wildcard origins", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef("https://attacker.com/capture"),
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid redirect URI")
	})

	t.Run("allows self-origin redirect_uri with wildcard origins", func(t *testing.T) {
		selfURI := "http://" + ts.HttpServer.Listener.Addr().String() + "/app/reset-password"
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef(selfURI),
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.Message)
	})

	t.Run("works without redirect_uri (uses default)", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email: refs.NewStringRef(email),
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.Message)
	})

	t.Run("rejects javascript scheme", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef("javascript:alert(1)"),
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("rejects data scheme", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef("data:text/html,<h1>evil</h1>"),
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
