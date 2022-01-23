package test

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func isValidJWTTests(t *testing.T, s TestSetup) {
	t.Helper()
	ctx := context.Background()
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd2VkX3JvbGVzIjpbIiJdLCJiaXJ0aGRhdGUiOm51bGwsImNyZWF0ZWRfYXQiOjAsImVtYWlsIjoiam9obi5kb2VAZ21haWwuY29tIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJleHAiOjE2NDI5NjEwMTEsImV4dHJhIjp7IngtZXh0cmEtaWQiOiJkMmNhMjQwNy05MzZmLTQwYzQtOTQ2NS05Y2M5MWYxZTJhNDQifSwiZmFtaWx5X25hbWUiOm51bGwsImdlbmRlciI6bnVsbCwiZ2l2ZW5fbmFtZSI6bnVsbCwiaWF0IjoxNjQyOTYwOTgxLCJpZCI6ImQyY2EyNDA3LTkzNmYtNDBjNC05NDY1LTljYzkxZjFlMmE0NCIsIm1pZGRsZV9uYW1lIjpudWxsLCJuaWNrbmFtZSI6bnVsbCwicGhvbmVfbnVtYmVyIjpudWxsLCJwaG9uZV9udW1iZXJfdmVyaWZpZWQiOmZhbHNlLCJwaWN0dXJlIjpudWxsLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJqb2huLmRvZUBnbWFpbC5jb20iLCJyb2xlIjpbXSwic2lnbnVwX21ldGhvZHMiOiIiLCJ0b2tlbl90eXBlIjoiYWNjZXNzX3Rva2VuIiwidXBkYXRlZF9hdCI6MH0.FrdyeOC5e8uU1SowGj0omFJuwRnh4BrEk89S_fbEkzs"

	t.Run(`should fail for invalid jwt`, func(t *testing.T) {
		_, err := resolvers.IsValidJwtResolver(ctx, &model.IsValidJWTQueryInput{
			Jwt: expiredToken,
		})
		assert.NotNil(t, err)
	})

	t.Run(`should pass with valid jwt`, func(t *testing.T) {
		authToken, err := token.CreateAuthToken(models.User{
			ID:    uuid.New().String(),
			Email: "john.doe@gmail.com",
		}, []string{})
		assert.Nil(t, err)
		res, err := resolvers.IsValidJwtResolver(ctx, &model.IsValidJWTQueryInput{
			Jwt: authToken.AccessToken.Token,
		})
		assert.Nil(t, err)
		assert.True(t, res.Valid)
	})
}
