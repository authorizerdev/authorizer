//go:build smoke

package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// TestReleaseSmoke249 covers the 2.4.9 fixes black-box, against the real
// binary, on every transport that exposes the affected operation. Each case
// boots its own server(s) because each needs flags the shared release-smoke
// instance does not have (SMTP, a role set that changes across a restart, a
// Redis memory store).
func TestReleaseSmoke249(t *testing.T) {
	bin := buildBinary(t)

	t.Run("GHSA-rpv7-grw6-r2jv unverified login MFA session", func(t *testing.T) {
		smokeUnverifiedLoginMFASession(t, bin)
	})
	t.Run("issue 922 roles removed from --roles", func(t *testing.T) {
		smokeRemovedRoles(t, bin)
	})
	t.Run("issue 924/932 infrastructure errors are masked", func(t *testing.T) {
		smokeMaskedInfraErrors(t, bin)
	})
	t.Run("issue 927 admin users sort and filter parity", func(t *testing.T) {
		smokeAdminUsersSortFilter(t, bin)
	})
}

// --- harness ---------------------------------------------------------------

type smokePorts struct{ http, metrics, grpc int }

func newSmokePorts(t *testing.T) smokePorts {
	return smokePorts{http: freePort(t), metrics: freePort(t), grpc: freePort(t)}
}

func (p smokePorts) baseURL() string { return fmt.Sprintf("http://127.0.0.1:%d", p.http) }

// smokeArgs is the flag set every server in this file shares. Rate limiting is
// off: these tests issue bursts of requests from one IP and must not be
// throttled into a different answer.
func smokeArgs(dbPath string, p smokePorts, extra ...string) []string {
	return append([]string{
		"--database-type=sqlite", "--database-url=" + dbPath,
		"--url=" + p.baseURL(),
		"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret,
		"--admin-secret=" + smokeAdminSecret,
		"--client-id=" + smokeClientID, "--client-secret=" + smokeClientSecret,
		fmt.Sprintf("--http-port=%d", p.http),
		fmt.Sprintf("--metrics-port=%d", p.metrics),
		fmt.Sprintf("--grpc-port=%d", p.grpc),
		"--rate-limit-rps=0",
	}, extra...)
}

func smokeGRPCConn(t *testing.T, p smokePorts) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", p.grpc),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func grpcCtx(t *testing.T, kv ...string) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	if len(kv) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, kv...)
	}
	return ctx
}

// gqlError is one entry of a GraphQL response's errors array.
type gqlError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
}

type gqlResult struct {
	Data    map[string]any
	Errors  []gqlError
	Raw     string
	Cookies []*http.Cookie
}

