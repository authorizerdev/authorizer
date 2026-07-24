// e2e-playground/tests/magic-link.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';

// This spec runs against authorizer-magic-link (docker-compose.yml, `magic-link`
// project in playwright.config.ts) - the only instance with
// --enable-magic-link-login=true AND --enable-email-verification=true (both
// required for magic_link_login to actually enqueue a verification email -
// internal/service/magic_link_login.go gates the whole
// verification-request+SendEmail block on Config.EnableEmailVerification) and
// --disable-mfa=true (keeps this a clean primary-login-method test - see that
// service's comment in docker-compose.yml for why MFA would otherwise let the
// same link be clicked more than once without failing).
const MAILPIT_BASE = process.env.MAILPIT_BASE_URL || 'http://localhost:8025';

function randomEmail() {
  return `magic-link-${crypto.randomUUID()}@example.com`;
}

// waitForMagicLinkURL polls Mailpit's real HTTP API for the verification link
// mailed to `email`. Confirmed live against a running axllent/mailpit
// container (not the plan's untested guess): GET /api/v1/messages returns
// {messages: [{ID, To: [{Address}], ...}]} - a summary list, body NOT
// included - so the link itself only shows up via GET /api/v1/message/:id,
// which returns the full {Text, HTML, ...}. The link is pulled from Text, not
// HTML - Mailpit's HTML body entity-encodes the URL's "&" as "&amp;", which
// would otherwise need decoding before the URL is directly usable.
async function waitForMagicLinkURL(request: import('@playwright/test').APIRequestContext, email: string): Promise<string> {
  for (let i = 0; i < 40; i++) {
    const res = await request.get(`${MAILPIT_BASE}/api/v1/messages`);
    const { messages } = await res.json();
    const msg = messages.find((m: { To: { Address: string }[] }) => m.To.some((t) => t.Address === email));
    if (msg) {
      const detail = await (await request.get(`${MAILPIT_BASE}/api/v1/message/${msg.ID}`)).json();
      const match = /(https?:\/\/\S+verify_email\?\S+)/.exec(detail.Text);
      if (match) return match[1].replace(/\)$/, '');
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`no magic link email received for ${email} within 10s`);
}

const magicLinkLoginMutation = `
  mutation ($params: MagicLinkLoginRequest!) {
    magic_link_login(params: $params) { message }
  }
`;

test.describe('Magic link login', () => {
  test('request link, click it, land on the dashboard with a session', async ({ page, request, baseURL }) => {
    const email = randomEmail();
    await graphql(request, baseURL!, magicLinkLoginMutation, { params: { email } });

    const link = await waitForMagicLinkURL(request, email);
    await page.goto(link);

    // Real rendered dashboard text (web/app/src/pages/dashboard.tsx: "Signed
    // in as <a href=mailto:...>{email}</a>") - the same established
    // session-established assertion tests/sms-otp.spec.ts and
    // tests/social/helpers.ts use, stronger than checking for the absence of
    // an "error" string.
    await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('link', { name: email })).toBeVisible();
  });

  test('link is single-use: replaying it after a successful click is rejected', async ({ request, baseURL }) => {
    const email = randomEmail();
    await graphql(request, baseURL!, magicLinkLoginMutation, { params: { email } });

    const link = await waitForMagicLinkURL(request, email);

    // GET /verify_email (internal/http_handlers/verify_email.go) always
    // redirects (307 Temporary Redirect) - even on error - instead of
    // returning a >=400 status, so a browser landing on the link never sees
    // a raw JSON error page; the error is carried back as a query param on
    // the /app redirect instead. Confirmed live against this stack: a
    // genuine first click's Location carries access_token=...; replaying the
    // exact same link carries error=... instead, with the SAME 307 status
    // both times. maxRedirects: 0 stops Playwright from auto-following so
    // the Location header itself can be inspected.
    const first = await request.get(link, { maxRedirects: 0 });
    expect(first.status()).toBe(307);
    const firstLocation = first.headers()['location'];
    expect(firstLocation).toContain('access_token=');

    const second = await request.get(link, { maxRedirects: 0 });
    expect(second.status()).toBe(307);
    const secondLocation = second.headers()['location'];
    expect(secondLocation).not.toContain('access_token=');
    expect(secondLocation).toContain('error=');
  });
});
