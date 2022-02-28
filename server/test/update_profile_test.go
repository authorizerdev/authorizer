package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func updateProfileTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update the profile with access token only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "update_profile." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		fName := "samani"
		_, err := resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			FamilyName: &fName,
		})
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeBasicAuthSignup)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token", token))
		_, err = resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			FamilyName: &fName,
		})
		assert.Nil(t, err)

		newEmail := "new_" + email
		_, err = resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			Email: &newEmail,
		})
		assert.Nil(t, err)
		_, err = resolvers.ProfileResolver(ctx)
		assert.NotNil(t, err, "unauthorized")

		cleanData(newEmail)
		cleanData(email)
	})
}
