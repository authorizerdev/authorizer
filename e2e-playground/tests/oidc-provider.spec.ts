// e2e-playground/tests/oidc-provider.spec.ts
import { test, expect } from '@playwright/test';
import { GraphQLClient, gql } from 'graphql-request';
import crypto from 'node:crypto';

const BASE_URL = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';
// CSRF middleware requires Origin/Referer on state-changing requests (see
// internal/http_handlers/csrf.go) — same rationale as fixtures/adminClient.ts.
const client = new GraphQLClient(`${BASE_URL}/graphql`, { headers: { Origin: BASE_URL } });

function randomEmail() {
  return `oidc-provider-${crypto.randomUUID()}@example.com`;
}

test.describe('OIDC — provider side', () => {
  test('discovery document matches actual endpoint behavior', async ({ request, baseURL }) => {
    const res = await request.get('/.well-known/openid-configuration');
    expect(res.status()).toBe(200);
    const doc = await res.json();
    expect(doc.authorization_endpoint).toBe(`${baseURL}/authorize`);
    expect(doc.token_endpoint).toBe(`${baseURL}/oauth/token`);
    expect(doc.jwks_uri).toBe(`${baseURL}/.well-known/jwks.json`);

    const jwksRes = await request.get('/.well-known/jwks.json');
    expect(jwksRes.status()).toBe(200);
    const jwks = await jwksRes.json();
    // This stack runs --jwt-type=HS256 (docker-compose.yml). HMAC keys are
    // symmetric and must never be exposed via JWKS (internal/http_handlers/jwks.go
    // generateJWKFromKey) — the keys array must be exactly empty. Asserting
    // the exact value (not just Array.isArray) catches a future regression
    // where an HMAC secret starts leaking into the JWKS response.
    expect(jwks.keys).toEqual([]);
  });

  test('signup then password login issues a valid session', async () => {
    const email = randomEmail();
    const signup = gql`
      mutation ($params: SignUpRequest!) {
        signup(params: $params) { message }
      }
    `;
    await client.request(signup, { params: { email, password: 'Str0ngPassw0rd!', confirm_password: 'Str0ngPassw0rd!' } });

    const login = gql`
      mutation ($params: LoginRequest!) {
        login(params: $params) { message }
      }
    `;
    const loginRes = await client.request<{ login: { message: string } }>(login, {
      params: { email, password: 'Str0ngPassw0rd!' },
    });
    expect(loginRes.login.message).toBeTruthy();
  });

  test('a fabricated authorization code is rejected', async ({ request }) => {
    // Unknown/fabricated code — proves /oauth/token doesn't accept arbitrary
    // codes. Real single-use (replay of a genuine, once-used code) is
    // covered separately below, where a real code is minted via a live
    // browser-driven PKCE /authorize flow.
    const res = await request.post('/oauth/token', {
      form: { grant_type: 'authorization_code', code: 'not-a-real-code', client_id: 'authorizer' },
    });
    expect(res.status()).toBeGreaterThanOrEqual(400);
  });

  test('real PKCE authorization-code flow exchanges a code for tokens, and replay is rejected', async ({
    page,
    request,
  }) => {
    // client_id/redirect_uri must match what this stack's own login UI is
    // configured for (docker-compose.yml: --client-id=e2e-client-id,
    // --allowed-origins=http://localhost:8080). No client is registered in
    // the clients table for e2e-client-id, so /authorize (internal/http_handlers/authorize.go)
    // falls back to validating redirect_uri's origin against AllowedOrigins —
    // any path under http://localhost:8080 is accepted.
    const clientId = 'e2e-client-id';
    const redirectUri = `${BASE_URL}/e2e-callback`;
    const email = randomEmail();
    const password = 'Str0ngPassw0rd!';

    const signup = gql`
      mutation ($params: SignUpRequest!) {
        signup(params: $params) { message }
      }
    `;
    await client.request(signup, { params: { email, password, confirm_password: password } });

    // RFC 7636 PKCE: verifier is a random string, challenge is its S256 hash.
    const codeVerifier = crypto.randomBytes(32).toString('base64url');
    const codeChallenge = crypto.createHash('sha256').update(codeVerifier).digest('base64url');
    const state = crypto.randomUUID();

    const authorizeUrl = new URL('/authorize', BASE_URL);
    authorizeUrl.searchParams.set('response_type', 'code');
    authorizeUrl.searchParams.set('client_id', clientId);
    authorizeUrl.searchParams.set('redirect_uri', redirectUri);
    authorizeUrl.searchParams.set('scope', 'openid');
    authorizeUrl.searchParams.set('state', state);
    authorizeUrl.searchParams.set('code_challenge', codeChallenge);
    authorizeUrl.searchParams.set('code_challenge_method', 'S256');

    await page.goto(authorizeUrl.toString());

    // Real rendered /app login form (web/app/src/pages/login.tsx →
    // AuthorizerBasicAuthLogin in authorizer-react): email/phone field id
    // authorizer-login-email-or-phone-number, password field id
    // authorizer-login-password, submit button inside the named form.
    await page.locator('#authorizer-login-email-or-phone-number').fill(email);
    await page.locator('#authorizer-login-password').fill(password);
    await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();

    // First-time login for a brand-new user hits the optional MFA-setup
    // offer screen (withheld-token first-time-setup redesign — see
    // authorizer-react AuthorizerMFASetup / commit 992b3bf4): the token is
    // withheld until a factor is set up or skipped. "Skip for now" calls
    // skipMfaSetup, which issues the token and completes login. click()
    // waits for the button up to its own timeout, so this also tolerates
    // the offer screen not appearing at all.
    await page
      .getByRole('button', { name: 'Skip for now' })
      .click({ timeout: 10_000 })
      .catch(() => {});

    // On success the SPA (web/app/src/Root.tsx) resumes the authorization
    // request against /authorize using the now-authenticated session cookie,
    // and the server issues the final redirect to redirect_uri?code=...&state=....
    await page.waitForURL((url) => url.origin === BASE_URL && url.pathname === '/e2e-callback' && url.searchParams.has('code'));

    const finalUrl = new URL(page.url());
    const code = finalUrl.searchParams.get('code');
    expect(code).toBeTruthy();
    expect(finalUrl.searchParams.get('state')).toBe(state);

    const tokenForm = {
      grant_type: 'authorization_code',
      code: code!,
      client_id: clientId,
      redirect_uri: redirectUri,
      code_verifier: codeVerifier,
    };

    const tokenRes = await request.post('/oauth/token', { form: tokenForm });
    expect(tokenRes.status()).toBe(200);
    const tokenBody = await tokenRes.json();
    expect(tokenBody.access_token).toBeTruthy();
    expect(tokenBody.id_token).toBeTruthy();

    // RFC 6749 §4.1.2: authorization codes MUST be single-use. Replaying the
    // exact same (real, already-consumed) code must now be rejected.
    const replayRes = await request.post('/oauth/token', { form: tokenForm });
    expect(replayRes.status()).toBeGreaterThanOrEqual(400);
  });

  test('redirect URI not in allowlist is rejected at /authorize', async ({ page }) => {
    await page.goto(
      '/authorize?response_type=code&client_id=authorizer&redirect_uri=https://evil.example.com/cb&scope=openid&state=xyz'
    );
    await expect(page.locator('body')).toContainText(/invalid redirect uri|redirect_uri/i);
  });
});
