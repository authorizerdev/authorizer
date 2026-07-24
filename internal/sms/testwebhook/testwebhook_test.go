package testwebhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTestWebhookProvider_WiresFixedSinkURL proves the provider always
// points at the fixed e2e-playground sms-sink address - there is no
// configurable URL to get wrong.
func TestNewTestWebhookProvider_WiresFixedSinkURL(t *testing.T) {
	log := zerolog.Nop()
	p, err := NewTestWebhookProvider(&Dependencies{Log: &log})
	require.NoError(t, err)
	assert.Equal(t, e2eSMSSinkURL, p.webhookURL)
}

// TestSendSMS_PostsPhoneAndMessageToWebhook constructs the unexported struct
// directly (this test lives in the same package) pointed at a local test
// server, since the real constructor no longer accepts a configurable URL.
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
	p := &testWebhookProvider{
		webhookURL: server.URL,
		client:     &http.Client{Timeout: 5 * time.Second},
		log:        &log,
	}

	err := p.SendSMS("+15551234567", "your code is 123456")
	require.NoError(t, err)
	assert.Equal(t, "+15551234567", received.Phone)
	assert.Equal(t, "your code is 123456", received.Message)
}
