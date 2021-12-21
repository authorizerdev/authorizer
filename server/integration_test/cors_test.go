package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/middlewares"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	constants.ENV_PATH = "../../.env.local"
	env.InitEnv()
	r := gin.Default()
	r.Use(location.Default())
	r.Use(middlewares.GinContextToContextMiddleware())
	r.Use(middlewares.CORSMiddleware())
	allowedOrigin := "http://localhost:8080" // The allowed origin that you want to check
	notAllowedOrigin := "http://myapp.com"

	server := httptest.NewServer(r)
	defer server.Close()

	client := &http.Client{}
	req, _ := http.NewRequest(
		"GET",
		"http://"+server.Listener.Addr().String()+"/api",
		nil,
	)
	req.Header.Add("Origin", allowedOrigin)

	get, _ := client.Do(req)

	// You should get your origin (or a * depending on your config) if the
	// passed origin is allowed.
	o := get.Header.Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, o, notAllowedOrigin, "Origins should not match")
	assert.Equal(t, o, allowedOrigin, "Origins don't match")
}
