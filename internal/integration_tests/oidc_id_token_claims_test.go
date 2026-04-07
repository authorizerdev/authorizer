package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// createAuthTokenForIDTokenClaimsTest is a tiny local helper that signs
// up a user and returns a minted AuthToken via CreateAuthToken. Uses
// only public provider APIs and deliberately avoids touching
// test_helper.go to keep the fixture self-contained.
func createAuthTokenForIDTokenClaimsTest(t *testing.T, loginMethod string, authTime int64) (*token.AuthToken, *testSetup) {
	t.Helper()
	cfg := getTestConfig()
	_, privateKey, publicKey, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	cfg.JWTType = "RS256"
	cfg.JWTPrivateKey = privateKey
	cfg.JWTPublicKey = publicKey
	cfg.JWTSecret = ""
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "id_token_claims_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	authToken, err := ts.TokenProvider.CreateAuthToken(nil, &token.AuthTokenConfig{
		User:        user,
		Roles:       []string{"user"},
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: loginMethod,
		Nonce:       "nonce-" + uuid.New().String(),
		HostName:    "http://localhost",
		AuthTime:    authTime,
	})
	require.NoError(t, err)
	return authToken, ts
}

func TestIDTokenAuthTimeClaim(t *testing.T) {
	t.Run("explicit_auth_time_is_echoed", func(t *testing.T) {
		expectedAuthTime := int64(1700000000)
		authToken, ts := createAuthTokenForIDTokenClaimsTest(t, "basic_auth", expectedAuthTime)
		claims, err := ts.TokenProvider.ParseJWTToken(authToken.IDToken.Token)
		require.NoError(t, err)

		got, ok := claims["auth_time"]
		require.True(t, ok, "auth_time claim MUST be present")
		// JSON-decoded numbers are float64 in Go.
		switch v := got.(type) {
		case float64:
			assert.Equal(t, float64(expectedAuthTime), v)
		case int64:
			assert.Equal(t, expectedAuthTime, v)
		default:
			t.Fatalf("auth_time claim has unexpected type %T", got)
		}
	})

	t.Run("zero_auth_time_defaults_to_now", func(t *testing.T) {
		authToken, ts := createAuthTokenForIDTokenClaimsTest(t, "basic_auth", 0)
		claims, err := ts.TokenProvider.ParseJWTToken(authToken.IDToken.Token)
		require.NoError(t, err)
		got, ok := claims["auth_time"]
		require.True(t, ok, "auth_time claim MUST be present even when caller did not supply one")
		// Should be non-zero and within a reasonable window of now.
		switch v := got.(type) {
		case float64:
			assert.Greater(t, v, float64(0), "auth_time MUST be > 0 after default fill-in")
		case int64:
			assert.Greater(t, v, int64(0))
		default:
			t.Fatalf("auth_time claim has unexpected type %T", got)
		}
	})
}

func TestIDTokenAmrClaim(t *testing.T) {
	tests := []struct {
		name        string
		loginMethod string
		wantPresent bool
		wantAmr     []string
	}{
		{"basic_auth_maps_to_pwd", "basic_auth", true, []string{"pwd"}},
		{"mobile_basic_auth_maps_to_pwd", "mobile_basic_auth", true, []string{"pwd"}},
		{"magic_link_maps_to_otp", "magic_link_login", true, []string{"otp"}},
		{"mobile_otp_maps_to_otp", "mobile_otp", true, []string{"otp"}},
		{"google_maps_to_fed", "google", true, []string{"fed"}},
		{"github_maps_to_fed", "github", true, []string{"fed"}},
		{"unknown_method_omits_claim", "weird_thing_xyz", false, nil},
		{"empty_method_omits_claim", "", false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authToken, ts := createAuthTokenForIDTokenClaimsTest(t, tc.loginMethod, 1700000000)
			claims, err := ts.TokenProvider.ParseJWTToken(authToken.IDToken.Token)
			require.NoError(t, err)
			amrRaw, present := claims["amr"]
			if !tc.wantPresent {
				assert.False(t, present, "amr claim MUST be omitted for login method %q", tc.loginMethod)
				return
			}
			require.True(t, present, "amr claim MUST be present for login method %q", tc.loginMethod)
			// JSON arrays decode to []interface{}.
			arr, ok := amrRaw.([]interface{})
			require.True(t, ok, "amr must decode as []interface{}")
			require.Len(t, arr, len(tc.wantAmr))
			for i, want := range tc.wantAmr {
				assert.Equal(t, want, arr[i])
			}
		})
	}
}

func TestIDTokenAcrClaim(t *testing.T) {
	authToken, ts := createAuthTokenForIDTokenClaimsTest(t, "basic_auth", 1700000000)
	claims, err := ts.TokenProvider.ParseJWTToken(authToken.IDToken.Token)
	require.NoError(t, err)
	got, ok := claims["acr"].(string)
	require.True(t, ok, "acr claim MUST be present and a string")
	assert.Equal(t, "0", got,
		"acr is hardcoded to \"0\" (minimal assurance) pending MFA-aware ACR support")
}
