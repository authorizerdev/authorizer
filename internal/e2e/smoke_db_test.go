//go:build smoke

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReleaseSmokeDB boots the real binary against an external database and
// runs the core user lifecycle through it. Opt-in: it is skipped unless
// SMOKE_DB_ARGS is set, so `make smoke` keeps running on SQLite with no
// Docker. `make smoke-db DB=serenedb|surrealdb|sqlserver` starts the database
// in Docker and sets it (see scripts/smoke-db.sh).
//
// SMOKE_DB_ARGS holds the server's --database-* flags, whitespace-separated.
// It replaces the SQLite flags of the shared smoke flag set.
//
// What the run covers, all on the external database:
//   - signup (behind the default MFA offer), skip_mfa_setup, login, session,
//     the refresh_token grant, the admin users list;
//   - forgot_password twice for one address and resend_verify_email twice
//     for one pending verification, each superseding link still redeemable.
//     The second forgot_password is an upsert of one (email, identifier),
//     which failed with a duplicate key on SQL Server before 2.4.7; with that
//     fix removed, this case fails with INTERNAL on SQL Server;
//   - TOTP and email-OTP MFA enrollment twice, and a concurrent burst of
//     first-time TOTP enrollments for one user. AddAuthenticator had the same
//     SQL Server bug, but the service reads before it writes, so these are
//     liveness checks: they did not reproduce the pre-2.4.7 failure (the
//     upsert itself is pinned by TestAddAuthenticatorConcurrentUpsert in
//     internal/storage/db/sql).
func TestReleaseSmokeDB(t *testing.T) {
	dbArgs := strings.Fields(os.Getenv("SMOKE_DB_ARGS"))
	if len(dbArgs) == 0 {
		t.Skip("SMOKE_DB_ARGS not set; run via `make smoke-db DB=...`")
	}
	bin := buildBinary(t)
	sink := startSMTPSink(t)
	p := newSmokePorts(t)
	baseURL := p.baseURL()

	var args []string
	for _, a := range smokeArgs(filepath.Join(t.TempDir(), "unused.db"), p) {
		if !strings.HasPrefix(a, "--database-") {
			args = append(args, a)
		}
	}
	args = append(args, dbArgs...)
	args = append(args,
		// SereneDB and SQL Server are not OpenFGA datastores; FGA is not under
		// test here, so every backend gets the same in-memory store.
		"--fga-store=memory",
		"--smtp-host=127.0.0.1",
		fmt.Sprintf("--smtp-port=%d", sink.port()),
		"--smtp-sender-email=noreply@smoke.authorizer.dev",
	)
	startServer(t, bin, args, baseURL)

	admin := newGraphQLClient(t, baseURL)
	admin.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)

	// Unique per run: an external database can outlive a failed run.
	run := strings.ReplaceAll(filepath.Base(t.TempDir()), "_", "")
	email := func(name string) string { return name + "-" + run + "@smokedb.dev" }
	const password = "SmokeDB-Pass-123!"

	// signupToken signs a user up and declines the MFA offer (MFA is on by
	// default), returning the access token that was withheld.
	signupToken := func(t *testing.T, addr string) string {
		t.Helper()
		res := gqlRaw(t, baseURL, `mutation { signup(params:{email:"`+addr+`", password:"`+password+`", confirm_password:"`+password+`"}) { message access_token } }`, "")
		require.Empty(t, res.Errors, res.Raw)
		if tok := res.field("signup", "access_token"); tok != "" {
			return tok
		}
		cookie := mfaSessionCookie(res.Cookies)
		require.NotEmpty(t, cookie, "signup neither issued a token nor armed an MFA offer: %s", res.Raw)
		skip := gqlRaw(t, baseURL, `mutation { skip_mfa_setup(params:{email:"`+addr+`"}) { access_token } }`, cookie)
		require.Empty(t, skip.Errors, skip.Raw)
		tok := skip.field("skip_mfa_setup", "access_token")
		require.NotEmpty(t, tok, skip.Raw)
		return tok
	}

	t.Run("signup, login, session, refresh, admin users", func(t *testing.T) {
		addr := email("core")
		require.NotEmpty(t, signupToken(t, addr))

		gql := newGraphQLClient(t, baseURL)
		login := gql.mutate(t, `mutation { login(params:{email:"`+addr+`", password:"`+password+`", scope:["openid","email","offline_access"]}) { access_token refresh_token user { id } } }`)["login"].(map[string]any)
		require.NotEmpty(t, login["access_token"], "a user who declined MFA logs in directly")
		require.NotEmpty(t, login["refresh_token"], "offline_access must yield a refresh_token")
		userID := login["user"].(map[string]any)["id"].(string)

		// session reads the cookie login set and rolls the session over in the
		// store, which retires login's refresh token; the refresh grant below
		// redeems the one session issued.
		sess := gql.mutate(t, `query { session(params:{scope:["openid","email","offline_access"]}) { access_token refresh_token } }`)["session"].(map[string]any)
		assert.NotEmpty(t, sess["access_token"])
		refresh, _ := sess["refresh_token"].(string)
		require.NotEmpty(t, refresh)

		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", refresh)
		form.Set("client_id", smokeClientID)
		status, tokens := oauthToken(t, baseURL, form)
		require.Equal(t, http.StatusOK, status, "%v", tokens)
		assert.NotEmpty(t, tokens["access_token"])
		assert.NotEqual(t, refresh, tokens["refresh_token"], "refresh tokens rotate")

		var out struct {
			Users []struct {
				ID string `json:"id"`
			} `json:"users"`
		}
		require.Equal(t, http.StatusOK, adminREST(t, baseURL, http.MethodPost, "/v1/admin/users", smokeAdminSecret,
			`{"pagination":{"limit":1000}}`, &out))
		var found bool
		for _, u := range out.Users {
			found = found || u.ID == userID
		}
		assert.True(t, found, "the new user is listed via /v1/admin/users")
	})

	t.Run("forgot_password twice: the second link supersedes the first", func(t *testing.T) {
		addr := email("forgot")
		signupToken(t, addr)
		for i := 1; i <= 2; i++ {
			res := gqlRaw(t, baseURL, `mutation { forgot_password(params:{email:"`+addr+`"}) { message } }`, "")
			require.Empty(t, res.Errors, "forgot_password #%d: %s", i, res.Raw)
			sink.waitMessage(t, addr, i)
		}
		tok := emailToken(t, sink.waitMessage(t, addr, 2))
		res := gqlRaw(t, baseURL, `mutation { reset_password(params:{token:"`+tok+`", password:"`+password+`x", confirm_password:"`+password+`x"}) { message } }`, "")
		require.Empty(t, res.Errors, "the second reset link must be redeemable: %s", res.Raw)
		login := gqlRaw(t, baseURL, `mutation { login(params:{email:"`+addr+`", password:"`+password+`x"}) { message } }`, "")
		require.Empty(t, login.Errors, "the reset password must work: %s", login.Raw)
	})

	t.Run("resend_verify_email twice: the latest link verifies", func(t *testing.T) {
		addr := email("resend")
		signupToken(t, addr)
		u := admin.mutate(t, `query { _user(params:{email:"`+addr+`"}) { id } }`)["_user"].(map[string]any)
		admin.mutate(t, `mutation { _update_user(params:{id:"`+u["id"].(string)+`", email_verified:false}) { id } }`)
		for i := 1; i <= 2; i++ {
			res := gqlRaw(t, baseURL, `mutation { resend_verify_email(params:{email:"`+addr+`", identifier:"basic_auth_signup"}) { message } }`, "")
			require.Empty(t, res.Errors, "resend #%d: %s", i, res.Raw)
			sink.waitMessage(t, addr, i)
		}
		tok := emailToken(t, sink.waitMessage(t, addr, 2))
		res := gqlRaw(t, baseURL, `mutation { verify_email(params:{token:"`+tok+`"}) { access_token } }`, "")
		require.Empty(t, res.Errors, "the latest verification link must be redeemable: %s", res.Raw)
		v := admin.mutate(t, `query { _user(params:{email:"`+addr+`"}) { email_verified } }`)["_user"].(map[string]any)
		assert.Equal(t, true, v["email_verified"])
	})

	t.Run("MFA enrollment twice", func(t *testing.T) {
		tok := signupToken(t, email("mfa"))
		for _, op := range []string{
			`mutation { totp_mfa_setup { authenticator_secret } }`,
			`mutation { email_otp_mfa_setup { message } }`,
		} {
			for i := 1; i <= 2; i++ {
				res := gqlBearer(t, baseURL, tok, op)
				require.Empty(t, res.Errors, "enrollment #%d (%s): %s", i, op, res.Raw)
			}
		}
	})

	t.Run("concurrent first-time TOTP enrollment", func(t *testing.T) {
		tok := signupToken(t, email("mfa-race"))
		const n = 8
		var wg sync.WaitGroup
		results := make([]gqlResult, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = gqlBearer(t, baseURL, tok, `mutation { totp_mfa_setup { authenticator_secret } }`)
			}()
		}
		wg.Wait()
		for i, res := range results {
			assert.Empty(t, res.Errors, "concurrent enrollment #%d: %s", i, res.Raw)
		}
	})
}

// gqlBearer posts a GraphQL operation with a bearer token and returns every
// part of the response. Safe to call from several goroutines: it fails the
// test only on transport errors, through assert rather than require.
func gqlBearer(t *testing.T, baseURL, bearer, query string) gqlResult {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"query": query})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/graphql", strings.NewReader(string(body)))
	if !assert.NoError(t, err) {
		return gqlResult{}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	if !assert.NoError(t, err) {
		return gqlResult{}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Data   map[string]any `json:"data"`
		Errors []gqlError     `json:"errors"`
	}
	assert.NoError(t, json.Unmarshal(raw, &out), "graphql body: %s", raw)
	return gqlResult{Data: out.Data, Errors: out.Errors, Raw: string(raw)}
}

var tokenInEmail = regexp.MustCompile(`[?&;]token=([A-Za-z0-9._-]+)`)

// emailToken extracts the verification token from a reset/verification email.
func emailToken(t *testing.T, raw string) string {
	t.Helper()
	m := tokenInEmail.FindStringSubmatch(decodeMailBody(raw))
	require.Len(t, m, 2, "no token link in email:\n%s", raw)
	return m[1]
}
