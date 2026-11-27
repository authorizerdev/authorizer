// e2e-playground/tests/magic-link-authorize.spec.ts
//
// Guards #934: a magic link started from /authorize must (a) deliver a code
// that /oauth/token actually redeems, once, but only to the browser that ran
// /authorize (the binding cookie); (b) still sign the user in, without a code,
// anywhere else; (c) never become an open redirect or hand an attacker a code
// for someone else's account; (d) survive a redirect_uri with its own query
// string under exact --redirect-uris matching.
//
// Runs in the `magic-link` project against authorizer-magic-link, which lists
// its exact --redirect-uris (docker-compose.yml) so the exact-match path is
// what is under test, not the origin fallback.
import { test, expect, Browser, BrowserContext, Page } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';
import { waitForEmailLink, VERIFY_EMAIL_LINK } from '../fixtures/mailpit';
import { newAuthorizeRequest, redeemCode, stubCallback, AuthorizeRequest } from '../fixtures/oauth';

const CALLBACK = '/e2e-callback';

function randomEmail() {
  return `ml-authz-${crypto.randomUUID()}@example.com`;
}

const magicLinkLoginMutation = `
  mutation ($params: MagicLinkLoginRequest!) {
    magic_link_login(params: $params) { message }
  }
`;

const isCallback = (url: URL) => url.pathname === CALLBACK;

// startFromAuthorize runs /authorize in `page` (which detours to the login UI
// and sets the browser-binding cookie), returns the /app URL it landed on.
async function startFromAuthorize(page: Page, req: AuthorizeRequest): Promise<URL> {
  await page.goto(req.url);
  await page.waitForURL((u) => /^\/app\/?$/.test(u.pathname));
  return new URL(page.url());
}

// requestLinkThroughLoginUI submits the rendered magic-link form, exactly as a
// user would on the page /authorize sent them to.
async function requestLinkThroughLoginUI(page: Page, email: string) {
  const input = page.locator('#authorizer-magic-link-login-email');
  await input.fill(email);
  const sent = page.waitForResponse((r) => r.url().endsWith('/graphql') && r.request().postData()?.includes('magic_link_login') === true);
  await page.locator('form[name="authorizer-magic-login-form"] button[type="submit"]').click();
  expect((await sent).ok()).toBe(true);
}

async function freshContext(browser: Browser, baseURL: string): Promise<{ ctx: BrowserContext; page: Page }> {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await stubCallback(page, baseURL, CALLBACK);
  return { ctx, page };
}

