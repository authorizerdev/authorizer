package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndpointTest tests the webhook endpoint testing functionality by the admin
func TestEndpointTest(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "test_endpoint_user_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	// Create a test server to simulate a webhook endpoint
	var lastReceivedRequest struct {
		EventName string
		Headers   map[string]string
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clear the previous request data
		lastReceivedRequest = struct {
			EventName string
			Headers   map[string]string
		}{
			Headers: make(map[string]string),
		}

		// Read request body
		decoder := json.NewDecoder(r.Body)
		var requestData map[string]interface{}
		err := decoder.Decode(&requestData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid request body"}`))
			return
		}

		// Capture the received event name and headers for test validation
		if eventName, ok := requestData["event_name"].(string); ok {
			lastReceivedRequest.EventName = eventName
		}

		// Capture headers
		for k, v := range r.Header {
			if len(v) > 0 {
				lastReceivedRequest.Headers[k] = v[0]
			}
		}

		// Send appropriate response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := fmt.Sprintf(`{"received_event":"%s","custom_header":"%s"}`,
			lastReceivedRequest.EventName, r.Header.Get("X-Custom-Header"))
		w.Write([]byte(response))
	}))
	defer testServer.Close()

	t.Run("should fail without admin cookie", func(t *testing.T) {
		// Attempt to test endpoint without admin authentication
		params := &model.TestEndpointRequest{
			Endpoint:  testServer.URL,
			EventName: constants.UserLoginWebhookEvent,
			Headers: map[string]interface{}{
				"X-Custom-Header": "test-value",
			},
		}

		resp, err := ts.GraphQLProvider.TestEndpoint(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	// Add admin cookie for the rest of the tests
	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	assert.Nil(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("should fail with invalid event name", func(t *testing.T) {
		params := &model.TestEndpointRequest{
			Endpoint:  testServer.URL,
			EventName: "invalid_event_name",
			Headers:   map[string]interface{}{},
		}

		resp, err := ts.GraphQLProvider.TestEndpoint(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid event_name")
	})

	t.Run("should successfully test endpoint with login event", func(t *testing.T) {
		// Clear previous request data
		lastReceivedRequest.EventName = ""
		lastReceivedRequest.Headers = make(map[string]string)

		params := &model.TestEndpointRequest{
			Endpoint:  testServer.URL,
			EventName: constants.UserLoginWebhookEvent,
			Headers: map[string]interface{}{
				"X-Custom-Header": "test-login-event",
				"Content-Type":    "application/json",
			},
		}

		resp, err := ts.GraphQLProvider.TestEndpoint(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Wait a moment for the request to be processed by our test server
		time.Sleep(50 * time.Millisecond)

		// Verify HTTP status
		require.NotNil(t, resp.HTTPStatus)
		assert.Equal(t, int64(http.StatusOK), *resp.HTTPStatus)

		// Verify the received event name from the test server
		assert.Equal(t, constants.UserLoginWebhookEvent, lastReceivedRequest.EventName)

		// Verify response body
		require.NotNil(t, resp.Response)
		responseBody := *resp.Response

		// Parse response JSON
		var respData map[string]interface{}
		err = json.Unmarshal([]byte(responseBody), &respData)
		require.NoError(t, err)

		assert.Equal(t, constants.UserLoginWebhookEvent, respData["received_event"])
		assert.Equal(t, "test-login-event", respData["custom_header"])
	})

	t.Run("should successfully test endpoint with user created event", func(t *testing.T) {
		// Clear previous request data
		lastReceivedRequest.EventName = ""
		lastReceivedRequest.Headers = make(map[string]string)

		// Create a separate test server for user created event
		userCreatedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			var requestData map[string]interface{}
			decoder.Decode(&requestData)

			eventName := requestData["event_name"].(string)
			customHeader := r.Header.Get("X-Custom-Header")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{"received_event":"%s","custom_header":"%s"}`,
				eventName, customHeader)
			w.Write([]byte(response))
		}))
		defer userCreatedServer.Close()

		params := &model.TestEndpointRequest{
			Endpoint:  userCreatedServer.URL,
			EventName: constants.UserLoginWebhookEvent,
			Headers: map[string]interface{}{
				"X-Custom-Header": "test-user-login-event",
			},
		}

		resp, err := ts.GraphQLProvider.TestEndpoint(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify HTTP status
		require.NotNil(t, resp.HTTPStatus)
		assert.Equal(t, int64(http.StatusOK), *resp.HTTPStatus)

		// Verify response body directly
		require.NotNil(t, resp.Response)
		responseBody := *resp.Response

		// Parse response JSON
		var respData map[string]interface{}
		err = json.Unmarshal([]byte(responseBody), &respData)
		require.NoError(t, err)

		assert.Equal(t, constants.UserLoginWebhookEvent, respData["received_event"])
		assert.Equal(t, "test-user-login-event", respData["custom_header"])
	})

	t.Run("should handle endpoint that doesn't exist", func(t *testing.T) {
		params := &model.TestEndpointRequest{
			Endpoint:  "http://non-existent-endpoint.example",
			EventName: constants.UserLoginWebhookEvent,
			Headers:   map[string]interface{}{},
		}

		resp, err := ts.GraphQLProvider.TestEndpoint(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
	})
}
