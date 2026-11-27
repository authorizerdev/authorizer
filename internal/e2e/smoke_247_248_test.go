//go:build smoke

package e2e

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// TestReleaseSmoke247248 covers the user-visible 2.4.7 and 2.4.8 changes
// black-box, against the real binary. The storage backends those releases
// added or fixed (SereneDB, SurrealDB, SQL Server) need Docker and run in the
// opt-in TestReleaseSmokeDB (`make smoke-db DB=...`), not here.
func TestReleaseSmoke247248(t *testing.T) {
	bin := buildBinary(t)

	t.Run("GHSA-p7g3-2258-8wmp SCIM cannot rewrite a shared user's identifiers", func(t *testing.T) {
		smokeSCIMSharedIdentifier(t, bin)
	})
	t.Run("issue 904 kid header names a published JWKS key", func(t *testing.T) {
		smokeJWTKid(t, bin)
	})
	t.Run("issue 913 client_id and azp claims", func(t *testing.T) {
		smokeClientIDClaims(t, bin)
	})
	t.Run("issue 903 list_permissions reports truncation", func(t *testing.T) {
		smokeListPermissionsTruncated(t, bin)
	})
}

// --- shared helpers --------------------------------------------------------

// withoutJWTFlags drops every --jwt-* flag from a smokeArgs set so a test can
// supply its own signing configuration.
func withoutJWTFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if !strings.HasPrefix(a, "--jwt-") {
			out = append(out, a)
		}
	}
	return out
}

// jwtHeader decodes a JWT's protected header without verifying it.
func jwtHeader(t *testing.T, tok string) map[string]any {
	t.Helper()
	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3, "not a JWT")
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var h map[string]any
	require.NoError(t, json.Unmarshal(raw, &h))
	return h
}

// fetchJWKS reads /.well-known/jwks.json the way a resource server does.
func fetchJWKS(t *testing.T, baseURL string) jose.JSONWebKeySet {
	t.Helper()
	resp, err := http.Get(baseURL + "/.well-known/jwks.json")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "jwks: %s", raw)
	var set jose.JSONWebKeySet
	require.NoError(t, json.Unmarshal(raw, &set), "jwks: %s", raw)
	return set
}

// oauthToken posts a form to /oauth/token and returns status and decoded body.
func oauthToken(t *testing.T, baseURL string, form url.Values) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(baseURL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	out := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &out), "token endpoint body: %s", raw)
	return resp.StatusCode, out
}

// authorizeCode runs /authorize for the session carried by gql's cookie jar
// and returns the authorization code it redirects with.
func authorizeCode(t *testing.T, gql *graphQLClient, baseURL, scope, challenge string) string {
	t.Helper()
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", smokeClientID)
	q.Set("redirect_uri", baseURL+"/smoke-callback")
	q.Set("scope", scope)
	q.Set("state", "smoke-state")
	q.Set("response_mode", "query")
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	noRedirect := &http.Client{
		Jar:           gql.client.Jar,
		Timeout:       15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := noRedirect.Get(baseURL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusFound, resp.StatusCode, "authorize must redirect with a code: %s", body)
	loc, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code, "authorize must mint a code: %s", resp.Header.Get("Location"))
	return code
}

// --- a. GHSA-p7g3-2258-8wmp ------------------------------------------------

