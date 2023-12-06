package test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/authenticators"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/gokyle/twofactor"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/qrcode"
)

func totpSignupTest(t *testing.T, s TestSetup) {
	t.Helper()
	// Test case to verify TOTP for signup
	t.Run(`should verify totp for signup`, func(t *testing.T) {
		// Create request and context using test setup
		req, ctx := createContext(s)
		email := "verify_totp." + s.TestInfo.Email

		// Test case: Invalid password (confirm password mismatch)
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)

		{
			// Test case: Invalid password ("test" as the password)
			res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        "test",
				ConfirmPassword: "test",
			})
			assert.NotNil(t, err, "invalid password")
			assert.Nil(t, res)
		}

		{
			// Test case: Signup disabled
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
			res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        s.TestInfo.Password,
				ConfirmPassword: s.TestInfo.Password,
			})
			assert.NotNil(t, err, "signup disabled")
			assert.Nil(t, res)
		}

		{
			// Test case: Successful signup
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
			res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        s.TestInfo.Password,
				ConfirmPassword: s.TestInfo.Password,
				AppData: map[string]interface{}{
					"test": "test",
				},
			})
			assert.Nil(t, err, "signup should be successful")
			user := *res.User
			assert.Equal(t, email, refs.StringValue(user.Email))
			assert.Equal(t, "test", user.AppData["test"])
			assert.Nil(t, res.AccessToken, "access token should be nil")
		}

		{
			// Test case: Duplicate email (should throw an error)
			res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        s.TestInfo.Password,
				ConfirmPassword: s.TestInfo.Password,
			})
			assert.NotNil(t, err, "should throw duplicate email error")
			assert.Nil(t, res)
		}

		// Clean up data for the email
		cleanData(email)

		{
			// Test case: Email verification and TOTP setup
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableEmailVerification, false)

			// Sign up a user
			res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
				Email:           refs.NewStringRef(email),
				Password:        s.TestInfo.Password,
				ConfirmPassword: s.TestInfo.Password,
			})
			assert.Nil(t, err, "Expected no error but got: %v", err)
			assert.Equal(t, "Verification email has been sent. Please check your inbox", res.Message)

			// Retrieve user and update for TOTP setup
			user, err := db.Provider.GetUserByID(ctx, res.User.ID)
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

			// Get TOTP URL for for validation
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

			// Set MFA cookie session
			mfaSession := uuid.NewString()
			memorystore.Provider.SetMfaSession(res.User.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
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
			sessionKey := res.User.ID
			if signUpMethod != nil && signUpMethod != "" {
				sessionKey = signUpMethod.(string) + ":" + res.User.ID
			}
			sessionToken, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+claims["nonce"].(string))
			assert.NoError(t, err)
			assert.NotEmpty(t, sessionToken)
			cookie = fmt.Sprintf("%s=%s;", constants.AppCookieName+"_session", sessionToken)
			cookie = strings.TrimSuffix(cookie, ";")
			req.Header.Set("Cookie", cookie)
		}
		// Clean up data for the email
		cleanData(email)
	})
}
