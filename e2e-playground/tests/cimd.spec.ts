// e2e-playground/tests/cimd.spec.ts
//
// Client ID Metadata Documents: a client identifies itself with an https URL
// pointing at a JSON document, instead of a client_id registered in advance. It
// is what lets a client with no prior relationship authenticate at all —
// without it Claude Code refuses the server outright ("Incompatible auth
// server: does not support dynamic client registration").
//
// This is the ONLY test that drives the browser leg. The Go tests cover the
// resolver and the consent handler's rules; only a real browser proves a user
// is shown the consent page and that approving it yields a code.
//
// It needs real TLS: the spec requires an https client_id and the SERVER
// fetches it, so the document host must present a certificate the server
// trusts. The tls-certs service generates a CA into a shared volume and the
// server is pointed at it via SSL_CERT_FILE — so the certificate path is
// exercised, not bypassed.
import { test, expect, type Page } from '@playwright/test';
import { GraphQLClient, gql } from 'graphql-request';
import crypto from 'node:crypto';

const BASE_URL = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';
const CIMD_BASE = process.env.CIMD_CLIENT_BASE_URL || 'https://cimd-client:4300';

// The client_id IS this URL, and the document served there must claim the same
// value — the equality that stops any host claiming to be any client.
const CLIENT_ID = `${CIMD_BASE}/client.json`;
const REDIRECT_URI = `${CIMD_BASE}/callback`;

const client = new GraphQLClient(`${BASE_URL}/graphql`, { headers: { Origin: BASE_URL } });

function authorizeURL(clientID: string, state: string) {
  const u = new URL('/authorize', BASE_URL);
  u.searchParams.set('response_type', 'code');
  u.searchParams.set('client_id', clientID);
  u.searchParams.set('redirect_uri', REDIRECT_URI);
  u.searchParams.set('scope', 'openid');
  u.searchParams.set('state', state);
  u.searchParams.set('response_mode', 'query');
  const verifier = crypto.randomBytes(32).toString('base64url');
  u.searchParams.set('code_challenge', crypto.createHash('sha256').update(verifier).digest('base64url'));
  u.searchParams.set('code_challenge_method', 'S256');
  return u.toString();
}

async function signupAndReach(page: Page, url: string) {
  const email = `cimd-${crypto.randomUUID()}@example.com`;
  const password = 'Str0ngPassw0rd!';
  await client.request(
    gql`mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`,
    { params: { email, password, confirm_password: password } },
  );
  await page.goto(url);
  await page.locator('#authorizer-login-email-or-phone-number').fill(email);
  await page.locator('#authorizer-login-password').fill(password);
  await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();
  // First login for a new user hits the optional MFA-setup offer.
  await page.getByRole('button', { name: 'Skip for now' }).click({ timeout: 10_000 }).catch(() => {});
}

// Assert on the redirect the SERVER issues, captured as the browser attempts it,
// rather than on the browser landing successfully.
//
// The callback host serves TLS signed by the e2e CA, which Chromium does not
// trust — and it should not have to: what is under test is that Authorizer sends
// an authorization code to the registered redirect URI, which is fully
// determined by the moment the request is made. Waiting for the navigation to
// COMPLETE would additionally require the browser to trust our test CA, making
// the assertion depend on something irrelevant to the behaviour.
function awaitCallback(page: Page) {
  return page.waitForRequest((req) => req.url().includes('/callback'), { timeout: 20_000 });
}

// The mock's certificate is signed by the e2e CA, which Chromium does not
// trust. The SERVER's trust is what this suite is about and is configured
// properly via SSL_CERT_FILE; the browser only has to survive the final
// redirect to the callback.
test.use({
  ignoreHTTPSErrors: true,
  launchOptions: { args: ['--ignore-certificate-errors'] },
});

// SCOPE — this file asserts only what a browser is uniquely good for: that the
// consent page renders, names the client, shows the redirect host, and is
// clickable.
//
// The security property — approving issues a code to the registered redirect,
// declining returns access_denied and no code — is asserted in Go, by
// TestCIMDConsentEndToEnd (internal/integration_tests/cimd_flow_test.go), which
// drives the same four requests over http.Client with a cookie jar.
//
// That split is deliberate. The consent page is plain HTML with no JavaScript,
// so a browser adds nothing to an assertion about status codes and Location
// headers — and trying to make it do so failed for a reason unrelated to the
// feature: the flow ends in a cross-origin redirect into an https host whose
// certificate is signed by a throwaway CA, and Chromium cancels that navigation
// (net::ERR_ABORTED, canceled=true, no response event) even with
// ignoreHTTPSErrors AND --ignore-certificate-errors. Measured, not assumed:
// a DIRECT navigation to the same host returns 200, so it is the redirect into
// it that Chromium refuses. Reading the Location in Go sidesteps the whole
// question.
test.describe('CIMD — self-registered clients', () => {
  test('the authorization server advertises the capability', async ({ request }) => {
    // Anthropic documents Claude selecting CIMD only when BOTH hold — the
    // second because its CIMD client authenticates as a public client. A server
    // advertising one without the other silently falls back to DCR, which this
    // server deliberately does not implement.
    const res = await request.get('/.well-known/oauth-authorization-server');
    expect(res.status()).toBe(200);
    const doc = await res.json();
    expect(doc.client_id_metadata_document_supported).toBe(true);
    expect(doc.token_endpoint_auth_methods_supported).toContain('none');
  });

  test('the consent page names the client and shows the redirect host', async ({ page }) => {
    await signupAndReach(page, authorizeURL(CLIENT_ID, crypto.randomUUID()));

    // A pre-registered client would have gone straight to the callback. The
    // page appearing at all is the gate working.
    await expect(page.getByRole('heading', { name: /E2E Playground Client/ })).toBeVisible();

    // The redirect HOST is the only verified fact about a self-asserted client,
    // so it must be presented as its own field. Targeted by class rather than
    // text, because the host also appears inside the full client_id below and
    // matching either would pass without proving it is shown on its own.
    await expect(page.locator('.host')).toHaveText('cimd-client:4300');

    // Both decisions must be offered; which one produces what is asserted in Go.
    await expect(page.getByRole('button', { name: 'Allow access' })).toBeEnabled();
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeEnabled();
  });
});