// gqlRaw posts a GraphQL operation with no cookie jar and returns everything,
// errors included — the negative paths assert on them. cookie, when set, is
// sent verbatim as the Cookie header.
func gqlRaw(t *testing.T, baseURL, query, cookie string) gqlResult {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/graphql", strings.NewReader(string(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var out struct {
		Data   map[string]any `json:"data"`
		Errors []gqlError     `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(raw, &out), "graphql body: %s", raw)
	return gqlResult{Data: out.Data, Errors: out.Errors, Raw: string(raw), Cookies: resp.Cookies()}
}

// field digs data[op][key] out of a GraphQL result as a string ("" if absent).
func (r gqlResult) field(op, key string) string {
	m, _ := r.Data[op].(map[string]any)
	s, _ := m[key].(string)
	return s
}

type restResult struct {
	Status  int
	Body    map[string]any
	Raw     string
	Cookies []*http.Cookie
}

// restRaw POSTs JSON to a /v1 path; cookie, when set, is sent as the Cookie header.
func restRaw(t *testing.T, baseURL, path, body, cookie string) restResult {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return restResult{Status: resp.StatusCode, Body: out, Raw: string(raw), Cookies: resp.Cookies()}
}

func (r restResult) str(key string) string {
	s, _ := r.Body[key].(string)
	return s
}

// mfaSessionCookie returns the "mfa_session=<id>" Cookie header value from a
// response's cookies, or "" when none was set.
func mfaSessionCookie(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "mfa_session" && c.Value != "" {
			return c.Name + "=" + c.Value
		}
	}
	return ""
}

// mfaSessionFromGRPC extracts the MFA session cookie from gRPC response
// header metadata, where the server serialises Set-Cookie.
func mfaSessionFromGRPC(t *testing.T, md metadata.MD) string {
	t.Helper()
	var cookies []*http.Cookie
	for _, line := range md.Get("set-cookie") {
		c, err := http.ParseSetCookie(line)
		require.NoError(t, err)
		cookies = append(cookies, c)
	}
	return mfaSessionCookie(cookies)
}

// --- in-test SMTP sink -----------------------------------------------------

// smtpSink is a minimal plaintext SMTP server that records every delivered
// message by recipient. It advertises no STARTTLS and no AUTH, so the
// server's mailer (opportunistic STARTTLS, no credentials) sends in clear.
type smtpSink struct {
	ln   net.Listener
	mu   sync.Mutex
	msgs map[string][]string
}

func startSMTPSink(t *testing.T) *smtpSink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &smtpSink{ln: ln, msgs: map[string][]string{}}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(c)
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *smtpSink) port() int { return s.ln.Addr().(*net.TCPAddr).Port }

func (s *smtpSink) handle(c net.Conn) {
	defer func() { _ = c.Close() }()
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	r := bufio.NewReader(c)
	reply := func(line string) { _, _ = fmt.Fprintf(c, "%s\r\n", line) }
	reply("220 smoke-sink ESMTP")
	var rcpts []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(line)
		upper := strings.ToUpper(cmd)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			reply("250-smoke-sink")
			reply("250 8BITMIME")
		case strings.HasPrefix(upper, "HELO"):
			reply("250 smoke-sink")
		case strings.HasPrefix(upper, "MAIL FROM"):
			rcpts = nil
			reply("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			addr := cmd[strings.Index(cmd, ":")+1:]
			addr = strings.Trim(strings.TrimSpace(addr), "<>")
			rcpts = append(rcpts, strings.ToLower(addr))
			reply("250 OK")
		case upper == "DATA":
			reply("354 End data with <CR><LF>.<CR><LF>")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" || l == ".\n" {
					break
				}
				b.WriteString(strings.TrimPrefix(l, ".")) // undo dot-stuffing
			}
			body := decodeMailBody(b.String())
			s.mu.Lock()
			for _, rc := range rcpts {
				s.msgs[rc] = append(s.msgs[rc], body)
			}
			s.mu.Unlock()
			reply("250 OK")
		case upper == "RSET":
			rcpts = nil
			reply("250 OK")
		case upper == "NOOP":
			reply("250 OK")
		case upper == "QUIT":
			reply("221 Bye")
			return
		default:
			reply("502 not implemented")
		}
	}
}

// decodeMailBody returns the message body, undoing quoted-printable transfer
// encoding when the message declares it.
func decodeMailBody(raw string) string {
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return raw
	}
	rd := msg.Body
	if strings.EqualFold(msg.Header.Get("Content-Transfer-Encoding"), "quoted-printable") {
		rd = quotedprintable.NewReader(msg.Body)
	}
	b, err := io.ReadAll(rd)
	if err != nil {
		return raw
	}
	return string(b)
}

func (s *smtpSink) count(rcpt string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs[strings.ToLower(rcpt)])
}

// waitMessage blocks until rcpt has received at least n messages and returns
// the n-th. Mail is sent asynchronously after the API response, so this is
// the explicit wait that orders "login responded" before "OTP read".
func (s *smtpSink) waitMessage(t *testing.T, rcpt string, n int) string {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		msgs := s.msgs[strings.ToLower(rcpt)]
		if len(msgs) >= n {
			m := msgs[n-1]
			s.mu.Unlock()
			return m
		}
		s.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no email #%d for %s within 15s", n, rcpt)
	return ""
}

var otpInEmail = regexp.MustCompile(`<b>([A-Z0-9]{6})</b> is your one time password`)

// --- 1. GHSA-rpv7-grw6-r2jv ------------------------------------------------

// smokeUnverifiedLoginMFASession: logging in to an account whose email is not
// verified sends a verification OTP BEFORE the password is checked. Up to
// 2.4.8 the MFA session it armed was a "Verified" one, so a caller with the
// WRONG password could redeem it at skip_mfa_setup for tokens, lock the
// account's MFA, or enroll their own TOTP/email/WebAuthn factor. 2.4.9 arms a
// Challenge session redeemable only by verify_otp with the emailed code.
//
// Phone accounts take the same branch but need a real SMS provider; they are
// covered in-process by TestUnverifiedLoginMFASessionCannotBypassPassword.
func smokeUnverifiedLoginMFASession(t *testing.T, bin string) {
	sink := startSMTPSink(t)
	p := newSmokePorts(t)
	// localhost, not 127.0.0.1: an IP address is not a valid WebAuthn RP ID,
	// so with it webauthn_registration_options fails in BeginRegistration
	// even on a vulnerable build and its assertion would prove nothing.
	baseURL := fmt.Sprintf("http://localhost:%d", p.http)
	// MFA stays at its default (enabled; TOTP, email OTP and WebAuthn need no
	// external provider). Email verification is off so signup yields a plain
	// account, which is then marked unverified via the admin API — signup
	// with verification on leaves a pending verification request that makes
	// login refuse outright and never reach the vulnerable branch.
	startServer(t, bin, smokeArgs(filepath.Join(t.TempDir(), "ghsa.db"), p,
		"--smtp-host=127.0.0.1",
		fmt.Sprintf("--smtp-port=%d", sink.port()),
		"--smtp-sender-email=noreply@smoke.authorizer.dev",
		"--url="+baseURL, // overrides the 127.0.0.1 --url from smokeArgs
	), baseURL)
	client := authorizerv1.NewAuthorizerServiceClient(smokeGRPCConn(t, p))

	const password = "Real-Pass-123!"
	const wrongPassword = "Wrong-Pass-456!"

	admin := newGraphQLClient(t, baseURL)
	admin.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)
	newUnverifiedUser := func(t *testing.T, email string) {
		t.Helper()
		res := gqlRaw(t, baseURL, `mutation { signup(params:{email:"`+email+`", password:"`+password+`", confirm_password:"`+password+`"}) { message } }`, "")
		require.Empty(t, res.Errors, res.Raw)
		// With MFA on, signup withholds the user behind the setup offer, so
		// resolve the id through the admin API.
		u := admin.mutate(t, `query { _user(params:{email:"`+email+`"}) { id } }`)
		id, _ := u["_user"].(map[string]any)["id"].(string)
		require.NotEmpty(t, id)
		admin.mutate(t, `mutation { _update_user(params:{id:"`+id+`", email_verified:false}) { id } }`)
	}

	// One surface's view of the flow.
	type surface struct {
		name string
		// login returns the MFA session Cookie header value, the access token
		// (must be empty) and the message.
		login func(t *testing.T, email, pw string) (cookie, token, msg string)
		// ops are the endpoints that act on a VERIFIED session. Each returns
		// rejected=true when the call failed, plus any token/secret it issued.
		ops       map[string]func(t *testing.T, email, cookie string) (rejected bool, issued string)
		verifyOTP func(t *testing.T, email, otp, cookie string) (token string)
	}

	gqlOp := func(op, query, tokenField string) func(t *testing.T, email, cookie string) (bool, string) {
		return func(t *testing.T, email, cookie string) (bool, string) {
			res := gqlRaw(t, baseURL, strings.ReplaceAll(query, "EMAIL", email), cookie)
			if len(res.Errors) > 0 {
				t.Logf("%s rejected: %s", op, res.Errors[0].Message)
				assert.NotEqual(t, "INTERNAL", res.Errors[0].Extensions["code"], "%s rejected with a server fault, not an auth error: %s", op, res.Raw)
			}
			issued := ""
			if tokenField != "" {
				issued = res.field(op, tokenField)
			}
			return len(res.Errors) > 0, issued
		}
	}
	restOp := func(path, tokenField string) func(t *testing.T, email, cookie string) (bool, string) {
		return func(t *testing.T, email, cookie string) (bool, string) {
			res := restRaw(t, baseURL, path, `{"email":"`+email+`"}`, cookie)
			if res.Status != http.StatusOK {
				t.Logf("%s rejected: %d %s", path, res.Status, res.Raw)
				assert.True(t, res.Status >= 400 && res.Status < 500, "%s: want a 4xx rejection, got %d: %s", path, res.Status, res.Raw)
			}
			issued := ""
			if tokenField != "" {
				issued = res.str(tokenField)
			}
			return res.Status != http.StatusOK, issued
		}
	}
	grpcReject := func(t *testing.T, err error) bool {
		if err != nil {
			t.Logf("rejected: %v", err)
			assert.NotContains(t, []codes.Code{codes.Internal, codes.Unknown}, status.Code(err), "rejected with a server fault: %v", err)
		}
		return err != nil
	}
	cookieCtx := func(t *testing.T, cookie string) context.Context { return grpcCtx(t, "cookie", cookie) }

	surfaces := []surface{
		{
			name: "graphql",
			login: func(t *testing.T, email, pw string) (string, string, string) {
				res := gqlRaw(t, baseURL, `mutation { login(params:{email:"`+email+`", password:"`+pw+`"}) { message access_token } }`, "")
				require.Empty(t, res.Errors, res.Raw)
				return mfaSessionCookie(res.Cookies), res.field("login", "access_token"), res.field("login", "message")
			},
			ops: map[string]func(*testing.T, string, string) (bool, string){
				"skip_mfa_setup":                 gqlOp("skip_mfa_setup", `mutation { skip_mfa_setup(params:{email:"EMAIL"}) { access_token } }`, "access_token"),
				"skip_mfa_setup (no identifier)": gqlOp("skip_mfa_setup", `mutation { skip_mfa_setup(params:{}) { access_token } }`, "access_token"),
				"lock_mfa":                       gqlOp("lock_mfa", `mutation { lock_mfa(params:{email:"EMAIL"}) { message } }`, ""),
				"totp_mfa_setup":                 gqlOp("totp_mfa_setup", `mutation { totp_mfa_setup(params:{email:"EMAIL"}) { access_token authenticator_secret } }`, "authenticator_secret"),
				"email_otp_mfa_setup":            gqlOp("email_otp_mfa_setup", `mutation { email_otp_mfa_setup(params:{email:"EMAIL"}) { message } }`, ""),
				"webauthn_registration_options":  gqlOp("webauthn_registration_options", `mutation { webauthn_registration_options(email:"EMAIL") { options } }`, "options"),
				// Sanity check, not a regression probe: a wrong OTP is refused on vulnerable builds too.
				"verify_otp (wrong otp)": gqlOp("verify_otp", `mutation { verify_otp(params:{email:"EMAIL", otp:"000000"}) { access_token } }`, "access_token"),
			},
			verifyOTP: func(t *testing.T, email, otp, cookie string) string {
				res := gqlRaw(t, baseURL, `mutation { verify_otp(params:{email:"`+email+`", otp:"`+otp+`"}) { access_token } }`, cookie)
				require.Empty(t, res.Errors, res.Raw)
				return res.field("verify_otp", "access_token")
			},
		},
		{
			name: "rest",
			login: func(t *testing.T, email, pw string) (string, string, string) {
				res := restRaw(t, baseURL, "/v1/login", `{"email":"`+email+`","password":"`+pw+`"}`, "")
				require.Equal(t, http.StatusOK, res.Status, res.Raw)
				return mfaSessionCookie(res.Cookies), res.str("access_token"), res.str("message")
			},
			ops: map[string]func(*testing.T, string, string) (bool, string){
				"skip_mfa_setup":                restOp("/v1/skip_mfa_setup", "access_token"),
				"lock_mfa":                      restOp("/v1/lock_mfa", ""),
				"totp_mfa_setup":                restOp("/v1/totp_mfa_setup", "authenticator_secret"),
				"email_otp_mfa_setup":           restOp("/v1/email_otp_mfa_setup", ""),
				"webauthn_registration_options": restOp("/v1/webauthn_registration_options", "options"),
				// Sanity check, not a regression probe: a wrong OTP is refused on vulnerable builds too.
				"verify_otp (wrong otp)": func(t *testing.T, email, cookie string) (bool, string) {
					res := restRaw(t, baseURL, "/v1/verify_otp", `{"email":"`+email+`","otp":"000000"}`, cookie)
					return res.Status != http.StatusOK, res.str("access_token")
				},
			},
			verifyOTP: func(t *testing.T, email, otp, cookie string) string {
				res := restRaw(t, baseURL, "/v1/verify_otp", `{"email":"`+email+`","otp":"`+otp+`"}`, cookie)
				require.Equal(t, http.StatusOK, res.Status, res.Raw)
				return res.str("access_token")
			},
		},
		{
			name: "grpc",
			login: func(t *testing.T, email, pw string) (string, string, string) {
				var md metadata.MD
				res, err := client.Login(grpcCtx(t), &authorizerv1.LoginRequest{Email: email, Password: pw}, grpc.Header(&md))
				require.NoError(t, err)
				return mfaSessionFromGRPC(t, md), res.GetAccessToken(), res.GetMessage()
			},
			ops: map[string]func(*testing.T, string, string) (bool, string){
				"skip_mfa_setup": func(t *testing.T, email, cookie string) (bool, string) {
					res, err := client.SkipMfaSetup(cookieCtx(t, cookie), &authorizerv1.SkipMfaSetupRequest{Email: email})
					return grpcReject(t, err), res.GetAccessToken()
				},
				"lock_mfa": func(t *testing.T, email, cookie string) (bool, string) {
					_, err := client.LockMfa(cookieCtx(t, cookie), &authorizerv1.LockMfaRequest{Email: email})
					return grpcReject(t, err), ""
				},
				"totp_mfa_setup": func(t *testing.T, email, cookie string) (bool, string) {
					res, err := client.TotpMfaSetup(cookieCtx(t, cookie), &authorizerv1.TotpMfaSetupRequest{Email: email})
					return grpcReject(t, err), res.GetAuthenticatorSecret() + res.GetAccessToken()
				},
				"email_otp_mfa_setup": func(t *testing.T, email, cookie string) (bool, string) {
					_, err := client.EmailOtpMfaSetup(cookieCtx(t, cookie), &authorizerv1.EmailOtpMfaSetupRequest{Email: email})
					return grpcReject(t, err), ""
				},
				"webauthn_registration_options": func(t *testing.T, email, cookie string) (bool, string) {
					res, err := client.WebauthnRegistrationOptions(cookieCtx(t, cookie), &authorizerv1.WebauthnRegistrationOptionsRequest{Email: email})
					return grpcReject(t, err), res.GetOptions()
				},
				// Sanity check, not a regression probe: a wrong OTP is refused on vulnerable builds too.
				"verify_otp (wrong otp)": func(t *testing.T, email, cookie string) (bool, string) {
					res, err := client.VerifyOtp(cookieCtx(t, cookie), &authorizerv1.VerifyOtpRequest{Email: email, Otp: "000000"})
					return err != nil, res.GetAccessToken()
				},
			},
			verifyOTP: func(t *testing.T, email, otp, cookie string) string {
				res, err := client.VerifyOtp(cookieCtx(t, cookie), &authorizerv1.VerifyOtpRequest{Email: email, Otp: otp})
				require.NoError(t, err)
				return res.GetAccessToken()
			},
		},
	}

	for _, s := range surfaces {
		t.Run(s.name, func(t *testing.T) {
			// login sends exactly one OTP email per call; waiting for it before
			// the next step keeps every later "latest email" read unambiguous.
			login := func(t *testing.T, email, pw string) string {
				t.Helper()
				n := sink.count(email)
				cookie, token, msg := s.login(t, email, pw)
				require.Equal(t, "Please check email inbox for the OTP", msg, "an unverified login must take the verification-OTP branch")
				require.Empty(t, token, "login must not issue tokens before the OTP")
				require.NotEmpty(t, cookie, "login must arm an MFA session cookie")
				sink.waitMessage(t, email, n+1)
				return cookie
			}

			// Every op gets its own account: on a vulnerable build a bypass
			// changes account state (skipped setup, an enrolled factor, a
			// lock), which would gate the next op for an unrelated reason
			// and hide that it, too, accepted the session.
			var lockedEmail string
			i := 0
			for _, name := range []string{
				"skip_mfa_setup", "skip_mfa_setup (no identifier)", "totp_mfa_setup",
				"email_otp_mfa_setup", "webauthn_registration_options",
				// Sanity check, not a regression probe: a wrong OTP is refused on vulnerable builds too.
				"verify_otp (wrong otp)", "lock_mfa",
			} {
				op, ok := s.ops[name]
				if !ok {
					continue
				}
				i++
				email := fmt.Sprintf("ghsa-rpv7-%s-%d@smoke.authorizer.dev", s.name, i)
				if name == "lock_mfa" {
					lockedEmail = email
				}
				t.Run("wrong password then "+name+" is rejected", func(t *testing.T) {
					newUnverifiedUser(t, email)
					cookie := login(t, email, wrongPassword)
					rejected, issued := op(t, email, cookie)
					assert.True(t, rejected, "%s accepted an MFA session armed by a WRONG password", name)
					assert.Empty(t, issued, "%s issued a credential to a caller with the wrong password", name)
				})
			}

			// The legitimate path still completes — on the account lock_mfa
			// was aimed at, which also proves the rejected call did not lock it.
			t.Run("correct password, emailed OTP, verify_otp issues tokens", func(t *testing.T) {
				require.NotEmpty(t, lockedEmail)
				email := lockedEmail
				n := sink.count(email)
				cookie := login(t, email, password)
				m := otpInEmail.FindStringSubmatch(sink.waitMessage(t, email, n+1))
				require.Len(t, m, 2, "OTP email did not contain a code")
				token := s.verifyOTP(t, email, m[1], cookie)
				require.NotEmpty(t, token, "verify_otp with the emailed OTP must issue an access token")
				u := admin.mutate(t, `query { _user(params:{email:"`+email+`"}) { email_verified } }`)
				assert.Equal(t, true, u["_user"].(map[string]any)["email_verified"], "verifying the emailed OTP must mark the email verified")
			})
		})
	}
}

// --- 2. issue 922: roles removed from --roles --------------------------------

// smokeRemovedRoles: a role dropped from --roles stayed on stored user records
// and kept being granted — at login, on tokens minted before the change, and
// on refresh. Here the change is a real operator action: restart the binary
// on the same database with the role gone.
func smokeRemovedRoles(t *testing.T, bin string) {
	p := newSmokePorts(t)
	baseURL := p.baseURL()
	dbPath := filepath.Join(t.TempDir(), "roles.db")
	const email = "roles-922@smoke.authorizer.dev"
	const password = "Roles-Pass-123!"

	stopA := startServer(t, bin, smokeArgs(dbPath, p, "--disable-mfa", "--roles=user,editor"), baseURL)
	signup := gqlRaw(t, baseURL, `mutation { signup(params:{email:"`+email+`", password:"`+password+`", confirm_password:"`+password+`", roles:["user","editor"]}) { access_token } }`, "")
	require.Empty(t, signup.Errors, signup.Raw)
	old := gqlRaw(t, baseURL, `mutation { login(params:{email:"`+email+`", password:"`+password+`", roles:["user","editor"], scope:["openid","email","profile","offline_access"]}) { access_token refresh_token } }`, "")
	require.Empty(t, old.Errors, old.Raw)
	oldAccess, oldRefresh := old.field("login", "access_token"), old.field("login", "refresh_token")
	require.NotEmpty(t, oldAccess)
	require.NotEmpty(t, oldRefresh)
	// Precondition: editor really was granted, so its absence later is the fix.
	require.Contains(t, jwtPayload(t, oldAccess)["roles"], "editor")
	stopA()

	// Restart: same DB, same secret, same URL (the old tokens' issuer), editor removed.
	startServer(t, bin, smokeArgs(dbPath, p, "--disable-mfa", "--roles=user"), baseURL)
	client := authorizerv1.NewAuthorizerServiceClient(smokeGRPCConn(t, p))

	t.Run("login requesting the removed role fails on every surface", func(t *testing.T) {
		res := gqlRaw(t, baseURL, `mutation { login(params:{email:"`+email+`", password:"`+password+`", roles:["editor"]}) { access_token } }`, "")
		assert.NotEmpty(t, res.Errors, "graphql login granted a removed role: %s", res.Raw)
		assert.Empty(t, res.field("login", "access_token"))

		rest := restRaw(t, baseURL, "/v1/login", `{"email":"`+email+`","password":"`+password+`","roles":["editor"]}`, "")
		assert.NotEqual(t, http.StatusOK, rest.Status, "REST login granted a removed role: %s", rest.Raw)
		assert.Empty(t, rest.str("access_token"))

		g, err := client.Login(grpcCtx(t), &authorizerv1.LoginRequest{Email: email, Password: password, Roles: []string{"editor"}})
		assert.Error(t, err, "gRPC login granted a removed role")
		assert.Empty(t, g.GetAccessToken())
	})

	t.Run("the still-configured role works", func(t *testing.T) {
		res := gqlRaw(t, baseURL, `mutation { login(params:{email:"`+email+`", password:"`+password+`", roles:["user"]}) { access_token } }`, "")
		require.Empty(t, res.Errors, res.Raw)
		claims := jwtPayload(t, res.field("login", "access_token"))
		assert.Equal(t, []any{"user"}, claims["roles"])
		assert.NotContains(t, claims["allowed_roles"], "editor")
	})

	// Before the refresh below: rotating the refresh token retires the
	// session the old access token belongs to.
	t.Run("old access token no longer satisfies the removed role", func(t *testing.T) {
		validate := func(role string) (gqlOK, restOK, grpcOK bool) {
			g := gqlRaw(t, baseURL, `query { validate_jwt_token(params:{token_type:"access_token", token:"`+oldAccess+`", roles:["`+role+`"]}) { is_valid } }`, "")
			v, _ := g.Data["validate_jwt_token"].(map[string]any)
			gqlOK = len(g.Errors) == 0 && v["is_valid"] == true

			r := restRaw(t, baseURL, "/v1/validate_jwt_token", `{"token_type":"access_token","token":"`+oldAccess+`","roles":["`+role+`"]}`, "")
			restOK = r.Status == http.StatusOK && r.Body["is_valid"] == true

			res, err := client.ValidateJwtToken(grpcCtx(t), &authorizerv1.ValidateJwtTokenRequest{TokenType: "access_token", Token: oldAccess, Roles: []string{role}})
			grpcOK = err == nil && res.GetIsValid()
			return
		}
		g, r, gr := validate("editor")
		assert.False(t, g, "graphql validate_jwt_token honoured a removed role")
		assert.False(t, r, "REST validate_jwt_token honoured a removed role")
		assert.False(t, gr, "gRPC validate_jwt_token honoured a removed role")

		g, r, gr = validate("user")
		assert.True(t, g, "graphql validate_jwt_token rejected a still-configured role")
		assert.True(t, r, "REST validate_jwt_token rejected a still-configured role")
		assert.True(t, gr, "gRPC validate_jwt_token rejected a still-configured role")
	})

	t.Run("refresh grant with the old refresh token drops the removed role", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", oldRefresh)
		form.Set("client_id", smokeClientID)
		form.Set("client_secret", smokeClientSecret)
		resp, err := http.Post(baseURL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "refresh failed: %s", raw)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))

		assert.NotContains(t, body["roles"], "editor", "refresh response roles")
		for _, key := range []string{"access_token", "refresh_token"} {
			tok, _ := body[key].(string)
			require.NotEmpty(t, tok, key)
			claims := jwtPayload(t, tok)
			assert.Equal(t, []any{"user"}, claims["roles"], key)
			assert.NotContains(t, claims["allowed_roles"], "editor", key)
		}
	})
}

// --- 3. issues 924/932: infrastructure error masking -------------------------

const fakeRedisMarker = "SMOKE_FAKE_REDIS_DRIVER_TEXT"

// startFakeRedis serves just enough RESP for go-redis to connect and pass the
// startup PING, then answers every data command with an error carrying
// fakeRedisMarker — a deterministic memory-store outage.
func startFakeRedis(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				r := bufio.NewReader(c)
				for {
					args, err := readRESPCommand(r)
					if err != nil {
						return
					}
					var out string
					switch strings.ToUpper(args[0]) {
					case "PING":
						out = "+PONG\r\n"
					case "HELLO":
						out = "-ERR unknown command 'HELLO'\r\n"
					case "CLIENT", "SELECT", "AUTH":
						out = "+OK\r\n"
					default:
						out = "-ERR " + fakeRedisMarker + " simulated backend outage\r\n"
					}
					if _, err := io.WriteString(c, out); err != nil {
						return
					}
				}
			}(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func readRESPCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		f := strings.Fields(line)
		if len(f) == 0 {
			return []string{""}, nil
		}
		return f, nil
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil || n < 1 {
		return nil, fmt.Errorf("bad array header %q", line)
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimRight(hdr, "\r\n")[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

var maskedErr = regexp.MustCompile(`internal server error \(error_id: ([A-Za-z0-9]+)\)`)

// smokeMaskedInfraErrors: a memory-store failure used to reach clients
// verbatim (driver text, Redis responses). 2.4.9 masks it on every transport
// as "internal server error (error_id: …)" and logs the original error under
// the same id.
//
// The memory store is the black-box-inducible backend: the storage layer is
// SQLite here, and SQLite in WAL mode keeps its file descriptors open, so
// deleting/chmod-ing the file mid-run does not fail writes deterministically.
//
// The probe is revoke: it is one of the paths that propagates a memory-store
// fault (login and signup deliberately fail open on the memory store, so they
// succeed during this outage and cannot be used). Signup still mints a
// perfectly valid signed refresh token, which revoke then looks up.
func smokeMaskedInfraErrors(t *testing.T, bin string) {
	redisPort := startFakeRedis(t)
	p := newSmokePorts(t)
	baseURL := p.baseURL()
	_, logPath := startServerWithLog(t, bin, smokeArgs(filepath.Join(t.TempDir(), "mask.db"), p,
		"--disable-mfa", fmt.Sprintf("--redis-url=redis://127.0.0.1:%d", redisPort)), baseURL)
	client := authorizerv1.NewAuthorizerServiceClient(smokeGRPCConn(t, p))

	const password = "Mask-Pass-123!"
	const email = "mask@smoke.authorizer.dev"
	signup := gqlRaw(t, baseURL, `mutation { signup(params:{email:"`+email+`", password:"`+password+`", confirm_password:"`+password+`", scope:["openid","offline_access"]}) { refresh_token } }`, "")
	require.Empty(t, signup.Errors, signup.Raw)
	refresh := signup.field("signup", "refresh_token")
	require.NotEmpty(t, refresh, "signup fails open on the memory store, so it must still mint a refresh token")

	var ids []string
	assertMasked := func(t *testing.T, surface, msg, raw string) {
		t.Helper()
		m := maskedErr.FindStringSubmatch(msg)
		if assert.Len(t, m, 2, "%s: want a masked internal error, got: %s", surface, raw) {
			ids = append(ids, m[1])
		}
		assert.NotContains(t, raw, fakeRedisMarker, "%s leaked the backend error text", surface)
		assert.NotContains(t, strings.ToLower(raw), "redis", "%s leaked the backend name", surface)
	}

	t.Run("graphql", func(t *testing.T) {
		res := gqlRaw(t, baseURL, `mutation { revoke(params:{refresh_token:"`+refresh+`"}) { message } }`, "")
		require.NotEmpty(t, res.Errors, res.Raw)
		assertMasked(t, "graphql", res.Errors[0].Message, res.Raw)
		assert.Equal(t, "INTERNAL", res.Errors[0].Extensions["code"], res.Raw)
	})

	t.Run("rest", func(t *testing.T) {
		res := restRaw(t, baseURL, "/v1/revoke", `{"refresh_token":"`+refresh+`"}`, "")
		assert.Equal(t, http.StatusInternalServerError, res.Status, res.Raw)
		assertMasked(t, "rest", res.str("message"), res.Raw)
	})

	t.Run("grpc", func(t *testing.T) {
		_, err := client.Revoke(grpcCtx(t), &authorizerv1.RevokeRequest{RefreshToken: refresh})
		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err), err)
		assertMasked(t, "grpc", status.Convert(err).Message(), err.Error())
	})

	t.Run("each error_id is logged with the original error", func(t *testing.T) {
		require.Len(t, ids, 3)
		logs, err := os.ReadFile(logPath)
		require.NoError(t, err)
		for _, line := range strings.Split(string(logs), "\n") {
			for i, id := range ids {
				if id != "" && strings.Contains(line, id) && strings.Contains(line, fakeRedisMarker) {
					ids[i] = ""
				}
			}
		}
		for _, id := range ids {
			assert.Empty(t, id, "no server log line carries error_id %s with the original error", id)
		}
	})
}

// --- 4. issue 927: admin users sort and filter -------------------------------

// smokeAdminUsersSortFilter: _users / AuthorizerAdminService.Users gained
// created_at_order and email_verified. The three surfaces must return the
// same users in the same order, and that order must be the real one.
func smokeAdminUsersSortFilter(t *testing.T, bin string) {
	p := newSmokePorts(t)
	baseURL := p.baseURL()
	startServer(t, bin, smokeArgs(filepath.Join(t.TempDir(), "users.db"), p, "--disable-mfa"), baseURL)
	admin := newGraphQLClient(t, baseURL)
	admin.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)
	adminClient := authorizerv1.NewAuthorizerAdminServiceClient(smokeGRPCConn(t, p))

	// Interleave verified and unverified so neither filter is a prefix of the
	// unfiltered order. created_at has one-second resolution: the explicit
	// sleep is what makes the order total.
	const password = "Users-Pass-123!"
	emails := []string{"u1@smoke927.dev", "u2@smoke927.dev", "u3@smoke927.dev", "u4@smoke927.dev"}
	verified := map[string]bool{emails[0]: true, emails[1]: false, emails[2]: true, emails[3]: false}
	for i, e := range emails {
		if i > 0 {
			time.Sleep(1100 * time.Millisecond)
		}
		res := admin.mutate(t, `mutation { signup(params:{email:"`+e+`", password:"`+password+`", confirm_password:"`+password+`"}) { user { id } } }`)
		id := res["signup"].(map[string]any)["user"].(map[string]any)["id"].(string)
		if !verified[e] {
			admin.mutate(t, `mutation { _update_user(params:{id:"`+id+`", email_verified:false}) { id } }`)
		}
	}

	reverse := func(in []string) []string {
		out := make([]string, len(in))
		for i, v := range in {
			out[len(in)-1-i] = v
		}
		return out
	}
	only := func(v bool) []string {
		var out []string
		for _, e := range emails {
			if verified[e] == v {
				out = append(out, e)
			}
		}
		return out
	}

	cases := []struct {
		name     string
		gqlArgs  string
		restBody string
		req      *authorizerv1.UsersRequest
		want     []string
	}{
		{"default is newest first", ``, `{}`, &authorizerv1.UsersRequest{}, reverse(emails)},
		{"ASC", `created_at_order:ASC`, `{"created_at_order":"SORT_ORDER_ASC"}`,
			&authorizerv1.UsersRequest{CreatedAtOrder: authorizerv1.SortOrder_SORT_ORDER_ASC}, emails},
		{"DESC", `created_at_order:DESC`, `{"created_at_order":"SORT_ORDER_DESC"}`,
			&authorizerv1.UsersRequest{CreatedAtOrder: authorizerv1.SortOrder_SORT_ORDER_DESC}, reverse(emails)},
		{"verified ASC", `created_at_order:ASC, email_verified:true`, `{"created_at_order":"SORT_ORDER_ASC","email_verified":true}`,
			&authorizerv1.UsersRequest{CreatedAtOrder: authorizerv1.SortOrder_SORT_ORDER_ASC, EmailVerified: proto.Bool(true)}, only(true)},
		{"unverified DESC", `created_at_order:DESC, email_verified:false`, `{"created_at_order":"SORT_ORDER_DESC","email_verified":false}`,
			&authorizerv1.UsersRequest{CreatedAtOrder: authorizerv1.SortOrder_SORT_ORDER_DESC, EmailVerified: proto.Bool(false)}, reverse(only(false))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			params := ""
			if c.gqlArgs != "" {
				params = "(params:{" + c.gqlArgs + "})"
			}
			data := admin.mutate(t, `query { _users`+params+` { users { email } } }`)
			var gqlEmails []string
			for _, u := range data["_users"].(map[string]any)["users"].([]any) {
				gqlEmails = append(gqlEmails, u.(map[string]any)["email"].(string))
			}

			var out struct {
				Users []struct {
					Email string `json:"email"`
				} `json:"users"`
			}
			code := adminREST(t, baseURL, http.MethodPost, "/v1/admin/users", smokeAdminSecret, c.restBody, &out)
			require.Equal(t, http.StatusOK, code)
			var restEmails []string
			for _, u := range out.Users {
				restEmails = append(restEmails, u.Email)
			}

			res, err := adminClient.Users(grpcCtx(t, "x-authorizer-admin-secret", smokeAdminSecret), c.req)
			require.NoError(t, err)
			var grpcEmails []string
			for _, u := range res.GetUsers() {
				grpcEmails = append(grpcEmails, u.GetEmail())
			}

			assert.Equal(t, c.want, gqlEmails, "graphql")
			assert.Equal(t, c.want, restEmails, "rest")
			assert.Equal(t, c.want, grpcEmails, "grpc")
		})
	}
}
