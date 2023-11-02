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
		phoneNumber := "1234567890"
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Password:        "test",
			ConfirmPassword: "test",
		})
		assert.Error(t, err, "phone number or email should be provided")
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication, true)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication, false)

		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef("   "),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef("test"),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
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
			PhoneNumber: refs.NewStringRef(phoneNumber),
			Otp:         otp.Otp,
		})
		assert.Nil(t, err)
		assert.NotEmpty(t, otpRes.Message)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		cleanData("1234567890@authorizer.dev")
	})
}
