package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func mobileSingupTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete the signup with mobile and check duplicates`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "mobile_basic_auth_signup." + s.TestInfo.Email
		res, err := resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        "test",
			ConfirmPassword: "test",
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication, true)
		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication, false)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "   ",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "test",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		phoneNumber := "1234567890"
		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     phoneNumber,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, *res.ShouldShowMobileOtpScreen)
		// Verify with otp
		otp, err := db.Provider.GetOTPByPhoneNumber(ctx, phoneNumber)
		assert.Nil(t, err)
		assert.NotEmpty(t, otp.Otp)
		// Get user by phone number
		user, err := db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		// Set mfa cookie session
		mfaSession := uuid.NewString()
		memorystore.Provider.SetMfaSession(user.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
		cookie := fmt.Sprintf("%s=%s;", constants.MfaCookieName+"_session", mfaSession)
		cookie = strings.TrimSuffix(cookie, ";")
		req, ctx := createContext(s)
		req.Header.Set("Cookie", cookie)
		otpRes, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			PhoneNumber: &phoneNumber,
			Otp:         otp.Otp,
		})
		assert.Nil(t, err)
		assert.NotEmpty(t, otpRes.Message)
		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "1234567890",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		cleanData(email)
		cleanData("1234567890@authorizer.dev")
	})
}
