package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// eventRecorder is a webhook sink that records the event names delivered to it.
type eventRecorder struct {
	mu     sync.Mutex
	events []string
	srv    *httptest.Server
}

func newEventRecorder(t *testing.T) *eventRecorder {
	t.Helper()
	r := &eventRecorder{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var payload struct {
			EventName string `json:"event_name"`
		}
		if err := json.Unmarshal(body, &payload); err == nil && payload.EventName != "" {
			r.mu.Lock()
			r.events = append(r.events, payload.EventName)
			r.mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(r.srv.Close)
	return r
}

// countOf waits briefly for asynchronous delivery, then counts an event name.
// Webhooks are dispatched via asyncutil.Go, so a bare read races the handler.
func (r *eventRecorder) countOf(name string) int {
	deadline := time.Now().Add(3 * time.Second)
	for {
		r.mu.Lock()
		n := 0
		for _, e := range r.events {
			if e == name {
				n++
			}
		}
		r.mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			return n
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestSignupEmitsEventsWhenTheAccountIsCreated is the regression test for
// signup webhooks going missing entirely.
//
// Every path after AddUser can return early: the email-verification branch,
// the phone branch, and the MFA gate. Since 2.4.0 MFA is ON BY DEFAULT, so
// that gate fires for ordinary signups and the emissions that used to sit at
// the bottom of SignUp became unreachable — a default install delivered NO
// signup webhook at all, and with email verification on it delivered only
// user.created. Integrations that provision on signup (CRM records, welcome
// mail, seat accounting) silently never ran.
//
// user.created and user.signup now fire when the account row is written, so
// they mean exactly "the account exists". user.login stays at token issuance,
// because a user who abandons MFA setup has signed up but not logged in.
func TestSignupEmitsEventsWhenTheAccountIsCreated(t *testing.T) {
	const password = "Password@123"

	// registerHooks arms webhooks for the three signup-journey events. Webhook
	// registration is admin-only, so the admin cookie is set on the request
	// first (same pattern as add_email_template_test.go).
	registerHooks := func(t *testing.T, ts *testSetup, ctx context.Context, endpoint string) {
		t.Helper()
		h, err := crypto.EncryptPassword(ts.Config.AdminSecret)
		require.NoError(t, err)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		for _, ev := range []string{
			constants.UserCreatedWebhookEvent,
			constants.UserSignUpWebhookEvent,
			constants.UserLoginWebhookEvent,
		} {
			desc := "test hook for " + ev
			_, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
				EventName:        ev,
				Endpoint:         endpoint,
				Enabled:          true,
				EventDescription: &desc,
			})
			require.NoError(t, err, "failed to register webhook for %s", ev)
		}
		// Drop the admin cookie so the signup under test runs unauthenticated.
		ts.GinContext.Request.Header.Del("Cookie")
	}

	t.Run("MFA on: signup still emits user.created and user.signup", func(t *testing.T) {
		rec := newEventRecorder(t)
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		// Webhook DELIVERY is SSRF-hardened and refuses private/loopback
		// targets unless Env is e2e — the same switch the e2e-playground uses
		// for its own webhook sink. httptest binds to 127.0.0.1, so without
		// this the events fire but never reach the recorder.
		cfg.Env = constants.E2EEnv
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		registerHooks(t, ts, ctx, rec.srv.URL)

		email := "signup_events_" + uuid.NewString() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.Nil(t, res.AccessToken,
			"precondition: MFA is on, so the token is withheld and the old emission point is unreachable")

		assert.Equal(t, 1, rec.countOf(constants.UserSignUpWebhookEvent),
			"user.signup must fire when the account is created — it previously never fired at all "+
				"on a default install, because the MFA gate returns before the old emission point")
		assert.Equal(t, 1, rec.countOf(constants.UserCreatedWebhookEvent),
			"user.created must fire when the account is created")
	})

	t.Run("user.login is not emitted while the token is still withheld", func(t *testing.T) {
		rec := newEventRecorder(t)
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		// Webhook DELIVERY is SSRF-hardened and refuses private/loopback
		// targets unless Env is e2e — the same switch the e2e-playground uses
		// for its own webhook sink. httptest binds to 127.0.0.1, so without
		// this the events fire but never reach the recorder.
		cfg.Env = constants.E2EEnv
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		registerHooks(t, ts, ctx, rec.srv.URL)

		email := "signup_nologin_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		// Wait for signup's own events so the sink has settled before asserting
		// the absence of a login event.
		require.Equal(t, 1, rec.countOf(constants.UserSignUpWebhookEvent))

		rec.mu.Lock()
		loginCount := 0
		for _, e := range rec.events {
			if e == constants.UserLoginWebhookEvent {
				loginCount++
			}
		}
		rec.mu.Unlock()
		assert.Zero(t, loginCount,
			"a user mid-MFA-setup has signed up but not logged in; user.login belongs at token issuance")
	})

	t.Run("the full journey emits each event exactly once", func(t *testing.T) {
		rec := newEventRecorder(t)
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		// Webhook DELIVERY is SSRF-hardened and refuses private/loopback
		// targets unless Env is e2e — the same switch the e2e-playground uses
		// for its own webhook sink. httptest binds to 127.0.0.1, so without
		// this the events fire but never reach the recorder.
		cfg.Env = constants.E2EEnv
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		registerHooks(t, ts, ctx, rec.srv.URL)

		email := "signup_journey_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		ts.GinContext.Request.Header.Set("Cookie",
			fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))
		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, skipRes.AccessToken)

		assert.Equal(t, 1, rec.countOf(constants.UserCreatedWebhookEvent), "user.created exactly once")
		assert.Equal(t, 1, rec.countOf(constants.UserSignUpWebhookEvent),
			"user.signup exactly once — moving the emission must not double-fire it")
		assert.Equal(t, 1, rec.countOf(constants.UserLoginWebhookEvent),
			"user.login exactly once, at token issuance")
	})
}
