package sdktests

// Shared test harness for the SDK-driven e2e-playground suite.
//
// Two kinds of client appear throughout:
//
//   - *authorizer.AuthorizerClient / *authorizer.AuthorizerAdminClient — the
//     real published SDK. Every call that CAN go through a typed SDK method
//     does, so the suite catches wire-shape drift between what the SDK
//     sends/parses and what the server actually does.
//
//   - a raw *http.Client carrying a cookie jar (jarClient/rawGraphQL) — used
//     ONLY for the two things the SDK genuinely cannot do in v2.2.0-rc.4:
//     (1) capture the `mfa_session` Set-Cookie a login/OTP-setup response
//     arms (the SDK's Login discards response headers), and (2) send that
//     cookie on `verify_otp` (the SDK's VerifyOTP takes no per-call headers).
//     Both are labelled at every call site as SDK gaps, not preferences. See
//     README.md "Confirmed SDK gaps".

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer-go/v2"
	"github.com/pquerna/otp/totp"
)

// testPassword is a fixed strong password reused across signups.
const testPassword = "Str0ngPassw0rd!"

// Target instances. Defaults are the host-published ports from
// e2e-playground/docker-compose.yml; the go-sdk-tests compose service overrides
// them with the docker-internal hostnames (so origins / the WebAuthn RPID line
// up exactly, same as the playwright service).
var (
	baseURL         = envOr("AUTHORIZER_BASE_URL", "http://localhost:8080")
	mfaEnforcedURL  = envOr("AUTHORIZER_MFA_ENFORCED_BASE_URL", "http://localhost:8084")
	mfaMagicLinkURL = envOr("AUTHORIZER_MFA_MAGIC_LINK_BASE_URL", "http://localhost:8085")
	webauthnURL     = envOr("AUTHORIZER_WEBAUTHN_BASE_URL", "http://localhost:8082")
	adminSecret     = envOr("AUTHORIZER_ADMIN_SECRET", "e2e-admin-secret")
	clientID        = envOr("AUTHORIZER_CLIENT_ID", "e2e-client-id")
	clientSecret    = envOr("AUTHORIZER_CLIENT_SECRET", "e2e-client-secret")
	smsSinkURL      = envOr("SMS_SINK_BASE_URL", "http://localhost:4100")
	webhookSinkURL  = envOr("WEBHOOK_SINK_BASE_URL", "http://localhost:4200")
	mailpitURL      = envOr("MAILPIT_BASE_URL", "http://localhost:8025")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// userClient builds an SDK user client for the given instance. Origin is set to
// the instance's own URL: the server's CSRF guard rejects state-changing
// requests with no Origin/Referer, and the SDK already auto-injects the same
// value, so this only makes the intent explicit.
func userClient(t *testing.T, instanceURL string) *authorizer.AuthorizerClient {
	t.Helper()
	c, err := authorizer.NewAuthorizerClient(clientID, instanceURL, "", map[string]string{"Origin": instanceURL})
	if err != nil {
		t.Fatalf("NewAuthorizerClient: %v", err)
	}
	return c
}

// adminClient builds an SDK admin client (graphql protocol) for the given
// instance.
func adminClient(t *testing.T, instanceURL string) *authorizer.AuthorizerAdminClient {
	t.Helper()
	c, err := authorizer.NewAuthorizerAdminClient(instanceURL, adminSecret,
		authorizer.WithAdminExtraHeaders(map[string]string{"Origin": instanceURL}))
	if err != nil {
		t.Fatalf("NewAuthorizerAdminClient: %v", err)
	}
	return c
}

func randomEmail(prefix string) string {
	return fmt.Sprintf("%s-%d-%d@example.com", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
}

func randomPhone() string {
	return fmt.Sprintf("+1555%07d", rand.Intn(9_000_000)+1_000_000)
}

// randomSlug returns a unique URL-safe organization name.
func randomSlug(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
}

// --- raw cookie-jar GraphQL (the labelled SDK-gap escape hatch) -------------

// jarClient returns an http.Client with an isolated cookie jar, so a login's
// mfa_session cookie is retained and replayed on the follow-up verify_otp call.
func jarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	return &http.Client{Jar: jar, Timeout: 30 * time.Second}
}

type gqlError struct {
	Message string `json:"message"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors"`
}

func (r gqlResponse) errorText() string {
	msgs := make([]string, 0, len(r.Errors))
	for _, e := range r.Errors {
		msgs = append(msgs, e.Message)
	}
	b, _ := json.Marshal(msgs)
	return string(b)
}

// rawGraphQL POSTs a GraphQL operation through the given cookie-jar client. Used
// only for verify_otp (SDK VerifyOTP takes no cookie header) and for login when
// the mfa_session Set-Cookie must be captured (SDK Login discards it).
func rawGraphQL(t *testing.T, hc *http.Client, instanceURL, query string, variables map[string]any) gqlResponse {
	t.Helper()
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		t.Fatalf("marshal gql: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, instanceURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", instanceURL)
	res, err := hc.Do(req)
	if err != nil {
		t.Fatalf("gql POST: %v", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var parsed gqlResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode gql response (status %d): %v\nbody: %s", res.StatusCode, err, raw)
	}
	return parsed
}

// mfaSessionCookieHeader serialises whatever cookies the jar holds for the
// instance into a single Cookie header value, so it can be threaded into an SDK
// method that DOES accept per-call headers (TotpMfaSetup, SmsOtpMfaSetup,
// WebauthnRegistration*). This is the genuine-SDK half of the hybrid flow.
func mfaSessionCookieHeader(t *testing.T, hc *http.Client, instanceURL string) string {
	t.Helper()
	u, err := url.Parse(instanceURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	var parts []byte
	for i, c := range hc.Jar.Cookies(u) {
		if i > 0 {
			parts = append(parts, ';', ' ')
		}
		parts = append(parts, []byte(c.Name+"="+c.Value)...)
	}
	if len(parts) == 0 {
		t.Fatal("no cookies captured from login (mfa_session expected)")
	}
	return string(parts)
}

// loginCapture drives the login mutation through the raw jar client so the
// mfa_session cookie is retained, returning the parsed login payload for
// assertions. (SDK Login is asserted separately in mfa_routing_test.go; here we
// need the cookie the SDK would throw away.)
type loginPayload struct {
	Message                   string  `json:"message"`
	AccessToken               *string `json:"access_token"`
	ShouldShowTotpScreen      *bool   `json:"should_show_totp_screen"`
	ShouldOfferSmsOtpMfaSetup *bool   `json:"should_offer_sms_otp_mfa_setup"`
}

func loginCapture(t *testing.T, hc *http.Client, instanceURL, email string) loginPayload {
	t.Helper()
	const q = `mutation ($params: LoginRequest!) {
		login(params: $params) { message access_token should_show_totp_screen should_offer_sms_otp_mfa_setup }
	}`
	res := rawGraphQL(t, hc, instanceURL, q, map[string]any{
		"params": map[string]any{"email": email, "password": testPassword},
	})
	if len(res.Errors) > 0 {
		t.Fatalf("login errored: %s", res.errorText())
	}
	var wrap struct {
		Login loginPayload `json:"login"`
	}
	if err := json.Unmarshal(res.Data, &wrap); err != nil {
		t.Fatalf("decode login payload: %v", err)
	}
	return wrap.Login
}

// --- mock-sink polling ------------------------------------------------------

// pollSMS polls sms-sink's GET /sms/:phone (404 until a message lands) for the
// OTP body sent to phone, mirroring the Playwright suite's helper.
func pollSMS(t *testing.T, phone string) string {
	t.Helper()
	for i := 0; i < 40; i++ {
		res, err := http.Get(smsSinkURL + "/sms/" + url.PathEscape(phone))
		if err == nil && res.StatusCode == http.StatusOK {
			var body struct {
				Message string `json:"message"`
			}
			_ = json.NewDecoder(res.Body).Decode(&body)
			res.Body.Close()
			if body.Message != "" {
				return body.Message
			}
		}
		if res != nil {
			res.Body.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("no SMS received for %s within timeout", phone)
	return ""
}

// extractOTP pulls the 6-char code out of "...code is: XXXXXX". The code charset
// is A-Z0-9 minus ambiguous I/O/0/1 (utils.GenerateOTP), so a numeric-only regex
// would miss it — anchor on the fixed prefix and take the first whitespace token.
func extractOTP(t *testing.T, message string) string {
	t.Helper()
	const marker = "code is:"
	idx := strings.Index(message, marker)
	if idx < 0 {
		t.Fatalf("no OTP marker in SMS body: %q", message)
	}
	fields := strings.Fields(message[idx+len(marker):])
	if len(fields) == 0 || len(fields[0]) != 6 {
		t.Fatalf("could not extract 6-char OTP from SMS body: %q", message)
	}
	return fields[0]
}

// --- TOTP code generation (server uses github.com/pquerna/otp/totp) ---------

func totpCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("totp.GenerateCode: %v", err)
	}
	return code
}

// wrongTotpCode returns a 6-digit code guaranteed different from the current
// valid one (increment mod 1e6), so it always fails verification.
func wrongTotpCode(t *testing.T, secret string) string {
	t.Helper()
	valid := totpCode(t, secret)
	var n int
	fmt.Sscanf(valid, "%d", &n)
	return fmt.Sprintf("%06d", (n+1)%1_000_000)
}

func boolValue(b *bool) bool { return b != nil && *b }

// --- small assertion / raw-verify helpers used across the OTP suites --------

// httpJar bundles a *testing.T with a cookie-jar client so the OTP tests can
// call verify_otp (the one call the SDK can't carry a cookie into) tersely.
type httpJar struct {
	t  *testing.T
	hc *http.Client
}

// verifyOTP runs the verify_otp mutation through the jar client against the
// default instance. SDK-gap escape hatch, labelled at every call site's helper.
func (j *httpJar) verifyOTP(params map[string]any) gqlResponse {
	j.t.Helper()
	return rawGraphQL(j.t, j.hc, baseURL, verifyOTPMutation, map[string]any{"params": params})
}

func mustDecode(t *testing.T, res gqlResponse, out any) {
	t.Helper()
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected gql errors: %s", res.errorText())
	}
	if err := json.Unmarshal(res.Data, out); err != nil {
		t.Fatalf("decode gql data: %v", err)
	}
}

func assertErrorContains(t *testing.T, res gqlResponse, want string) {
	t.Helper()
	if len(res.Errors) == 0 {
		t.Fatalf("expected an error containing %q, got none (data: %s)", want, res.Data)
	}
	if !strings.Contains(strings.ToLower(res.errorText()), strings.ToLower(want)) {
		t.Fatalf("expected error containing %q, got %s", want, res.errorText())
	}
}
