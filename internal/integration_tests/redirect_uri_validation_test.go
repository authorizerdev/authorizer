package integration_tests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestRedirectURIValidation verifies that all endpoints accepting redirect_uri
// validate it against AllowedOrigins to prevent open-redirect token theft.
func TestRedirectURIValidation(t *testing.T) {
	cfg := getTestConfig()
	// Use SQLite so tests can run without Postgres/Docker
	cfg.DatabaseType = constants.DbTypeSqlite
	cfg.DatabaseURL = "authorizer_redirect_test.db"
	// IsValidOrigin compares hostname:port, so AllowedOrigins must not include protocol
	cfg.AllowedOrigins = []string{"localhost:3000"}
	cfg.EnableMagicLinkLogin = true
	cfg.EnableEmailVerification = true
	cfg.IsEmailServiceEnabled = true
	cfg.IsSMSServiceEnabled = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true

	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	attackerURL := "https://attacker.com/steal"
	validURL := "http://localhost:3000/callback"

	t.Run("ForgotPassword should reject invalid redirect_uri", func(t *testing.T) {
		// First create a user
		email := "fp_redirect_test_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, signupRes)

		// Attacker-controlled redirect_uri should be rejected
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef(attackerURL),
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid redirect URI")
	})

	t.Run("ForgotPassword should accept valid redirect_uri", func(t *testing.T) {
		email := "fp_valid_redirect_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, signupRes)

		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email:       refs.NewStringRef(email),
			RedirectURI: refs.NewStringRef(validURL),
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("MagicLinkLogin should reject invalid redirect_uri", func(t *testing.T) {
		email := "ml_redirect_test_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
			Email:       email,
			RedirectURI: &attackerURL,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid redirect URI")
	})

	t.Run("MagicLinkLogin should accept valid redirect_uri", func(t *testing.T) {
		email := "ml_valid_redirect_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
			Email:       email,
			RedirectURI: &validURL,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Signup should reject invalid redirect_uri", func(t *testing.T) {
		email := "signup_redirect_test_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
			RedirectURI:     &attackerURL,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid redirect URI")
	})

	t.Run("Signup should accept valid redirect_uri", func(t *testing.T) {
		email := "signup_valid_redirect_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
			RedirectURI:     &validURL,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("InviteMembers should reject invalid redirect_uri", func(t *testing.T) {
		cfg.IsEmailServiceEnabled = true
		cfg.EnableBasicAuthentication = true
		cfg.EnableMagicLinkLogin = true

		req, _ := createContext(ts)
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		emailTo := "invite_redirect_test_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.InviteMembers(ctx, &model.InviteMemberRequest{
			Emails:      []string{emailTo},
			RedirectURI: &attackerURL,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid redirect URI")
	})

	t.Run("InviteMembers should accept valid redirect_uri", func(t *testing.T) {
		cfg.IsEmailServiceEnabled = true
		cfg.EnableBasicAuthentication = true
		cfg.EnableMagicLinkLogin = true

		req, _ := createContext(ts)
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		emailTo := "invite_valid_redirect_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.InviteMembers(ctx, &model.InviteMemberRequest{
			Emails:      []string{emailTo},
			RedirectURI: &validURL,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
}
