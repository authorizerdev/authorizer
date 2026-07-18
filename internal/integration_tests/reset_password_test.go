package integration_tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure imports are used
var _ = time.Now
var _ = fmt.Sprintf
var _ = strings.Contains

// TestResetPassword tests the reset password functionality
func TestResetPassword(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "reset_password_test_" + uuid.New().String() + "@authorizer.dev"
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

	// Create forgot password request
	t.Run("should fail for invalid request", func(t *testing.T) {
		resetPasswordReq := &model.ResetPasswordRequest{
			Token:           refs.NewStringRef("test"),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.Error(t, err)
		assert.Nil(t, forgotPasswordRes)
	})

	t.Run("should fail for password mismatch", func(t *testing.T) {
		resetPasswordReq := &model.ResetPasswordRequest{
			Token:           refs.NewStringRef("test"),
			Password:        "NewPassword@123",
			ConfirmPassword: "DifferentPassword@123",
		}
		res, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail reset password with expired OTP", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.IsSMSServiceEnabled = true
		cfg2.EnableMobileBasicAuthentication = true
		cfg2.EnablePhoneVerification = true
		ts2 := initTestSetup(t, cfg2)
		req2, ctx2 := createContext(ts2)

		mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
		signupReq2 := &model.SignUpRequest{
			PhoneNumber:     &mobile,
			Password:        password,
			ConfirmPassword: password,
		}
		_, err := ts2.GraphQLProvider.SignUp(ctx2, signupReq2)
		require.NoError(t, err)

		// Get user and OTP
		user, err := ts2.StorageProvider.GetUserByPhoneNumber(ctx2, mobile)
		require.NoError(t, err)

		otpData, err := ts2.StorageProvider.GetOTPByPhoneNumber(ctx2, mobile)
		require.NoError(t, err)

		// Seed an expired OTP with a known plaintext/digest pair so the
		// reset call gets past the OTP-match check and trips the expiry
		// check (the behaviour this subtest exercises). The hardening
		// stores the digest, never the plaintext.
		const knownExpiredPlainOTP = "111111"
		expiredOTP := &schemas.OTP{
			ID:          otpData.ID,
			Email:       otpData.Email,
			PhoneNumber: otpData.PhoneNumber,
			Otp:         crypto.HashOTP(knownExpiredPlainOTP, ts2.Config.JWTSecret),
			ExpiresAt:   time.Now().Add(-10 * time.Minute).Unix(),
		}
		_, err = ts2.StorageProvider.UpsertOTP(ctx2, expiredOTP)
		require.NoError(t, err)

		// Get MFA session
		allData, err := ts2.MemoryStoreProvider.GetAllData()
		require.NoError(t, err)
		sessionKey := ""
		for k := range allData {
			if strings.Contains(k, user.ID) {
				splitData := strings.Split(k, ":")
				if len(splitData) > 1 {
					sessionKey = splitData[1]
					break
				}
			}
		}
		req2.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", sessionKey))

		resetReq := &model.ResetPasswordRequest{
			Otp:             refs.NewStringRef(knownExpiredPlainOTP),
			PhoneNumber:     refs.NewStringRef(mobile),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}
		res, err := ts2.GraphQLProvider.ResetPassword(ctx2, resetReq)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("should revoke existing sessions after reset", func(t *testing.T) {
		// A password reset must lock out anyone holding a pre-existing
		// session/refresh token. Log in to mint a session + refresh token,
		// reset the password, then assert the session is gone from the store
		// and the old refresh token is no longer honoured by /oauth/token.
		//
		// NOTE: refresh tokens are single-use (rotated on a successful
		// refresh), so we must NOT poll the refresh endpoint to detect
		// revocation — one successful refresh would itself invalidate the
		// original token and mask the bug. We poll the (non-mutating) memory
		// store instead, then hit /oauth/token exactly once.
		revokeEmail := "reset_revoke_" + uuid.New().String() + "@authorizer.dev"
		signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &revokeEmail,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		userID := signupRes.User.ID

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &revokeEmail,
			Password: password,
			Scope:    []string{"openid", "email", "profile", "offline_access"},
		})
		require.NoError(t, err)
		require.NotNil(t, loginRes.RefreshToken)

		hasUserSession := func() bool {
			allData, err := ts.MemoryStoreProvider.GetAllData()
			require.NoError(t, err)
			for k := range allData {
				if strings.Contains(k, userID) {
					return true
				}
			}
			return false
		}
		// Baseline: login created a live session for the user.
		require.True(t, hasUserSession(), "login should create a session before reset")

		// Reset the password via the forgot-password verification token.
		forgotRes, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{
			Email: refs.NewStringRef(revokeEmail),
		})
		require.NoError(t, err)
		require.NotNil(t, forgotRes)
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, revokeEmail, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)
		resetRes, err := ts.GraphQLProvider.ResetPassword(ctx, &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(request.Token),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		})
		require.NoError(t, err)
		require.NotNil(t, resetRes)

		// Revocation is now a synchronous call rather than a fire-and-forget
		// goroutine, so it must have already happened by the time
		// ResetPassword returns - asserted immediately, no polling needed.
		// NOTE: this assertion alone does not reliably distinguish the fix
		// from the old fire-and-forget version - an in-memory single-map-
		// delete goroutine typically completes before the surrounding code
		// (LogEvent, metrics, building the return value) finishes anyway, so
		// the same assertion can pass by luck against the old code too. The
		// guarantee that matters here comes from the source (no goroutine =
		// no race, full stop), not from this test's timing sensitivity -
		// confirmed by reading internal/service/reset_password.go directly
		// rather than relying on this check alone.
		assert.False(t, hasUserSession(), "all user sessions must be revoked before password reset returns")

		// End-to-end: the pre-existing refresh token is now rejected.
		issuer := "http://" + ts.HttpServer.Listener.Addr().String()
		router := gin.New()
		router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", *loginRes.RefreshToken)
		form.Set("client_id", cfg.ClientID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", issuer)
		router.ServeHTTP(w, req)
		// RFC 6749 §5.2: invalid_grant responses MUST use HTTP 400, not 401.
		assert.Equal(t, http.StatusBadRequest, w.Code,
			"pre-existing refresh token must be rejected after password reset")
	})

	t.Run("should reset password with verification token", func(t *testing.T) {
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

		// Reset password using the token
		resetPasswordReq := &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(request.Token),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}

		resetPasswordRes, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, resetPasswordRes)
		assert.NotEmpty(t, resetPasswordRes.Message)

		// Validate if the password is updated in db by logging in
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: "NewPassword@123",
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.NotNil(t, loginRes.AccessToken)
	})
}