test.describe('Magic link from /authorize (#934)', () => {
  test('same browser: redirect carries a code that redeems exactly once', async ({ page, context, request, baseURL }) => {
    await stubCallback(page, baseURL!, CALLBACK);
    const email = randomEmail();
    const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}`);

    await startFromAuthorize(page, req);
    await requestLinkThroughLoginUI(page, email);
    const link = await waitForEmailLink(request, email, VERIFY_EMAIL_LINK);

    // A second tab in the SAME context — how a user opens the mail — so it
    // carries the binding cookie /authorize set.
    const tab = await context.newPage();
    await tab.goto(link);
    await tab.waitForURL((u) => isCallback(u) && u.searchParams.has('code'));
    const cb = new URL(tab.url());
    expect(cb.searchParams.get('state')).toBe(req.state);
    for (const p of ['access_token', 'id_token', 'refresh_token', 'token']) {
      expect(cb.searchParams.has(p), `${p} must not be in the redirect`).toBe(false);
    }

    const code = cb.searchParams.get('code')!;
    const first = await redeemCode(request, baseURL!, code, req);
    expect(first.status(), await first.text()).toBe(200);
    const body = await first.json();
    expect(body.access_token).toBeTruthy();
    expect(body.id_token).toBeTruthy();

    const replay = await redeemCode(request, baseURL!, code, req);
    expect(replay.status()).toBe(400);
    expect((await replay.json()).error).toBe('invalid_grant');
  });

  test('different browser (no binding cookie): user is signed in but gets no code', async ({
    page,
    browser,
    request,
    baseURL,
  }) => {
    await stubCallback(page, baseURL!, CALLBACK);
    const email = randomEmail();
    const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}`);
    await startFromAuthorize(page, req);
    await requestLinkThroughLoginUI(page, email);
    const link = await waitForEmailLink(request, email, VERIFY_EMAIL_LINK);

    const other = await freshContext(browser, baseURL!);
    try {
      await other.page.goto(link);
      await other.page.waitForURL(isCallback);
      const cb = new URL(other.page.url());
      expect(cb.searchParams.has('code'), 'no code without the binding cookie').toBe(false);
      expect(cb.searchParams.has('error')).toBe(false);

      // ...but the click did establish a session in that browser.
      await other.page.goto(`${baseURL}/app`);
      await expect(other.page.getByText('Signed in as')).toBeVisible();
      await expect(other.page.getByRole('link', { name: email })).toBeVisible();
    } finally {
      await other.ctx.close();
    }
  });

  test('attacker state + victim email: never redirected to an unlisted URI, never a code', async ({
    browser,
    baseURL,
  }) => {
    const attacker = await freshContext(browser, baseURL!);
    const victim = await freshContext(browser, baseURL!);
    try {
      // Requests to the attacker's landing page, from either browser.
      const attackerHits: string[] = [];
      for (const ctx of [attacker.ctx, victim.ctx]) {
        ctx.on('request', (r) => {
          if (new URL(r.url()).pathname === '/attacker-cb') attackerHits.push(r.url());
        });
      }

      // 1. Unlisted redirect on an allowed origin. magic_link_login checks the
      //    redirect only against --allowed-origins, so it mails the link;
      //    /verify_email must hold it to the exact --redirect-uris list,
      //    because the state's client widening only applies to the bound
      //    browser and the victim is not it.
      {
        const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}`);
        await startFromAuthorize(attacker.page, req);
        const victimEmail = randomEmail();
        await graphql(attacker.ctx.request, baseURL!, magicLinkLoginMutation, {
          params: { email: victimEmail, state: req.state, redirect_uri: `${baseURL}/attacker-cb` },
        });
        const link = await waitForEmailLink(attacker.ctx.request, victimEmail, VERIFY_EMAIL_LINK);

        const res = await victim.page.goto(link);
        expect(res!.status()).toBe(400);
        await expect(victim.page.locator('body')).toContainText('invalid redirect uri');
        expect(new URL(victim.page.url()).pathname).toBe('/verify_email');
      }

      // 2. Listed redirect with the attacker's state: the victim signs in to
      //    their own account, the redirect carries no code, and the code
      //    /authorize pre-generated for the attacker (visible in the attacker's
      //    own /app URL) is not redeemable with the attacker's verifier.
      {
        const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}`);
        const appUrl = await startFromAuthorize(attacker.page, req);
        const pendingCode = appUrl.searchParams.get('code');
        const victimEmail = randomEmail();
        await graphql(attacker.ctx.request, baseURL!, magicLinkLoginMutation, {
          params: { email: victimEmail, state: req.state, redirect_uri: req.redirectUri },
        });
        const link = await waitForEmailLink(attacker.ctx.request, victimEmail, VERIFY_EMAIL_LINK);

        await victim.page.goto(link);
        await victim.page.waitForURL(isCallback);
        expect(new URL(victim.page.url()).searchParams.has('code')).toBe(false);

        // /authorize always threads the pre-generated code into the login
        // UI URL for the code flow, so this check can never be skipped.
        expect(pendingCode, 'the /app URL carries the pending code').toBeTruthy();
        const stolen = await redeemCode(attacker.ctx.request, baseURL!, pendingCode!, req);
        expect(stolen.status(), 'attacker must not redeem a code for the victim').toBe(400);
        expect((await stolen.json()).error).toBe('invalid_grant');
      }

      expect(attackerHits, 'no browser may reach the attacker URI').toEqual([]);
    } finally {
      await attacker.ctx.close();
      await victim.ctx.close();
    }
  });

  test('redirect_uri with its own query string survives exact matching', async ({ page, context, request, baseURL }) => {
    await stubCallback(page, baseURL!, CALLBACK);
    const email = randomEmail();
    // Listed verbatim in authorizer-magic-link's --redirect-uris.
    const req = newAuthorizeRequest(baseURL!, `${baseURL}${CALLBACK}?tenant=acme`);

    await startFromAuthorize(page, req);
    await requestLinkThroughLoginUI(page, email);
    const link = await waitForEmailLink(request, email, VERIFY_EMAIL_LINK);
    // The emailed link carries the whole redirect_uri as ONE escaped value.
    expect(new URL(link).searchParams.get('redirect_uri')).toBe(req.redirectUri);

    const tab = await context.newPage();
    await tab.goto(link);
    await tab.waitForURL((u) => isCallback(u) && u.searchParams.has('code'));
    const cb = new URL(tab.url());
    expect(cb.searchParams.get('tenant')).toBe('acme');
    expect(cb.searchParams.get('state')).toBe(req.state);

    const tokenRes = await redeemCode(request, baseURL!, cb.searchParams.get('code')!, req);
    expect(tokenRes.status(), await tokenRes.text()).toBe(200);
  });
});
