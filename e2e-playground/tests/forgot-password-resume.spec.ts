// e2e-playground/tests/forgot-password-resume.spec.ts
//
// Guards #923: Forgot Password started from a login page that /authorize sent
// the user to must return them to that same /authorize request after the
// reset — and a reset link (which anyone can request for any address) must
// never silently authorize an already-signed-in user into a client.
//
// Runs in the `email-verify` project against authorizer-email-verify: the only
// instance with basic-auth login AND a working email service (SMTP to Mailpit
// with a sender address), with MFA disabled so login completes in one step.
import { test, expect, APIRequestContext } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';
import { waitForEmailLink, RESET_PASSWORD_LINK } from '../fixtures/mailpit';
import { getUserIdByEmail, verifyUserEmail } from '../fixtures/adminClient';
import { newAuthorizeRequest, redeemCode, stubCallback, CLIENT_ID } from '../fixtures/oauth';

const OLD_PASSWORD = 'Str0ngPassw0rd!';
const NEW_PASSWORD = 'N3wStr0ngPassw0rd!';
const CALLBACK = '/e2e-callback';

function randomEmail() {
  return `reset-${crypto.randomUUID()}@example.com`;
}

async function signup(request: APIRequestContext, baseURL: string, email: string) {
  await graphql(request, baseURL, `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`, {
    params: { email, password: OLD_PASSWORD, confirm_password: OLD_PASSWORD },
  });
}

async function loginErrors(request: APIRequestContext, baseURL: string, email: string, password: string) {
  const res = await request.post(`${baseURL}/graphql`, {
    data: { query: `mutation ($params: LoginRequest!) { login(params: $params) { message } }`, variables: { params: { email, password } } },
    headers: { Origin: baseURL },
  });
  return (await res.json()).errors;
}

test.describe('Forgot password resumes /authorize (#923)', () => {
  test('logged out: reset returns to /app with the authorize params, login reaches the RP with a code', async ({
    page,
    request,
    baseURL,
  }) => {
    await stubCallback(page, baseURL!, CALLBACK);
    const email = randomEmail();
    await signup(request, baseURL!, email);
    const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}`);

    await page.goto(req.url);
    await page.waitForURL((u) => /^\/app\/?$/.test(u.pathname));

    await page.getByText('Forgot Password?').click();
    await page.locator('#authorizer-forgot-password-email-or-phone-number').fill(email);
    await page.locator('form[name="authorizer-forgot-password-form"] button[type="submit"]').click();

    const link = await waitForEmailLink(request, email, RESET_PASSWORD_LINK);
    const linkUrl = new URL(link);
    expect(linkUrl.searchParams.get('token')).toBeTruthy();
    expect(linkUrl.searchParams.get('state')).toBe(req.state);
    expect(linkUrl.searchParams.get('redirect_uri')).toBe(req.redirectUri);

    await page.goto(link);
    await page.locator('#authorizer-reset-password').fill(NEW_PASSWORD);
    await page.locator('#authorizer-reset-confirm-password').fill(NEW_PASSWORD);
    await page.locator('form[name="authorizer-reset-password-form"] button[type="submit"]').click();

    // Back on the login page with the original request, minus the token.
    await page.waitForURL((u) => /^\/app\/?$/.test(u.pathname) && u.searchParams.get('state') === req.state);
    const resumed = new URL(page.url());
    expect(resumed.searchParams.get('redirect_uri')).toBe(req.redirectUri);
    expect(resumed.searchParams.get('client_id')).toBe(CLIENT_ID);
    expect(resumed.searchParams.has('token'), 'reset token must be stripped').toBe(false);

    await page.locator('#authorizer-login-email-or-phone-number').fill(email);
    await page.locator('#authorizer-login-password').fill(NEW_PASSWORD);
    await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();

    await page.waitForURL((u) => u.pathname === CALLBACK && u.searchParams.has('code'));
    const cb = new URL(page.url());
    expect(cb.searchParams.get('state')).toBe(req.state);
    const tokenRes = await redeemCode(request, baseURL!, cb.searchParams.get('code')!, req);
    expect(tokenRes.status(), await tokenRes.text()).toBe(200);

    expect(await loginErrors(request, baseURL!, email, OLD_PASSWORD), 'old password must be rejected').toBeTruthy();
    expect(await loginErrors(request, baseURL!, email, NEW_PASSWORD)).toBeFalsy();
  });

  test('signed in: a reset link carrying authorize params lands on /app, never on /authorize', async ({
    page,
    request,
    baseURL,
  }) => {
    const email = randomEmail();
    await signup(request, baseURL!, email);
    // Login on this instance needs a verified address.
    await verifyUserEmail(await getUserIdByEmail(email, baseURL!), { givenName: 'Reset', familyName: 'Probe' }, baseURL!);

    // Sign in this browser context (page.request shares its cookie jar).
    await graphql(page.request, baseURL!, `mutation ($params: LoginRequest!) { login(params: $params) { message } }`, {
      params: { email, password: OLD_PASSWORD },
    });

    // The link the login UI would have requested mid-/authorize.
    const authorizeQuery = new URLSearchParams({
      state: crypto.randomUUID(),
      client_id: CLIENT_ID,
      response_type: 'code',
      scope: 'openid',
      redirect_uri: `${baseURL}${CALLBACK}`,
    });
    await graphql(request, baseURL!, `mutation ($params: ForgotPasswordRequest!) { forgot_password(params: $params) { message } }`, {
      params: { email, redirect_uri: `${baseURL}/app/reset-password?${authorizeQuery}` },
    });
    const link = await waitForEmailLink(request, email, RESET_PASSWORD_LINK);
    const resetToken = new URL(link).searchParams.get('token')!;
    expect(resetToken).toBeTruthy();

    const authorizeHits: string[] = [];
    const leaked: string[] = [];
    page.on('request', (r) => {
      const u = new URL(r.url());
      if (u.pathname === '/authorize') authorizeHits.push(r.url());
      if (r.url() !== link && r.url().includes(resetToken)) leaked.push(r.url());
    });
    const urls: string[] = [];
    page.on('framenavigated', (f) => {
      if (f === page.mainFrame()) urls.push(f.url());
    });

    await page.goto(link);
    await expect(page.getByText('Signed in as')).toBeVisible();
    await expect(page).toHaveURL((u) => /^\/app\/?$/.test(u.pathname));
    expect(urls.length, 'the app moved off the reset page').toBeGreaterThan(1);
    await page.waitForLoadState('networkidle');

    expect(authorizeHits, 'no /authorize navigation from a reset link').toEqual([]);
    expect(leaked, 'reset token must not be forwarded anywhere').toEqual([]);
    expect(new URL(page.url()).searchParams.has('token')).toBe(false);
    for (const u of urls.filter((u) => u !== link)) expect(u, 'no later URL may carry token=').not.toContain('token=');
  });
});
