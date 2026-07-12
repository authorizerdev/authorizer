package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMeta tests the meta query returns correct flags based on config
func TestMeta(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should return meta with default config", func(t *testing.T) {
		meta, err := ts.GraphQLProvider.Meta(ctx)
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, cfg.ClientID, meta.ClientID)
		assert.True(t, meta.IsSignUpEnabled)
		assert.True(t, meta.IsBasicAuthenticationEnabled)
	})

	t.Run("should reflect disabled signup", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableSignup = false
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		meta, err := ts2.GraphQLProvider.Meta(ctx2)
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.False(t, meta.IsSignUpEnabled)
	})

	t.Run("should reflect disabled basic auth", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableBasicAuthentication = false
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		meta, err := ts2.GraphQLProvider.Meta(ctx2)
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.False(t, meta.IsBasicAuthenticationEnabled)
	})

	t.Run("should expose per-method MFA availability with default config", func(t *testing.T) {
		meta, err := ts.GraphQLProvider.Meta(ctx)
		require.NoError(t, err)
		require.NotNil(t, meta)
		// Default config has MFA disabled, so all OTP/TOTP methods are unavailable.
		assert.False(t, meta.IsTotpMfaEnabled)
		assert.False(t, meta.IsEmailOtpMfaEnabled)
		assert.False(t, meta.IsSmsOtpMfaEnabled)
		// WebAuthn/passkey ships always-on.
		assert.True(t, meta.IsWebauthnEnabled)
	})

	t.Run("should enable TOTP MFA when MFA and TOTP login are on", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableMFA = true
		cfg2.EnableTOTPLogin = true
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		meta, err := ts2.GraphQLProvider.Meta(ctx2)
		require.NoError(t, err)
		require.NotNil(t, meta)
		assert.True(t, meta.IsTotpMfaEnabled)
		// Email/SMS OTP still require their own service + flag.
		assert.False(t, meta.IsEmailOtpMfaEnabled)
		assert.False(t, meta.IsSmsOtpMfaEnabled)
		assert.True(t, meta.IsWebauthnEnabled)
	})

	t.Run("should gate email/SMS OTP MFA on service availability", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableMFA = true
		cfg2.EnableEmailOTP = true
		cfg2.EnableSMSOTP = true
		cfg2.IsEmailServiceEnabled = true
		cfg2.IsSMSServiceEnabled = true
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		meta, err := ts2.GraphQLProvider.Meta(ctx2)
		require.NoError(t, err)
		require.NotNil(t, meta)
		assert.True(t, meta.IsEmailOtpMfaEnabled)
		assert.True(t, meta.IsSmsOtpMfaEnabled)
		assert.False(t, meta.IsTotpMfaEnabled)

		// OTP flags flip off when the underlying service is unavailable, even with MFA on.
		cfg3 := getTestConfig()
		cfg3.EnableMFA = true
		cfg3.EnableEmailOTP = true
		cfg3.EnableSMSOTP = true
		cfg3.IsEmailServiceEnabled = false
		cfg3.IsSMSServiceEnabled = false
		ts3 := initTestSetup(t, cfg3)
		_, ctx3 := createContext(ts3)

		meta2, err := ts3.GraphQLProvider.Meta(ctx3)
		require.NoError(t, err)
		require.NotNil(t, meta2)
		assert.False(t, meta2.IsEmailOtpMfaEnabled)
		assert.False(t, meta2.IsSmsOtpMfaEnabled)
	})
}
