// e2e-playground/tests/social/github.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath, configureProviderProfile } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

const BASE_URL = process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080';
// The #855 client's callback: loopback, so NOT in --allowed-origins.
const resumeRedirectURI = 'http://127.0.0.1:47822/callback';

test.describe('Social login — GitHub', () => {
  test('first-time signup via GitHub creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `github-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'github',
      buttonName: /github/i,
      // GitHub is a REST-profile provider (not OIDC-verified): mock-oauth's
      // /github/userinfo route returns this JSON verbatim, and
      // processGithubUserInfo (internal/http_handlers/oauth_callback.go)
      // decodes it into map[string]string, then splits `name` on the first
      // space into given_name/family_name - keep every field a plain string.
      profile: { name: 'Grace Hopper', email, avatar_url: 'https://example.com/a.png' },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves a real session; this
    // proves the given_name/family_name split actually landed on the stored
    // user, and that "github" was recorded as the signup method.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Grace');
    expect(user.family_name).toBe('Hopper');
    expect(user.signup_methods).toContain('github');
  });

  test('empty primary email falls back to the /user/emails endpoint', async ({ page, request }) => {
    // Real GitHub behavior: profile.email is "" when the user has their
    // email set to private, and processGithubUserInfo falls back to
    // GET /user/emails for the primary address. mock-oauth's
    // /:provider/user/emails handler (server.ts) ignores whatever email was
    // configured and always serves a hardcoded fallback address, so
    // configuring email: '' here exercises the real fallback code path
    // deterministically rather than the happy-path email-passthrough above.
    const fallbackEmail = 'mock-user@github.example.com';
    await runSocialLoginHappyPath(page, request, {
      provider: 'github',
      buttonName: /github/i,
      profile: { name: 'Grace Hopper', email: '', avatar_url: '' },
      expectedEmail: fallbackEmail,
    });

    // Proves the fallback /user/emails roundtrip actually happened - the
    // stored user has the fallback address, not an empty one, and the name
    // split still landed correctly off the same /userinfo response.
    const user = await getUserByEmail(fallbackEmail);
    expect(user.email).toBe(fallbackEmail);
    expect(user.given_name).toBe('Grace');
    expect(user.family_name).toBe('Hopper');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'github');
  });

  // #855: social login inside an /authorize request resumes through
  // /authorize. The client registers a loopback callback that is NOT in this
  // stack's --allowed-origins. Before the fix the GitHub button on /app sent
  // that callback straight to /oauth_login and got {"error":"invalid redirect
  // uri"}; had it been allowed, the self-registered client would have received
  // a code with no consent screen. Only a real browser proves the login page's
  // button now sends its own /app URL, the callback returns there, /app replays
  // the request to /authorize, and consent comes BEFORE anything reaches the
  // client. The final consent hop is driven with page.request for the same
  // Chromium Private Network Access reason documented in dcr.spec.ts. Lives in
  // this file, not its own, because every test here shares mock-oauth's single
  // GitHub profile and Playwright runs files in parallel.
  test('#855: inside /authorize, a client outside --allowed-origins completes GitHub login, after consent', async ({
    page,
    request,
  }) => {
    const regRes = await request.post('/oauth/register', {
      headers: { 'Content-Type': 'application/json' },
      data: {
        client_name: 'E2E Social Resume Client',
        redirect_uris: [resumeRedirectURI],
        grant_types: ['authorization_code', 'refresh_token'],
        token_endpoint_auth_method: 'none',
      },
    });
    expect(regRes.status(), await regRes.text()).toBe(201);
    const clientId = (await regRes.json()).client_id as string;

    const email = `resume-${crypto.randomUUID()}@example.com`;
    // Same profile shape as github.spec.ts.
    await configureProviderProfile(request, 'github', {
      name: 'Resume User',
      email,
      avatar_url: 'https://example.com/a.png',
    });

    const codeVerifier = crypto.randomBytes(32).toString('base64url');
    const codeChallenge = crypto.createHash('sha256').update(codeVerifier).digest('base64url');
    const nonce = crypto.randomUUID();
    const authorizeUrl = new URL('/authorize', BASE_URL);
    authorizeUrl.searchParams.set('response_type', 'code');
    authorizeUrl.searchParams.set('response_mode', 'query');
    authorizeUrl.searchParams.set('client_id', clientId);
    authorizeUrl.searchParams.set('redirect_uri', resumeRedirectURI);
    authorizeUrl.searchParams.set('scope', 'openid profile email');
    authorizeUrl.searchParams.set('state', crypto.randomUUID());
    authorizeUrl.searchParams.set('nonce', nonce);
    authorizeUrl.searchParams.set('code_challenge', codeChallenge);
    authorizeUrl.searchParams.set('code_challenge_method', 'S256');

    // Record every request the browser makes to the client's callback; there
    // must be none before consent.
    const clientHits: string[] = [];
    page.on('request', (req) => {
      if (req.url().startsWith(resumeRedirectURI)) clientHits.push(req.url());
    });

    // The login button must send this page's own /app URL, never the client's.
    const oauthLogin = page.waitForRequest((req) => new URL(req.url()).pathname === '/oauth_login/github');
    await page.goto(authorizeUrl.toString());
    await page.getByRole('button', { name: /github/i }).click();
    const loginURL = new URL((await oauthLogin).url());
    const sent = new URL(loginURL.searchParams.get('redirect_uri')!);
    expect(sent.origin).toBe(new URL(BASE_URL).origin);
    expect(sent.pathname).toMatch(/^\/app\/?$/); // the browser is on /app/ (the server 301s /app)
    expect(sent.searchParams.get('client_id')).toBe(clientId);
    expect(loginURL.searchParams.has('redirectURL')).toBe(false);

    // Back from GitHub, /app resumes /authorize, which shows consent.
    await page
      .getByRole('button', { name: 'Skip for now' })
      .click({ timeout: 10_000 })
      .catch(() => {});
    await expect(page.getByRole('heading', { name: /E2E Social Resume Client/ })).toBeVisible({ timeout: 15_000 });
    expect(clientHits, 'nothing reaches the client before consent').toEqual([]);

    const consentID = await page.locator('form[action="/authorize/consent"] input[name="consent_id"]').inputValue();
    const consentRes = await page.request.post('/authorize/consent', {
      form: { consent_id: consentID, action: 'approve' },
      maxRedirects: 0,
    });
    expect(consentRes.status(), await consentRes.text()).toBe(302);
    const resumed = await page.request.get(new URL(consentRes.headers()['location'], BASE_URL).toString(), {
      maxRedirects: 0,
    });
    expect(resumed.status(), await resumed.text()).toBe(302);

    const landed = new URL(resumed.headers()['location']);
    expect(landed.origin + landed.pathname).toBe(resumeRedirectURI);
    const code = landed.searchParams.get('code')!;
    expect(code).toBeTruthy();

    const tokenRes = await request.post('/oauth/token', {
      form: {
        grant_type: 'authorization_code',
        code,
        client_id: clientId,
        redirect_uri: resumeRedirectURI,
        code_verifier: codeVerifier,
      },
    });
    expect(tokenRes.status(), await tokenRes.text()).toBe(200);
    const tokens = await tokenRes.json();
    expect(tokens.access_token).toBeTruthy();

    // The client's own nonce reached the ID token.
    const claims = JSON.parse(Buffer.from(tokens.id_token.split('.')[1], 'base64url').toString());
    expect(claims.nonce).toBe(nonce);
    expect(claims.email).toBe(email);
  });
});