// smokeSCIMSharedIdentifier: a SCIM token is scoped to one organization, but
// up to 2.4.7 PATCH /scim/v2/Users/{id} wrote email/phone to the GLOBAL user
// row. Org A could point a user it shares with org B at an address A
// controls, request a password reset there, and sign in as that user in B.
// 2.4.8 refuses the change with 400 / scimType "mutability" and writes
// nothing; single-org users and case-only email changes still sync.
func smokeSCIMSharedIdentifier(t *testing.T, bin string) {
	sink := startSMTPSink(t)
	p := newSmokePorts(t)
	baseURL := p.baseURL()
	startServer(t, bin, smokeArgs(filepath.Join(t.TempDir(), "scim.db"), p, "--disable-mfa",
		"--smtp-host=127.0.0.1",
		fmt.Sprintf("--smtp-port=%d", sink.port()),
		"--smtp-sender-email=noreply@smoke.authorizer.dev",
	), baseURL)
	admin := newGraphQLClient(t, baseURL)
	admin.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)

	newOrg := func(name string) (string, string) {
		id := createSmokeOrg(t, admin, name)
		ep := admin.mutate(t, `mutation { _create_scim_endpoint(params:{org_id:"`+id+`"}) { token } }`)
		tok := ep["_create_scim_endpoint"].(map[string]any)["token"].(string)
		require.NotEmpty(t, tok)
		return id, tok
	}
	_, tokenA := newOrg("ghsa-p7g3-org-a")
	orgB, tokenB := newOrg("ghsa-p7g3-org-b")

	scim := func(t *testing.T, method, path, bearer, payload string) (int, map[string]any, string) {
		t.Helper()
		var rdr io.Reader
		if payload != "" {
			rdr = strings.NewReader(payload)
		}
		req, err := http.NewRequest(method, baseURL+path, rdr)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/scim+json")
		req.Header.Set("Authorization", "Bearer "+bearer)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		out := map[string]any{}
		_ = json.Unmarshal(raw, &out)
		return resp.StatusCode, out, string(raw)
	}
	provision := func(t *testing.T, bearer, email string) string {
		t.Helper()
		status, body, raw := scim(t, http.MethodPost, "/scim/v2/Users", bearer,
			`{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"`+email+`","active":true,`+
				`"emails":[{"value":"`+email+`","primary":true}]}`)
		require.Equal(t, http.StatusCreated, status, "SCIM create: %s", raw)
		id, _ := body["id"].(string)
		require.NotEmpty(t, id)
		return id
	}
	patchOp := func(ops string) string {
		return `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[` + ops + `]}`
	}
	// The admin API's view of the global user row: what org B, the login form
	// and password recovery all read.
	globalUser := func(t *testing.T, id string) (email, phone string) {
		t.Helper()
		u := admin.mutate(t, `query { _user(params:{id:"`+id+`"}) { email phone_number } }`)["_user"].(map[string]any)
		email, _ = u["email"].(string)
		phone, _ = u["phone_number"].(string)
		return email, phone
	}

	const sharedEmail = "shared-ghsa@smoke248.dev"
	sharedID := provision(t, tokenA, sharedEmail)
	admin.mutate(t, `mutation { _add_org_member(params:{org_id:"`+orgB+`", user_id:"`+sharedID+`"}) { id } }`)

	const attacker = "attacker-ghsa@evil248.dev"
	refused := map[string]string{
		"userName":     `{"op":"replace","path":"userName","value":"` + attacker + `"}`,
		"emails":       `{"op":"replace","path":"emails","value":[{"value":"` + attacker + `","primary":true}]}`,
		"no-path":      `{"op":"replace","value":{"userName":"` + attacker + `"}}`,
		"phoneNumbers": `{"op":"replace","path":"phoneNumbers","value":[{"value":"+15550001111"}]}`,
		// The refusal must take the whole PatchOp down, not just the identifier
		// op: the harmless rename riding along is not written either.
		"mixed with a name change": `{"op":"replace","path":"name.givenName","value":"Mallory"},` +
			`{"op":"replace","path":"userName","value":"` + attacker + `"}`,
	}
	for name, ops := range refused {
		t.Run("org A PATCH "+name+" on shared user -> 400 mutability", func(t *testing.T) {
			status, body, raw := scim(t, http.MethodPatch, "/scim/v2/Users/"+sharedID, tokenA, patchOp(ops))
			require.Equal(t, http.StatusBadRequest, status, raw)
			assert.Equal(t, "mutability", body["scimType"], raw)

			email, phone := globalUser(t, sharedID)
			assert.Equal(t, sharedEmail, email, "the refused PATCH must not have written the email")
			assert.Empty(t, phone, "the refused PATCH must not have written the phone")
		})
	}

	t.Run("nothing in a refused PatchOp is written", func(t *testing.T) {
		status, body, raw := scim(t, http.MethodGet, "/scim/v2/Users/"+sharedID, tokenB, "")
		require.Equal(t, http.StatusOK, status, raw)
		assert.Equal(t, sharedEmail, body["userName"], "org B still sees the original identifier")
		name, _ := body["name"].(map[string]any)
		assert.NotEqual(t, "Mallory", name["givenName"], "a refused PatchOp must be all-or-nothing")
	})

	t.Run("password reset cannot be redirected to the attacker address", func(t *testing.T) {
		// The takeover's second step. On a vulnerable build the shared user's
		// email is now the attacker's, so this mails them a reset link.
		res := gqlRaw(t, baseURL, `mutation { forgot_password(params:{email:"`+attacker+`"}) { message } }`, "")
		require.Empty(t, res.Errors, res.Raw) // generic response by design
		// The real owner's reset is sent after it; once that has arrived the
		// mailer has had its chance to send the attacker's.
		res = gqlRaw(t, baseURL, `mutation { forgot_password(params:{email:"`+sharedEmail+`"}) { message } }`, "")
		require.Empty(t, res.Errors, res.Raw)
		sink.waitMessage(t, sharedEmail, 1)
		assert.Zero(t, sink.count(attacker), "a reset link for the shared account reached the attacker's mailbox")
	})

	t.Run("case-only email change on the shared user still syncs", func(t *testing.T) {
		upper := strings.ToUpper(sharedEmail[:1]) + sharedEmail[1:]
		status, _, raw := scim(t, http.MethodPatch, "/scim/v2/Users/"+sharedID, tokenA,
			patchOp(`{"op":"replace","path":"userName","value":"`+upper+`"}`))
		require.Equal(t, http.StatusOK, status, "an IdP normalising case must not start failing sync: %s", raw)
		email, _ := globalUser(t, sharedID)
		assert.True(t, strings.EqualFold(sharedEmail, email), "still the same mailbox, got %q", email)
	})

	t.Run("single-org user's email change still works", func(t *testing.T) {
		soloID := provision(t, tokenA, "solo-ghsa@smoke248.dev")
		const renamed = "solo-renamed@smoke248.dev"
		status, body, raw := scim(t, http.MethodPatch, "/scim/v2/Users/"+soloID, tokenA,
			patchOp(`{"op":"replace","path":"userName","value":"`+renamed+`"}`))
		require.Equal(t, http.StatusOK, status, raw)
		assert.Equal(t, renamed, body["userName"])
		email, _ := globalUser(t, soloID)
		assert.Equal(t, renamed, email, "directory sync of a single-org user must write through")
	})
}

