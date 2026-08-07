package http_handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/config"
)

// TestCORSMiddleware pins the one rule that matters here: a reflected origin and
// Access-Control-Allow-Credentials must never appear together under a wildcard
// allow-list. That pairing is "any site may read credentialed responses from
// this API", which the Fetch spec forbids for exactly that reason.
func TestCORSMiddleware(t *testing.T) {
	t.Parallel()

	run := func(allowedOrigins []string, requestOrigin string) http.Header {
		gin.SetMode(gin.TestMode)
		logger := zerolog.Nop()
		h := &httpProvider{
			Config:       &config.Config{AllowedOrigins: allowedOrigins},
			Dependencies: Dependencies{Log: &logger},
		}
		router := gin.New()
		router.Use(h.CORSMiddleware())
		router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		if requestOrigin != "" {
			req.Header.Set("Origin", requestOrigin)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Header()
	}

	t.Run("wildcard returns * and never credentials", func(t *testing.T) {
		t.Parallel()
		hdr := run([]string{"*"}, "https://evil.example.com")
		assert.Equal(t, "*", hdr.Get("Access-Control-Allow-Origin"),
			"the caller's origin must not be reflected back under a wildcard")
		assert.Empty(t, hdr.Get("Access-Control-Allow-Credentials"),
			"credentials must never be combined with a wildcard")
	})

	t.Run("unset allow-list behaves as wildcard", func(t *testing.T) {
		t.Parallel()
		hdr := run(nil, "https://evil.example.com")
		assert.Equal(t, "*", hdr.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, hdr.Get("Access-Control-Allow-Credentials"))
	})

	t.Run("explicit allow-list reflects the origin with credentials", func(t *testing.T) {
		t.Parallel()
		hdr := run([]string{"https://app.example.com"}, "https://app.example.com")
		assert.Equal(t, "https://app.example.com", hdr.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", hdr.Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, hdr.Values("Vary"), "Origin",
			"a per-origin response must not be cached and replayed to another origin")
	})

	t.Run("a disallowed origin gets neither header", func(t *testing.T) {
		t.Parallel()
		hdr := run([]string{"https://app.example.com"}, "https://evil.example.com")
		assert.Empty(t, hdr.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, hdr.Get("Access-Control-Allow-Credentials"))
	})
}
