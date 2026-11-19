package scim

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/parsers"
)

// groupLocation is the one place left in the server that built an absolute URL
// from the request's own Host and X-Forwarded-Proto, so a SCIM client could
// choose the host of the meta.location it got back — on a deployment that had
// pinned --url specifically to stop that. It now goes through
// parsers.GetHostFromRequest, where --url wins over every header.
//
// Without that change the first subtest returns
// "https://evil.example.net/scim/v2/Groups/g-1".
func TestGroupLocation_PinsToTrustedURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctxWith := func(t *testing.T, host string, headers map[string]string) *gin.Context {
		t.Helper()
		r, err := http.NewRequest(http.MethodGet, "http://"+host+"/scim/v2/Groups/g-1", nil)
		require.NoError(t, err)
		r.Host = host
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r
		return c
	}

	t.Run("--url wins over a caller-chosen Host and X-Forwarded-Proto", func(t *testing.T) {
		parsers.SetTrustedURL("https://auth.example.com")
		t.Cleanup(func() { parsers.SetTrustedURL("") })
		require.Equal(t, "https://auth.example.com", parsers.TrustedURL(),
			"precondition: --url must be configured for this case to mean anything")

		got := groupLocation(ctxWith(t, "evil.example.net", map[string]string{
			"X-Forwarded-Proto": "https",
			"X-Forwarded-Host":  "evil.example.net",
		}), "g-1")

		assert.Equal(t, "https://auth.example.com/scim/v2/Groups/g-1", got,
			"a caller must not be able to pick the host of a URL the server returns")
	})

	t.Run("without --url it still derives a usable base from the request", func(t *testing.T) {
		parsers.SetTrustedURL("")
		require.Empty(t, parsers.TrustedURL(), "precondition: --url must be unset here")

		got := groupLocation(ctxWith(t, "auth.internal:8080", nil), "g-2")

		assert.Equal(t, "http://auth.internal:8080/scim/v2/Groups/g-2", got,
			"the un-pinned deployment keeps the documented header-derived behaviour")
	})
}
