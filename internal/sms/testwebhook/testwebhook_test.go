package testwebhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

func TestSendSMS_PostsPhoneAndMessageToWebhook(t *testing.T) {
	var received struct {
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := zerolog.Nop()
	p, err := NewTestWebhookProvider(&config.Config{TestSMSWebhookURL: server.URL}, &Dependencies{Log: &log})
	require.NoError(t, err)

	err = p.SendSMS("+15551234567", "your code is 123456")
	require.NoError(t, err)
	assert.Equal(t, "+15551234567", received.Phone)
	assert.Equal(t, "your code is 123456", received.Message)
}
