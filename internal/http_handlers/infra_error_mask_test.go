package http_handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/internalerr"
)

const redisDialMsg = "dial tcp 10.0.0.5:6379: connect: connection refused"

// TestGraphQLErrorPresenter_MasksInfraProviderError: a Redis-style fault
// marked at the memory-store boundary is masked on GraphQL exactly like a
// storage error (issue #356), and logged with its error_id and source.
func TestGraphQLErrorPresenter_MasksInfraProviderError(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	gqlErr := graphQLErrorPresenter(&log)(context.Background(), fmt.Errorf("login: %w", internalerr.Wrap("memory_store", errors.New(redisDialMsg))))
	require.NotNil(t, gqlErr)
	assert.NotContains(t, gqlErr.Message, "6379")
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, gqlErr.Message)
	assert.Equal(t, "INTERNAL", gqlErr.Extensions["code"])

	id := gqlErr.Message[len("internal server error (error_id: ") : len(gqlErr.Message)-1]
	logged := buf.String()
	assert.Contains(t, logged, redisDialMsg)
	assert.Contains(t, logged, `"error_id":"`+id+`"`)
	assert.Contains(t, logged, `"error_source":"memory_store"`)
}

// TestPublicErrorMessage_InfraProviderError: the plain-HTTP helper applies
// the same rule, and an unmarked user-facing message passes through.
func TestPublicErrorMessage_InfraProviderError(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	msg := publicErrorMessage(&log, internalerr.Wrap("token", errors.New("asn1: structure error")))
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, msg)
	assert.Contains(t, buf.String(), `"error_source":"token"`)

	assert.Equal(t, "unsupported provider: foo", publicErrorMessage(&log, errors.New("unsupported provider: foo")))
}

// TestJWKsHandler_MasksKeyError: an unrenderable public key is server
// misconfiguration; its parse error must not be published on the
// unauthenticated JWKS endpoint.
func TestJWKsHandler_MasksKeyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	h := &httpProvider{
		Config:       &config.Config{JWTType: "RS256", JWTPublicKey: "not a pem key"},
		Dependencies: Dependencies{Log: &log},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	h.JWKsHandler()(c)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, body["error"])
	assert.Contains(t, buf.String(), `"error_source":"jwks"`)
}
