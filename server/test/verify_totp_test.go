package test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/gokyle/twofactor"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/qrcode"
	"strings"
	"testing"
)

func verifyTOTPTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should verify totp`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "verify_totp." + s.TestInfo.Email
		cleanData(email)
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Login should fail as email is not verified
		loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, loginRes)
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
		updateProfileRes, err := resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, updateProfileRes.Message)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableTOTPLogin, false)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMailOTPLogin, true)

		// Login should not return error but access token should be empty
		loginRes, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.NotNil(t, loginRes.TotpBase64url)
		assert.NotNil(t, loginRes.TokenTotp)
		assert.Nil(t, loginRes.AccessToken)
		assert.Equal(t, loginRes.Message, `Proceed to totp screen`)

		// get totp url for validation
		pngBytes, err := base64.StdEncoding.DecodeString(*loginRes.TotpBase64url)
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

		valid, err := resolvers.VerifyTotpResolver(ctx, model.VerifyTOTPRequest{
			Otp:   code,
			Token: *loginRes.TokenTotp,
		})

		accessToken := *valid.AccessToken
		assert.NoError(t, err)
		assert.NotNil(t, accessToken)
		assert.Equal(t, `Logged in successfully`, valid.Message)

		assert.NotEmpty(t, accessToken)
		claims, err := token.ParseJWTToken(accessToken)
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

		cookie := fmt.Sprintf("%s=%s;", constants.AppCookieName+"_session", sessionToken)
		cookie = strings.TrimSuffix(cookie, ";")

		req.Header.Set("Cookie", cookie)
		//logged out
		logout, err := resolvers.LogoutResolver(ctx)
		assert.NoError(t, err)
		assert.Equal(t, logout.Message, `Logged out successfully`)

		loginRes, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.NotNil(t, loginRes.TokenTotp)
		assert.Nil(t, loginRes.TotpBase64url)
		assert.Nil(t, loginRes.AccessToken)
		assert.Equal(t, loginRes.Message, `Proceed to totp screen`)

		code = tf.OTP()
		assert.NotEmpty(t, code)

		valid, err = resolvers.VerifyTotpResolver(ctx, model.VerifyTOTPRequest{
			Otp:   code,
			Token: *loginRes.TokenTotp,
		})
		assert.NoError(t, err)
		assert.NotNil(t, *valid.AccessToken)
		assert.Equal(t, `Logged in successfully`, valid.Message)

		cleanData(email)
	})
}
