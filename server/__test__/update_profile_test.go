package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func updateProfileTests(s TestSetup, t *testing.T) {
	t.Run(`should update the profile with access token only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "update_profile." + s.TestInfo.Email

		resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		fName := "samani"
		_, err := resolvers.UpdateProfile(ctx, model.UpdateProfileInput{
			FamilyName: &fName,
		})
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
		verifyRes, err := resolvers.VerifyEmail(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.COOKIE_NAME, token))
		_, err = resolvers.UpdateProfile(ctx, model.UpdateProfileInput{
			FamilyName: &fName,
		})
		assert.Nil(t, err)

		newEmail := "new_" + email
		_, err = resolvers.UpdateProfile(ctx, model.UpdateProfileInput{
			Email: &newEmail,
		})
		assert.Nil(t, err)
		_, err = resolvers.Profile(ctx)
		assert.NotNil(t, err, "unauthorized")

		cleanData(newEmail)
		cleanData(email)
	})
}