// --- b. #904 kid header ----------------------------------------------------

type smokeSigningKey struct {
	alg       string
	priv, pub string // PEM
}

func newSmokeRSAKey(t *testing.T) smokeSigningKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return smokeSigningKey{
		alg:  "RS256",
		priv: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})),
		pub:  string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&k.PublicKey)})),
	}
}

func newSmokeECKey(t *testing.T) smokeSigningKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(k)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	require.NoError(t, err)
	return smokeSigningKey{
		alg:  "ES256",
		priv: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})),
		pub:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})),
	}
}

// assertVerifiesAgainstJWKS does what a kid-selecting JWKS consumer (SurrealDB's
// DEFINE ACCESS ... WITH JWT URL, most API gateways) does: read the header's
// kid, pick that key from the JWKS, verify the signature with it. It also
// checks the kid is the key's RFC 7638 thumbprint, as the changelog promises.
func assertVerifiesAgainstJWKS(t *testing.T, tok, wantAlg string, set jose.JSONWebKeySet) string {
	t.Helper()
	h := jwtHeader(t, tok)
	assert.Equal(t, wantAlg, h["alg"])
	kid, _ := h["kid"].(string)
	require.NotEmpty(t, kid, "an %s token must carry a kid header", wantAlg)

	keys := set.Key(kid)
	require.Len(t, keys, 1, "kid %q must name exactly one key in /.well-known/jwks.json", kid)
	thumb, err := keys[0].Thumbprint(crypto.SHA256)
	require.NoError(t, err)
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(thumb), kid, "kid must be the RFC 7638 thumbprint")

	sig, err := jose.ParseSignedCompact(tok, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(wantAlg)})
	require.NoError(t, err)
	_, err = sig.Verify(keys[0].Key)
	require.NoError(t, err, "the JWKS key the kid names must verify the signature")
	return kid
}

