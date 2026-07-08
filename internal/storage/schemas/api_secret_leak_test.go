package schemas

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are storage-backend agnostic: they exercise the struct→API model
// conversion that every provider (SQL and NoSQL, including Cassandra/ScyllaDB)
// routes through before returning a record. They guarantee that the secret
// fields tagged json:"-" (User.Password, Client.ClientSecret) — which
// the fix now correctly PERSISTS — can never LEAK back out through any Get/List/
// session-derived API path. model.User / model.Client structurally have
// no field able to carry the secret; these tests fail loudly if that regresses.

const (
	sentinelPassword = "PLAINTEXT-OR-HASH-PASSWORD-SENTINEL"
	sentinelSecret   = "PLAINTEXT-OR-HASH-CLIENT-SECRET-SENTINEL"
)

// TestAsAPIUserNeverLeaksPassword proves User.Password never appears in the JSON
// of the API model, no matter how the record is serialized downstream.
func TestAsAPIUserNeverLeaksPassword(t *testing.T) {
	user := &User{
		ID:            "user-1",
		Email:         refs.NewStringRef("a@b.com"),
		Password:      refs.NewStringRef(sentinelPassword),
		SignupMethods: "basic_auth",
	}

	apiUser := user.AsAPIUser()

	out, err := json.Marshal(apiUser)
	require.NoError(t, err)
	assert.NotContains(t, string(out), sentinelPassword,
		"AsAPIUser output must never contain the password")

	// Marshaling the raw schema struct must also drop it (json:"-" contract that
	// keeps it out of logs/webhooks/error dumps).
	rawOut, err := json.Marshal(user)
	require.NoError(t, err)
	assert.NotContains(t, string(rawOut), sentinelPassword,
		"json.Marshal of schemas.User must never contain the password")
	assert.False(t, strings.Contains(strings.ToLower(string(rawOut)), "password"),
		"schemas.User JSON must not even carry a password key")
}

// TestAsAPIClientNeverLeaksClientSecret proves Client.ClientSecret
// never appears in the JSON of the API model.
func TestAsAPIClientNeverLeaksClientSecret(t *testing.T) {
	sa := &Client{
		ID:            "sa-1",
		Name:          "payments-worker",
		ClientSecret:  sentinelSecret,
		AllowedScopes: "read,write",
		IsActive:      true,
	}

	apiSA := sa.AsAPIClient()

	out, err := json.Marshal(apiSA)
	require.NoError(t, err)
	assert.NotContains(t, string(out), sentinelSecret,
		"AsAPIClient output must never contain the client secret")

	rawOut, err := json.Marshal(sa)
	require.NoError(t, err)
	assert.NotContains(t, string(rawOut), sentinelSecret,
		"json.Marshal of schemas.Client must never contain the client secret")
	assert.False(t, strings.Contains(strings.ToLower(string(rawOut)), "client_secret"),
		"schemas.Client JSON must not even carry a client_secret key")
}
