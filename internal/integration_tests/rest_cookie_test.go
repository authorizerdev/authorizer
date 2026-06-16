package integration_tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestRESTSetCookieOnSignup is the regression test for cookie forwarding over
// the grpc-gateway REST surface. The service layer emits the session cookies as
// `set-cookie` gRPC server metadata (transport.ApplyToGRPC -> grpc.SendHeader);
// without the gateway's WithOutgoingHeaderMatcher promoting them, grpc-gateway
// renames them to `Grpc-Metadata-Set-Cookie` and browsers drop the session.
// This asserts a real `Set-Cookie: <app>_session=...` header comes back.
func TestRESTSetCookieOnSignup(t *testing.T) {
	base := bootRESTGateway(t)
	const hostURL = "http://auth.test.example"

	email := "rest_cookie_" + uuid.New().String() + "@authorizer.dev"
	body := strings.NewReader(`{"email":"` + email + `","password":"Cookie@123","confirm_password":"Cookie@123"}`)
	req, err := http.NewRequest(http.MethodPost, base+"/v1/signup", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authorizer-URL", hostURL)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	setCookies := resp.Header.Values("Set-Cookie")
	require.NotEmpty(t, setCookies,
		"signup must return a real Set-Cookie header over REST (not Grpc-Metadata-Set-Cookie)")

	sessionName := constants.AppCookieName + "_session"
	found := false
	for _, c := range setCookies {
		if strings.HasPrefix(c, sessionName+"=") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a %s cookie; got %v", sessionName, setCookies)

	// The internal metadata name must NOT leak as a header.
	assert.Empty(t, resp.Header.Values("Grpc-Metadata-Set-Cookie"),
		"raw gRPC metadata cookie header must not be exposed over REST")
}
