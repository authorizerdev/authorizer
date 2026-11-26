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
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage"
)

const testDriverMsg = "mssql: Cannot insert duplicate key row in object 'dbo.authorizer_users' with unique index 'idx_email'"

func TestPublicErrorMessage(t *testing.T) {
	t.Run("storage error is masked and logged at error", func(t *testing.T) {
		var buf bytes.Buffer
		log := zerolog.New(&buf)
		msg := publicErrorMessage(&log, fmt.Errorf("consume: %w", &storage.Error{Err: errors.New(testDriverMsg)}))
		assert.NotContains(t, msg, "mssql")
		assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, msg)
		assert.Contains(t, buf.String(), testDriverMsg)
		assert.Contains(t, buf.String(), `"level":"error"`)
		assert.Contains(t, buf.String(), `"error_id":"`+msg[len("internal server error (error_id: "):len(msg)-1]+`"`)
	})

	t.Run("storage not-found is masked and logged at warn", func(t *testing.T) {
		var buf bytes.Buffer
		log := zerolog.New(&buf)
		msg := publicErrorMessage(&log, &storage.Error{Err: gorm.ErrRecordNotFound})
		assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, msg)
		assert.Contains(t, buf.String(), `"level":"warn"`)
	})

	t.Run("typed and plain errors are unchanged and not logged", func(t *testing.T) {
		var buf bytes.Buffer
		log := zerolog.New(&buf)
		assert.Equal(t, "invalid token", publicErrorMessage(&log, service.InvalidArgument("invalid token")))
		assert.Equal(t, "token is expired", publicErrorMessage(&log, errors.New("token is expired")))
		assert.Empty(t, buf.String())
	})
}

// consumeFailingService fails ConsumeEmailVerificationToken the way a storage
// outage does (raw storage error from GetUserByEmail/UpdateUser). The embedded
// nil Provider satisfies the rest of the interface; the handler must not reach it.
type consumeFailingService struct {
	service.Provider
	err error
}

func (s *consumeFailingService) ConsumeEmailVerificationToken(_ context.Context, _, _ string) (*service.EmailVerification, error) {
	return nil, s.err
}

// TestVerifyEmailHandler_MasksStorageError is the issue #356 regression guard
// for the plain HTTP surface: a storage failure behind /verify_email must not
// put driver text into the JSON body (or the redirect URL).
func TestVerifyEmailHandler_MasksStorageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	h := &httpProvider{
		Config: &config.Config{},
		Dependencies: Dependencies{
			Log:             &log,
			ServiceProvider: &consumeFailingService{err: &storage.Error{Err: errors.New(testDriverMsg)}},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/verify_email?token=abc", nil)
	h.VerifyEmailHandler()(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), "mssql")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, body["error"])
	assert.Contains(t, buf.String(), testDriverMsg, "original error must be logged server-side")
}
