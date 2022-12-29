package test

import (
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func validateJwtTokenTest(t *testing.T, s TestSetup) {
	t.Helper()
	_, ctx := createContext(s)
	t.Run(`validate params`, func(t *testing.T) {
		res, err := resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "access_token",
			Token:     "",
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		res, err = resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "access_token",
			Token:     "invalid",
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		_, err = resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "access_token_invalid",
			Token:     "invalid@invalid",
		})
		assert.Error(t, err, "invalid token")
	})

	scope := []string{"openid", "email", "profile", "offline_access"}
	user := models.User{
		ID:        uuid.New().String(),
		Email:     "jwt_test_" + s.TestInfo.Email,
		Roles:     "user",
		UpdatedAt: time.Now().Unix(),
		CreatedAt: time.Now().Unix(),
	}

	roles := []string{"user"}
	gc, err := utils.GinContextFromContext(ctx)
	assert.NoError(t, err)
	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	nonce := uuid.New().String()
	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodBasicAuth, nonce, "")
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token)

	if authToken.RefreshToken != nil {
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
	}

	t.Run(`should validate the access token`, func(t *testing.T) {
		res, err := resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "access_token",
			Token:     authToken.AccessToken.Token,
			Roles:     []string{"user"},
		})
		assert.NoError(t, err)
		assert.True(t, res.IsValid)

		res, err = resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "access_token",
			Token:     authToken.AccessToken.Token,
			Roles:     []string{"invalid_role"},
		})

		assert.Error(t, err)
	})

	t.Run(`should validate the refresh token`, func(t *testing.T) {
		res, err := resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "refresh_token",
			Token:     authToken.RefreshToken.Token,
		})
		assert.NoError(t, err)
		assert.True(t, res.IsValid)
	})

	t.Run(`should validate the id token`, func(t *testing.T) {
		res, err := resolvers.ValidateJwtTokenResolver(ctx, model.ValidateJWTTokenInput{
			TokenType: "id_token",
			Token:     authToken.IDToken.Token,
		})
		assert.NoError(t, err)
		assert.True(t, res.IsValid)
		assert.Equal(t, user.Email, res.Claims["email"])
	})
}
