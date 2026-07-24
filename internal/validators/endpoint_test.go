package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateEndpointURL_AllowPrivateFalse_RejectsPrivate is the production-default
// no-op guard: with allowPrivate=false (Config.TestAllowPrivateWebhookHosts unset,
// the production default) a loopback endpoint is still rejected at registration
// exactly as before the escape hatch existed.
func TestValidateEndpointURL_AllowPrivateFalse_RejectsPrivate(t *testing.T) {
	err := ValidateEndpointURL("http://127.0.0.1:4100/webhook", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

// TestValidateEndpointURL_AllowPrivateTrue_AcceptsPrivate proves the escape hatch
// accepts a private host when explicitly opted into.
func TestValidateEndpointURL_AllowPrivateTrue_AcceptsPrivate(t *testing.T) {
	require.NoError(t, ValidateEndpointURL("http://127.0.0.1:4100/webhook", true))
}

// TestValidateEndpointURL_SchemeAlwaysEnforced proves allowPrivate does NOT relax
// the scheme allow-list — a non-http(s) scheme is rejected either way.
func TestValidateEndpointURL_SchemeAlwaysEnforced(t *testing.T) {
	for _, allowPrivate := range []bool{false, true} {
		err := ValidateEndpointURL("ftp://127.0.0.1/webhook", allowPrivate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheme")
	}
}
