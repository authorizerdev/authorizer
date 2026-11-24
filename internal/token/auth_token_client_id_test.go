package token

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

func parseUnverified(t *testing.T, tok string) jwt.MapClaims {
	t.Helper()
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tok, claims)
	require.NoError(t, err)
	return claims
}

// A machine token's client_id must never fall back to the deployment's
// client_id: that would label a service account as the first-party app.
func TestMachineAccessTokenClientID(t *testing.T) {
	p := &provider{config: &config.Config{
		ClientID: "deployment-client", JWTType: "HS256", JWTSecret: "test-secret",
	}}

	cases := []struct {
		name, clientID, resource, wantClientID, wantAud string
	}{
		{"public client_id", "svc-public", "", "svc-public", "deployment-client"},
		{"public client_id with resource", "svc-public", "https://api.example.com", "svc-public", "https://api.example.com"},
		{"no client_id falls back to the account id", "", "", "sa-row-id", "deployment-client"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := p.CreateMachineAuthToken(&AuthTokenConfig{
				ServiceAccountID: "sa-row-id",
				ClientID:         tc.clientID,
				Resource:         tc.resource,
				HostName:         "http://localhost:8080",
			})
			require.NoError(t, err)
			claims := parseUnverified(t, tok.Token)
			assert.Equal(t, tc.wantClientID, claims["client_id"])
			assert.Equal(t, tc.wantAud, claims["aud"])
			assert.Equal(t, "sa-row-id", claims["sub"])
		})
	}
}
