package clientauth

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// newSafeClient carries an escape hatch that lets outbound fetches reach private
// addresses, so the e2e stack can host a JWKS mirror as a compose service. It is
// gated on Config.Env == E2EEnv, which only --env=e2e sets.
//
// That gate is the entire safety argument for the hatch existing, and a gate is
// exactly the kind of thing that survives a refactor in form while losing its
// effect — an inverted condition, a default that drifts, a Config that arrives
// nil. This asserts the gate by OUTCOME (does a private address actually get
// through?) rather than by reading the flag, so any of those changes fail here.
func TestPrivateFetchIsRefusedUnlessEnvIsE2E(t *testing.T) {
	srv := httptest.NewServer(nil) // loopback: private by definition
	defer srv.Close()

	newProvider := func(env string) *provider {
		logger := zerolog.Nop()
		return New(&config.Config{Env: env}, &Dependencies{Log: &logger}).(*provider)
	}

	t.Run("production refuses a private address", func(t *testing.T) {
		_, err := newProvider("production").newSafeClient(context.Background(), srv.URL, time.Second)
		require.Error(t, err, "the escape hatch must not be reachable outside --env=e2e")
		assert.Contains(t, err.Error(), "private/internal networks are not allowed")
	})

	t.Run("an empty env refuses a private address", func(t *testing.T) {
		// The zero value must fail closed: a deployment that sets no --env at all
		// must not inherit the test behaviour.
		_, err := newProvider("").newSafeClient(context.Background(), srv.URL, time.Second)
		require.Error(t, err, "an unset --env must not enable the escape hatch")
	})

	t.Run("e2e allows it", func(t *testing.T) {
		client, err := newProvider(constants.E2EEnv).newSafeClient(context.Background(), srv.URL, time.Second)
		require.NoError(t, err)

		// Constructing the client proves nothing on its own — it does not dial.
		// Reach the server to show the hatch actually works end to end.
		resp, err := client.Get(srv.URL)
		require.NoError(t, err, "under --env=e2e the mirror must be genuinely reachable")
		_ = resp.Body.Close()
	})

	t.Run("a nil Config refuses", func(t *testing.T) {
		p := &provider{}
		_, err := p.newSafeClient(context.Background(), srv.URL, time.Second)
		require.Error(t, err, "a provider with no Config must fail closed, not open")
	})
}
