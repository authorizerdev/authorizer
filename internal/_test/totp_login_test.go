package test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/gokyle/twofactor"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/qrcode"
)

func totpLoginTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should verify totp`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "verify_totp." + s.TestInfo.Email
		cleanData(email)
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           &email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Login should fail as email is not verified
		loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    &email,
			Password: s.TestInfo.Password,
		})
		// access token should be empty as email is not verified
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.Nil(t, loginRes.AccessToken)
		assert.NotEmpty(t, loginRes.Message)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.Nil(t, err)
		assert.Equal(t, email, verificationRequest.Email)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")

		// Using access token update profile
		s.GinContext.Request.Header.Set("Authorization", "Bearer "+refs.StringValue(verifyRes.AccessToken))
		ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableTOTPLogin, false)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMailOTPLogin, true)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisablePhoneVerification, true)
		updateProfileRes, err := resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, updateProfileRes.Message)

		authenticators.InitTOTPStore()
		// Login should not return error but access token should be empty
		loginRes, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    &email,
			Password: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.True(t, *loginRes.ShouldShowTotpScreen)
		assert.NotNil(t, *loginRes.AuthenticatorScannerImage)
		assert.NotNil(t, *loginRes.AuthenticatorSecret)
		assert.NotNil(t, loginRes.AuthenticatorRecoveryCodes)
		assert.Nil(t, loginRes.AccessToken)
		assert.NotEmpty(t, loginRes.Message)

		// get totp url for validation
		pngBytes, err := base64.StdEncoding.DecodeString(*loginRes.AuthenticatorScannerImage)
		assert.NoError(t, err)
		qrmatrix, err := qrcode.Decode(bytes.NewReader(pngBytes))
		assert.NoError(t, err)
		tf, label, err := twofactor.FromURL(qrmatrix.Content)
		data := strings.Split(label, ":")
		assert.NoError(t, err)
		assert.Equal(t, email, data[1])
		assert.NotNil(t, tf)
		code := tf.OTP()
		assert.NotEmpty(t, code)

		// Set mfa cookie session
		mfaSession := uuid.NewString()
		memorystore.Provider.SetMfaSession(verifyRes.User.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
		cookie := fmt.Sprintf("%s=%s;", constants.MfaCookieName+"_session", mfaSession)
		cookie = strings.TrimSuffix(cookie, ";")
		req.Header.Set("Cookie", cookie)
		valid, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			Email:  &email,
			IsTotp: refs.NewBoolRef(true),
			Otp:    code,
		})
		accessToken := valid.AccessToken
		assert.NoError(t, err)
		assert.NotNil(t, accessToken)
		assert.NotEmpty(t, valid.Message)
		assert.NotEmpty(t, accessToken)
		claims, err := token.ParseJWTToken(*accessToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, claims)
		loginMethod := claims["login_method"]
		sessionKey := verifyRes.User.ID
		if loginMethod != nil && loginMethod != "" {
			sessionKey = loginMethod.(string) + ":" + verifyRes.User.ID
		}
		sessionToken, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+claims["nonce"].(string))
		assert.NoError(t, err)
		assert.NotEmpty(t, sessionToken)
		cookie = fmt.Sprintf("%s=%s;", constants.AppCookieName+"_session", sessionToken)
		cookie = strings.TrimSuffix(cookie, ";")
		req.Header.Set("Cookie", cookie)

		//logged out
		logout, err := resolvers.LogoutResolver(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, logout.Message)
		loginRes, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    &email,
			Password: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.Nil(t, loginRes.AuthenticatorRecoveryCodes)
		assert.Nil(t, loginRes.AccessToken)
		assert.Nil(t, loginRes.AuthenticatorScannerImage)
		assert.Nil(t, loginRes.AuthenticatorSecret)
		assert.True(t, *loginRes.ShouldShowTotpScreen)
		assert.NotEmpty(t, loginRes.Message)
		code = tf.OTP()
		assert.NotEmpty(t, code)
		// Set mfa cookie session
		mfaSession = uuid.NewString()
		memorystore.Provider.SetMfaSession(verifyRes.User.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
		cookie = fmt.Sprintf("%s=%s;", constants.MfaCookieName+"_session", mfaSession)
		cookie = strings.TrimSuffix(cookie, ";")
		req.Header.Set("Cookie", cookie)
		valid, err = resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			Otp:    code,
			Email:  &email,
			IsTotp: refs.NewBoolRef(true),
		})
		assert.NoError(t, err)
		assert.NotNil(t, *valid.AccessToken)
		assert.NotEmpty(t, valid.Message)
		cleanData(email)
	})
}
