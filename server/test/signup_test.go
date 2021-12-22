package test

import (
	"context"
	"log"
	"net/http/httptest"
	"testing"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSQLSignUp(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/graphql", nil)
	c, _ := gin.CreateTestContext(w)
	ctx := context.WithValue(req.Context(), "GinContextKey", c)

	res, err := resolvers.Signup(ctx, model.SignUpInput{
		Email:           "test@yopmail.com",
		Password:        "test",
		ConfirmPassword: "test",
	})
	log.Println("=> signup err:", err)
	log.Println("=> singup res:", res)
	assert.Equal(t, "success", "success")
}
