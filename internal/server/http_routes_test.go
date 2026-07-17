package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSpaBuildCacheMiddleware guards the entry-file allowlist that decides
// whether a build asset gets a revalidate-on-every-load header or a
// year-long immutable cache. web/app and web/dashboard's Vite configs
// disagree on the entry CSS filename (index.css vs main.css) - a request
// path whose base name isn't in the allowlist silently falls into the
// immutable branch, so a real deploy fixing a style bug would leave any
// browser that already cached the old file broken for up to a year.
func TestSpaBuildCacheMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		path        string
		wantNoCache bool
	}{
		{"/app/build/index.js", true},
		{"/app/build/index.css", true},
		{"/dashboard/build/index.js", true},
		{"/dashboard/build/main.css", true},
		{"/app/build/chunk-login-abc123.js", false},
		{"/app/build/assets/logo-def456.png", false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			r := gin.New()
			r.Use(spaBuildCacheMiddleware())
			r.GET("/*path", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			cacheControl := w.Header().Get("Cache-Control")
			if tc.wantNoCache {
				assert.Equal(t, "no-cache, must-revalidate", cacheControl)
			} else {
				assert.Equal(t, "public, max-age=31536000, immutable", cacheControl)
			}
		})
	}
}