// smokeJWTKid: before 2.4.8 asymmetric JWTs had no kid, so JWKS consumers
// that select the key by kid rejected every token.
func smokeJWTKid(t *testing.T, bin string) {
	dbPath := filepath.Join(t.TempDir(), "kid.db")
	const email = "kid@smoke248.dev"

	// boot starts a server with the given signing flags and returns the tokens
	// a login yields plus the JWKS it publishes. Each
	// boot is stopped before the next so they can share dbPath.
	signedUp := false
	boot := func(t *testing.T, signing []string) (access, id string, set jose.JSONWebKeySet) {
		t.Helper()
		p := newSmokePorts(t)
		baseURL := p.baseURL()
		// An asymmetric key leaves no --jwt-secret to derive the encryption
		// key from; the same explicit key serves every boot of this database.
		args := append(withoutJWTFlags(smokeArgs(dbPath, p, "--disable-mfa",
			"--encryption-key=smoke-encryption-key-0123456789")), signing...)
		stop := startServer(t, bin, args, baseURL)
		defer stop()
		gql := newGraphQLClient(t, baseURL)
		if !signedUp {
			signedUp = true
			gql.mutate(t, `mutation { signup(params:{email:"`+email+`", password:"`+smokeUserPassword+`", confirm_password:"`+smokeUserPassword+`"}) { message } }`)
		}
		// login, not signup: signup returns no id_token.
		res := gql.mutate(t, `mutation { login(params:{email:"`+email+`", password:"`+smokeUserPassword+`"}) { access_token id_token } }`)
		access, _ = res["login"].(map[string]any)["access_token"].(string)
		id, _ = res["login"].(map[string]any)["id_token"].(string)
		require.NotEmpty(t, access)
		require.NotEmpty(t, id)
		return access, id, fetchJWKS(t, baseURL)
	}
	keyFlags := func(k smokeSigningKey) []string {
		return []string{"--jwt-type=" + k.alg, "--jwt-private-key=" + k.priv, "--jwt-public-key=" + k.pub}
	}

	rsaKey, ecKey := newSmokeRSAKey(t), newSmokeECKey(t)
	var rsaKid, preRotation string

	t.Run("RS256 access and id tokens name a published kid", func(t *testing.T) {
		access, id, set := boot(t, keyFlags(rsaKey))
		preRotation = access
		rsaKid = assertVerifiesAgainstJWKS(t, access, "RS256", set)
		assert.Equal(t, rsaKid, assertVerifiesAgainstJWKS(t, id, "RS256", set), "both tokens are signed by the one key")
	})

	t.Run("rotation: ES256 primary, old RS256 key as secondary", func(t *testing.T) {
		require.NotEmpty(t, rsaKid, "depends on the RS256 case")
		signing := append(keyFlags(ecKey),
			"--jwt-secondary-type=RS256",
			"--jwt-secondary-private-key="+rsaKey.priv,
			"--jwt-secondary-public-key="+rsaKey.pub)
		access, _, set := boot(t, signing)
		ecKid := assertVerifiesAgainstJWKS(t, access, "ES256", set)
		assert.NotEqual(t, rsaKid, ecKid)
		// The kid follows the key: the RS256 key keeps its kid as secondary,
		// so a token minted before the rotation still resolves and verifies.
		assert.Equal(t, rsaKid, assertVerifiesAgainstJWKS(t, preRotation, "RS256", set),
			"a pre-rotation token must still name a published key")
	})

	t.Run("HS256 tokens carry no kid and nothing is published", func(t *testing.T) {
		access, id, set := boot(t, []string{"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret})
		for _, tok := range []string{access, id} {
			h := jwtHeader(t, tok)
			assert.Equal(t, "HS256", h["alg"])
			assert.NotContains(t, h, "kid", "an HMAC key is never published, so naming it helps no one")
		}
		assert.Empty(t, set.Keys, "an HMAC secret must never appear in the JWKS")
	})
}

// --- c. #913 client_id / azp ---------------------------------------------

// smokeClientIDClaims: 2.4.8 names the client in every access token
// (RFC 9068 client_id) and in every ID token (OIDC azp == aud) from the
// authorization_code, refresh_token and client_credentials grants. The
// authorization_code leg is also asserted in TestReleaseSmoke; it is repeated
// here only because it is how the refresh token is obtained.
func smokeClientIDClaims(t *testing.T, bin string) {
	p := newSmokePorts(t)
	baseURL := p.baseURL()
	startServer(t, bin, smokeArgs(filepath.Join(t.TempDir(), "claims.db"), p, "--disable-mfa"), baseURL)
	gql := newGraphQLClient(t, baseURL)
	gql.mutate(t, `mutation { signup(params:{email:"claims@smoke248.dev", password:"`+smokeUserPassword+`", confirm_password:"`+smokeUserPassword+`"}) { message } }`)

	assertClaims := func(t *testing.T, grant string, tokens map[string]any) {
		t.Helper()
		access, _ := tokens["access_token"].(string)
		require.NotEmpty(t, access, "%s: no access_token in %v", grant, tokens)
		assert.Equal(t, smokeClientID, jwtPayload(t, access)["client_id"], "%s: access token client_id", grant)
		id, _ := tokens["id_token"].(string)
		require.NotEmpty(t, id, "%s: no id_token", grant)
		idClaims := jwtPayload(t, id)
		assert.Equal(t, smokeClientID, idClaims["azp"], "%s: id_token azp", grant)
		assert.Equal(t, idClaims["aud"], idClaims["azp"], "%s: azp must equal aud", grant)
	}

	var refreshToken string
	t.Run("authorization_code + PKCE", func(t *testing.T) {
		verifier := "smoke-913-code-verifier-0123456789abcdefghijk"
		sum := sha256.Sum256([]byte(verifier))
		code := authorizeCode(t, gql, baseURL, "openid profile email offline_access",
			base64.RawURLEncoding.EncodeToString(sum[:]))
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("redirect_uri", baseURL+"/smoke-callback")
		form.Set("client_id", smokeClientID)
		form.Set("client_secret", smokeClientSecret)
		form.Set("code_verifier", verifier)
		status, tokens := oauthToken(t, baseURL, form)
		require.Equal(t, http.StatusOK, status, "%v", tokens)
		assertClaims(t, "authorization_code", tokens)
		refreshToken, _ = tokens["refresh_token"].(string)
		require.NotEmpty(t, refreshToken, "offline_access must yield a refresh_token")
	})

	t.Run("refresh_token", func(t *testing.T) {
		require.NotEmpty(t, refreshToken, "depends on the authorization_code case")
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", refreshToken)
		form.Set("client_id", smokeClientID)
		form.Set("client_secret", smokeClientSecret)
		status, tokens := oauthToken(t, baseURL, form)
		require.Equal(t, http.StatusOK, status, "%v", tokens)
		assertClaims(t, "refresh_token", tokens)
	})

	t.Run("client_credentials names the service account, not the default client", func(t *testing.T) {
		gql.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)
		created := gql.mutate(t, `mutation { _create_client(params:{name:"smoke-913-m2m", allowed_scopes:["openid"]}) { client { client_id } client_secret } }`)
		c := created["_create_client"].(map[string]any)
		clientID := c["client"].(map[string]any)["client_id"].(string)
		require.NotEqual(t, smokeClientID, clientID)

		form := url.Values{}
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", clientID)
		form.Set("client_secret", c["client_secret"].(string))
		status, tokens := oauthToken(t, baseURL, form)
		require.Equal(t, http.StatusOK, status, "%v", tokens)
		access, _ := tokens["access_token"].(string)
		require.NotEmpty(t, access)
		assert.Equal(t, clientID, jwtPayload(t, access)["client_id"])
		assert.NotContains(t, tokens, "id_token", "RFC 6749 §4.4.3: no id_token for client_credentials")
	})
}

// --- d. #903 list_permissions truncation -----------------------------------

// smokeListPermissionsTruncated: up to 2.4.6 the embedded OpenFGA engine
// capped each list call at 1000 — the same as the service's cap — so a
// relation with more matches came back as exactly 1000 with truncated=false.
// The cap is a compile-time constant (service.maxFgaListResults), so this
// seeds cap+1 grants: 11 admin writes of at most 100 tuples.
func smokeListPermissionsTruncated(t *testing.T, bin string) {
	const listCap = 1000
	p := newSmokePorts(t)
	baseURL := p.baseURL()
	startServer(t, bin, smokeArgs(filepath.Join(t.TempDir(), "fga.db"), p, "--disable-mfa"), baseURL)
	gql := newGraphQLClient(t, baseURL)
	gql.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)
	gql.mutate(t, `mutation { _fga_write_model(params:{dsl:"`+strings.ReplaceAll(fgaModelDSL, "\n", `\n`)+`"}) { id } }`)
	grpcClient := authorizerv1.NewAuthorizerServiceClient(smokeGRPCConn(t, p))

	newUser := func(t *testing.T, email string, grants int) (string, string) {
		t.Helper()
		res := gql.mutate(t, `mutation { signup(params:{email:"`+email+`", password:"`+smokeUserPassword+`", confirm_password:"`+smokeUserPassword+`"}) { access_token user { id } } }`)
		s := res["signup"].(map[string]any)
		id := s["user"].(map[string]any)["id"].(string)
		for start := 0; start < grants; start += 100 {
			var b strings.Builder
			for i := start; i < start+100 && i < grants; i++ {
				fmt.Fprintf(&b, `{user:"user:%s", relation:"viewer", object:"document:d%04d"},`, id, i)
			}
			gql.mutate(t, `mutation { _fga_write_tuples(params:{tuples:[`+b.String()+`]}) { message } }`)
		}
		return s["access_token"].(string), id
	}

	cases := []struct {
		name          string
		email         string
		grants        int
		wantLen       int
		wantTruncated bool
	}{
		{"over the cap -> cap results, truncated", "fga-over@smoke247.dev", listCap + 1, listCap, true},
		{"under the cap -> all results, not truncated", "fga-under@smoke247.dev", 3, 3, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start := time.Now()
			token, _ := newUser(t, c.email, c.grants)
			t.Logf("seeded %d grants in %s", c.grants, time.Since(start).Round(time.Millisecond))

			data := gql.query(t, token, `query { list_permissions(params:{relation:"viewer", object_type:"document"}) { objects truncated } }`)
			lp := data["list_permissions"].(map[string]any)
			assert.Len(t, lp["objects"], c.wantLen, "graphql")
			assert.Equal(t, c.wantTruncated, lp["truncated"], "graphql")

			var rest struct {
				Objects   []string `json:"objects"`
				Truncated bool     `json:"truncated"`
			}
			status := restJSON(t, baseURL, "/v1/list_permissions", token, `{"relation":"viewer","object_type":"document"}`, &rest)
			require.Equal(t, http.StatusOK, status)
			assert.Len(t, rest.Objects, c.wantLen, "rest")
			assert.Equal(t, c.wantTruncated, rest.Truncated, "rest")

			res, err := grpcClient.ListPermissions(
				grpcCtx(t, "authorization", "Bearer "+token, "x-authorizer-url", baseURL),
				&authorizerv1.ListPermissionsRequest{Relation: "viewer", ObjectType: "document"})
			require.NoError(t, err)
			assert.Len(t, res.GetObjects(), c.wantLen, "grpc")
			assert.Equal(t, c.wantTruncated, res.GetTruncated(), "grpc")
		})
	}
}
