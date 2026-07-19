package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestEnrolledMFAMethods covers the lazily-resolved User.enrolled_mfa_methods
// GraphQL field: it must report exactly the MFA factors a user has actually
// verified/enrolled, mirroring the verified checks the login MFA gate uses,
// and never report a factor that exists but is unverified.
//
// The tests exercise the resolver's delegate target
// (GraphQLProvider.EnrolledMFAMethods) directly, matching the integration-test
// style of this package. The field is proven lazy separately by the generated
// exec wiring (_User_enrolled_mfa_methods calls ec.resolvers.User().
// EnrolledMfaMethods only when the field is selected — it never reads a struct
// field), not at runtime here.
func TestEnrolledMFAMethods(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	newUser := func(t *testing.T) string {
		t.Helper()
		email := "enrolled_mfa_" + uuid.NewString() + "@authorizer.dev"
		password := "Password@123"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		u, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		// With EnableMFA/EnableTOTPLogin on, the signup MFA offer pre-creates an
		// unverified TOTP authenticator row. Clear it so each test starts from a
		// known-empty enrollment state and controls exactly which rows exist.
		require.NoError(t, ts.StorageProvider.DeleteAuthenticatorsByUserID(ctx, u.ID))
		return u.ID
	}

	t.Run("totp and passkey enrolled, otp not", func(t *testing.T) {
		userID := newUser(t)
		now := time.Now().Unix()

		_, err := ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     userID,
			Method:     constants.EnvKeyTOTPAuthenticator,
			Secret:     "test-secret",
			VerifiedAt: &now,
		})
		require.NoError(t, err)
		_, err = ts.StorageProvider.AddWebauthnCredential(ctx, &schemas.WebauthnCredential{
			UserID:       userID,
			CredentialID: uuid.NewString(),
			PublicKey:    "test-public-key",
		})
		require.NoError(t, err)

		methods, err := ts.GraphQLProvider.EnrolledMFAMethods(ctx, &model.User{ID: userID})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			constants.EnvKeyTOTPAuthenticator,  // "totp"
			constants.AuthRecipeMethodWebauthn, // "webauthn"
		}, methods, "must report exactly the verified/enrolled factors")
	})

	t.Run("nothing enrolled returns empty non-nil slice", func(t *testing.T) {
		userID := newUser(t)

		methods, err := ts.GraphQLProvider.EnrolledMFAMethods(ctx, &model.User{ID: userID})
		require.NoError(t, err)
		require.NotNil(t, methods, "field is [String!]! — must never be nil")
		assert.Empty(t, methods)
	})

	t.Run("unverified authenticator is excluded, verified email otp included", func(t *testing.T) {
		userID := newUser(t)
		now := time.Now().Unix()

		// A TOTP row that was never verified (VerifiedAt == nil) — half-enrolled.
		// It must NOT count as an enrolled method.
		_, err := ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID: userID,
			Method: constants.EnvKeyTOTPAuthenticator,
			Secret: "unverified-secret",
		})
		require.NoError(t, err)
		// A verified email-OTP authenticator — must count.
		_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     userID,
			Method:     constants.EnvKeyEmailOTPAuthenticator,
			Secret:     "email-otp-secret",
			VerifiedAt: &now,
		})
		require.NoError(t, err)

		methods, err := ts.GraphQLProvider.EnrolledMFAMethods(ctx, &model.User{ID: userID})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{constants.EnvKeyEmailOTPAuthenticator}, methods,
			"unverified TOTP must be excluded; verified email OTP must be included")
	})
}
