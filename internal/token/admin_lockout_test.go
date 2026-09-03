package token

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	inmemorystore "github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// newLockoutProvider builds a token provider backed by a real in-memory store,
// so the failed-attempt counter actually persists between calls. The providers
// in admin_token_test.go deliberately omit the store (exercising the
// fail-open-without-a-store branch); this one needs it.
func newLockoutProvider(t *testing.T, adminSecret string) *provider {
	t.Helper()
	logger := zerolog.Nop()
	cfg := &config.Config{AdminSecret: adminSecret}
	ms, err := inmemorystore.NewInMemoryProvider(cfg, &inmemorystore.Dependencies{Log: &logger})
	require.NoError(t, err)
	return &provider{
		config: cfg,
		dependencies: &Dependencies{
			Log:                 &logger,
			MemoryStoreProvider: ms,
		},
	}
}

// adminReq builds a request from a fixed, untrusted peer carrying whatever
// forwarded header the caller wants to forge.
func adminReq(remoteAddr, xff string) *gin.Context {
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

// TestAdminSecretLockoutSurvivesForgedForwardedHeaders is the regression test
// for GHSA-93hc-xq3w-xw87.
//
// The reported exploit: rotate X-Forwarded-For on every request so each failed
// guess lands in a fresh lockout bucket, turning "10 failures per 15 minutes"
// into an unlimited online guessing budget against the highest-privilege
// credential in the system.
//
// With no --trusted-proxies configured (the default), the forged header must be
// ignored entirely and every attempt must bucket on the connection peer.
func TestAdminSecretLockoutSurvivesForgedForwardedHeaders(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))
	p := newLockoutProvider(t, "correct-horse-battery-staple")

	var lockedAt int
	for i := 1; i <= adminSecretMaxFailedAttempts+5; i++ {
		// A different forged client IP on every single request.
		gc := adminReq("198.51.100.10:40000", fmt.Sprintf("203.0.113.%d", i))
		valid, locked := p.VerifyAdminSecret(utils.GetIP(gc.Request), fmt.Sprintf("wrong-%d", i))
		assert.False(t, valid, "a wrong secret must never validate")
		if locked && lockedAt == 0 {
			lockedAt = i
		}
	}

	require.NotZero(t, lockedAt, "rotating X-Forwarded-For defeated the lockout entirely")
	assert.Equal(t, adminSecretMaxFailedAttempts+1, lockedAt,
		"lockout must fire on the attempt after the budget is spent, regardless of forged headers")
}

// TestAdminSecretLockoutSurvivesNewConnections covers the second, independent
// bypass found while fixing the above: GetIP returned RemoteAddr verbatim,
// which is "IP:port". The ephemeral port changes on every new TCP connection,
// so the bucket key rotated on its own — no header forgery required at all.
func TestAdminSecretLockoutSurvivesNewConnections(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))
	p := newLockoutProvider(t, "correct-horse-battery-staple")

	var lockedAt int
	for i := 1; i <= adminSecretMaxFailedAttempts+5; i++ {
		// Same host, a fresh source port each time — i.e. a new connection.
		gc := adminReq(fmt.Sprintf("198.51.100.11:%d", 40000+i), "")
		_, locked := p.VerifyAdminSecret(utils.GetIP(gc.Request), fmt.Sprintf("wrong-%d", i))
		if locked && lockedAt == 0 {
			lockedAt = i
		}
	}

	require.NotZero(t, lockedAt, "opening a new connection per guess defeated the lockout")
	assert.Equal(t, adminSecretMaxFailedAttempts+1, lockedAt)
}

// TestAdminSecretLockoutIsPerClientBehindTrustedProxy confirms the fix did not
// collapse every client into one bucket when a proxy IS configured: a locked
// client must not lock out a different real client.
func TestAdminSecretLockoutIsPerClientBehindTrustedProxy(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies([]string{"10.0.0.0/8"}))
	t.Cleanup(func() { require.NoError(t, utils.SetTrustedProxies(nil)) })

	p := newLockoutProvider(t, "correct-horse-battery-staple")

	// Spend the budget for one real client arriving through the proxy.
	for i := 1; i <= adminSecretMaxFailedAttempts; i++ {
		gc := adminReq("10.0.0.5:443", "1.1.1.1")
		p.VerifyAdminSecret(utils.GetIP(gc.Request), fmt.Sprintf("wrong-%d", i))
	}
	_, locked := p.VerifyAdminSecret(utils.GetIP(adminReq("10.0.0.5:443", "1.1.1.1").Request), "wrong-again")
	assert.True(t, locked, "the offending client should be locked out")

	// A different real client behind the same proxy is unaffected.
	_, locked = p.VerifyAdminSecret(utils.GetIP(adminReq("10.0.0.5:443", "2.2.2.2").Request), "wrong")
	assert.False(t, locked, "a different client must not inherit another's lockout")

	// And the correct secret still works for the unaffected client.
	valid, locked := p.VerifyAdminSecret(utils.GetIP(adminReq("10.0.0.5:443", "2.2.2.2").Request), "correct-horse-battery-staple")
	assert.True(t, valid)
	assert.False(t, locked)
}
