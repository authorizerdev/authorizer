// e2e-playground/tests/login-flash.spec.ts
//
// Guards #921: after a login on a page /authorize sent the user to, the login
// UI resumes /authorize with a full navigation. Until that navigation commits
// the old document stays on screen, and it used to render the signed-in
// dashboard ("Signed in as ...") for that whole time. The /authorize resume is
// slowed down here so the window is wide enough for any flash to be seen, and
// a MutationObserver installed in every document reports the dashboard text
// the moment it enters the DOM — what a real user would see, not a final
// state.
//
// Runs in the `email-verify` project (authorizer-email-verify: basic auth,
// MFA disabled, so a login completes straight into the resume).
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';
import { getUserIdByEmail, verifyUserEmail } from '../fixtures/adminClient';
import { newAuthorizeRequest, stubCallback } from '../fixtures/oauth';

const PASSWORD = 'Str0ngPassw0rd!';
const CALLBACK = '/e2e-callback';
const DASHBOARD_TEXT = 'Signed in as';
// How long the resumed /authorize request is held. Long enough that a
// dashboard render in between cannot be missed; it is the window under test.
const RESUME_DELAY_MS = 1500;

test('login via /authorize never renders the signed-in dashboard before redirecting (#921)', async ({
  page,
  request,
  baseURL,
}) => {
  const email = `flash-${crypto.randomUUID()}@example.com`;
  await graphql(request, baseURL!, `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`, {
    params: { email, password: PASSWORD, confirm_password: PASSWORD },
  });
  await verifyUserEmail(await getUserIdByEmail(email, baseURL!), { givenName: 'Flash', familyName: 'Probe' }, baseURL!);

  const flashes: string[] = [];
  await page.exposeFunction('__e2eReportDashboard', (href: string) => flashes.push(href));
  await page.addInitScript((text: string) => {
    let reported = false;
    const check = () => {
      if (!reported && document.body && (document.body.textContent || '').includes(text)) {
        reported = true;
        (window as unknown as { __e2eReportDashboard: (h: string) => void }).__e2eReportDashboard(location.href);
      }
    };
    new MutationObserver(check).observe(document, { subtree: true, childList: true, characterData: true });
  }, DASHBOARD_TEXT);
  await stubCallback(page, baseURL!, CALLBACK);

  const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}`);
  await page.goto(req.url);
  await page.waitForURL((u) => /^\/app\/?$/.test(u.pathname));
  await expect(page.locator('#authorizer-login-password')).toBeVisible();

  // Only the resume request is delayed: the initial /authorize already ran.
  let delayed = 0;
  const isAuthorize = (u: URL) => u.pathname === '/authorize';
  await page.route(
    isAuthorize,
    async (route) => {
      delayed++;
      await new Promise((r) => setTimeout(r, RESUME_DELAY_MS));
      await route.continue();
    }
  );

  await page.locator('#authorizer-login-email-or-phone-number').fill(email);
  await page.locator('#authorizer-login-password').fill(PASSWORD);
  await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();

  await page.waitForURL((u) => u.pathname === CALLBACK && u.searchParams.has('code'), { timeout: 15_000 });
  expect(new URL(page.url()).searchParams.get('state')).toBe(req.state);
  expect(delayed, 'the resume went through the delayed /authorize').toBeGreaterThan(0);
  expect(flashes, 'signed-in dashboard rendered before the authorize redirect').toEqual([]);

  // Control: the detector does fire on a genuine dashboard render, so the
  // empty result above is not a blind observer.
  await page.unroute(isAuthorize);
  await page.goto(`${baseURL}/app`);
  await expect(page.getByText(DASHBOARD_TEXT)).toBeVisible();
  await expect.poll(() => flashes.length).toBe(1);
});
