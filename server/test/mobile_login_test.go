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

func mobileLoginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should login via mobile`, func(t *testing.T) {
		_, ctx := createContext(s)
		phoneNumber := "2234567890"
		signUpRes, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, signUpRes)
		// should fail because phone is not verified
		res, err := resolvers.LoginResolver(ctx, model.LoginInput{
			PhoneNumber: refs.NewStringRef(phoneNumber),
			Password:    s.TestInfo.Password,
		})
		assert.NotNil(t, err, "should fail because phone is not verified")
		assert.Nil(t, res)
		smsRequest, err := db.Provider.GetOTPByPhoneNumber(ctx, phoneNumber)
		assert.NoError(t, err)
		assert.NotEmpty(t, smsRequest.Otp)
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
		verifySMSRequest, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			PhoneNumber: &phoneNumber,
			Otp:         smsRequest.Otp,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifySMSRequest.Message, "", "message should not be empty")
		assert.NotEmpty(t, verifySMSRequest.AccessToken)
		assert.NotEmpty(t, verifySMSRequest.IDToken)
	})
}
