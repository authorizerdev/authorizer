package test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gokyle/twofactor"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/qrcode"

	"github.com/authorizerdev/authorizer/server/authenticators"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
)

func verifyOTPTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should verify otp`, func(t *testing.T) {
		// Set up request and context using test setup
		req, ctx := createContext(s)
		email := "verify_otp." + s.TestInfo.Email

		// Test case: Setup email OTP MFA for login
		{
			// Sign up a user
			res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        s.TestInfo.Password,
				ConfirmPassword: s.TestInfo.Password,
			})
			assert.NoError(t, err)
			assert.NotNil(t, res)

			// Attempt to login should fail as email is not verified
			loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
				Email:    refs.NewStringRef(email),
				Password: s.TestInfo.Password,
			})
			assert.NoError(t, err)
			assert.NotNil(t, loginRes)
			assert.Nil(t, loginRes.AccessToken)
			assert.NotEmpty(t, loginRes.Message)

			// Verify the email
			verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
			assert.Nil(t, err)
			assert.Equal(t, email, verificationRequest.Email)
			verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
				Token: verificationRequest.Token,
			})
			assert.Nil(t, err)
			assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")

			// Use access token to update the profile
			s.GinContext.Request.Header.Set("Authorization", "Bearer "+refs.StringValue(verifyRes.AccessToken))
			ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMailOTPLogin, false)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableTOTPLogin, true)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisablePhoneVerification, true)
			updateProfileRes, err := resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
				IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
			})
			assert.NoError(t, err)
			assert.NotEmpty(t, updateProfileRes.Message)

			// Login should not return an error, but the access token should be empty as OTP should have been sent
			loginRes, err = resolvers.LoginResolver(ctx, model.LoginInput{
				Email:    refs.NewStringRef(email),
				Password: s.TestInfo.Password,
			})
			assert.NoError(t, err)
			assert.NotNil(t, loginRes)
			assert.Nil(t, loginRes.AccessToken)

			// Get OTP from db
			otp, err := db.Provider.GetOTPByEmail(ctx, email)
			assert.NoError(t, err)
			assert.NotEmpty(t, otp.Otp)

			// Get user by email
			user, err := db.Provider.GetUserByEmail(ctx, email)
			assert.NoError(t, err)
			assert.NotNil(t, user)

			// Set MFA cookie session
			mfaSession := uuid.NewString()
			memorystore.Provider.SetMfaSession(user.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
			cookie := fmt.Sprintf("%s=%s;", constants.MfaCookieName+"_session", mfaSession)
			cookie = strings.TrimSuffix(cookie, ";")
			req.Header.Set("Cookie", cookie)

			// Verify OTP
			verifyOtpRes, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
				Email: &email,
				Otp:   otp.Otp,
			})
			assert.Nil(t, err)
			assert.NotEqual(t, verifyOtpRes.AccessToken, "", "access token should not be empty")

			// Clean up data for the email
			cleanData(email)
		}

		// Test case: Setup TOTP MFA for signup
		{
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableEmailVerification, false)
			signUpRes, err := resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        s.TestInfo.Password,
				ConfirmPassword: s.TestInfo.Password,
			})
			assert.Nil(t, err, "Expected no error but got: %v", err)
			assert.Equal(t, "Verification email has been sent. Please check your inbox", signUpRes.Message)

			// Retrieve user and update for TOTP setup
			user, err := db.Provider.GetUserByID(ctx, signUpRes.User.ID)
			assert.Nil(t, err, "Expected no error but got: %v", err)
			assert.NotNil(t, user)

			// Enable multi-factor authentication and update the user
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableTOTPLogin, false)
			user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
			updatedUser, err := db.Provider.UpdateUser(ctx, user)
			assert.Nil(t, err, "Expected no error but got: %v", err)
			assert.Equal(t, true, *updatedUser.IsMultiFactorAuthEnabled)

			// Initialise totp authenticator store
			authenticators.InitTOTPStore()

			// Verify an email and get TOTP response
			verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
			assert.Nil(t, err)
			assert.Equal(t, email, verificationRequest.Email)
			verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
				Token: verificationRequest.Token,
			})
			assert.Nil(t, err, "Expected no error but got: %v", err)
			assert.NotNil(t, &verifyRes)
			assert.Nil(t, verifyRes.AccessToken)
			assert.Equal(t, "Proceed to totp verification screen", verifyRes.Message)
			assert.NotEqual(t, *verifyRes.AuthenticatorScannerImage, "", "totp url should not be empty")
			assert.NotEqual(t, *verifyRes.AuthenticatorSecret, "", "totp secret should not be empty")
			assert.NotNil(t, verifyRes.AuthenticatorRecoveryCodes)

			// Get TOTP URL for validation
			pngBytes, err := base64.StdEncoding.DecodeString(*verifyRes.AuthenticatorScannerImage)
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
			memorystore.Provider.SetMfaSession(signUpRes.User.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
			cookie := fmt.Sprintf("%s=%s;", constants.MfaCookieName+"_session", mfaSession)
			cookie = strings.TrimSuffix(cookie, ";")
			req.Header.Set("Cookie", cookie)
			valid, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
				Email:  &email,
				IsTotp: refs.NewBoolRef(true),
				Otp:    code,
			})
			accessToken := *valid.AccessToken
			assert.NoError(t, err)
			assert.NotNil(t, accessToken)
			assert.NotEmpty(t, valid.Message)
			assert.NotEmpty(t, accessToken)
			claims, err := token.ParseJWTToken(accessToken)
			assert.NoError(t, err)
			assert.NotEmpty(t, claims)
			signUpMethod := claims["login_method"]
			sessionKey := signUpRes.User.ID
			if signUpMethod != nil && signUpMethod != "" {
				sessionKey = signUpMethod.(string) + ":" + signUpRes.User.ID
			}
			sessionToken, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+claims["nonce"].(string))
			assert.NoError(t, err)
			assert.NotEmpty(t, sessionToken)
			cookie = fmt.Sprintf("%s=%s;", constants.AppCookieName+"_session", sessionToken)
			cookie = strings.TrimSuffix(cookie, ";")
			req.Header.Set("Cookie", cookie)

			// Clean up data for the email
			cleanData(email)
		}
	})
}
