// e2e-playground/tests/webauthn.spec.ts
import { test, expect } from '@playwright/test';
import { GraphQLClient, gql } from 'graphql-request';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

// This spec runs against authorizer-webauthn (docker-compose.yml), NOT the
// shared `authorizer`/`authorizer-sso` instances every other spec uses - see
// that service's comment for why (go-webauthn's RPID validation rejects
// their single-label hostnames outright). So it can't reuse
// fixtures/adminClient.ts, which is hardcoded to AUTHORIZER_BASE_URL.
const BASE_URL = process.env.AUTHORIZER_WEBAUTHN_BASE_URL || 'http://localhost:8082';
// CSRF middleware requires Origin/Referer on state-changing requests (see
// internal/http_handlers/csrf.go) - same rationale as fixtures/adminClient.ts.
const client = new GraphQLClient(`${BASE_URL}/graphql`, { headers: { Origin: BASE_URL } });

async function signupUser(email: string, password: string): Promise<void> {
  const query = gql`
    mutation ($params: SignUpRequest!) {
      signup(params: $params) { message }
    }
  `;
  await client.request(query, { params: { email, password, confirm_password: password } });
}

function randomEmail() {
  return `webauthn-${crypto.randomUUID()}@example.com`;
}

// Resolves the full `chrome` binary Playwright's own installer already
// downloaded, without hardcoding its revision-numbered directory (e.g.
// chromium-1228 - tied to this exact Playwright version, would silently
// break on the next `npx playwright install` bump). Deliberately excludes
// "chromium_headless_shell-*" - that's the OTHER binary Playwright installs,
// and the one that doesn't work here (see comment below).
function findFullChromePath(): string | undefined {
  const browsersDir = process.env.PLAYWRIGHT_BROWSERS_PATH || '/ms-playwright';
  const dir = fs
    .readdirSync(browsersDir, { withFileTypes: true })
    .find((e) => e.isDirectory() && e.name.startsWith('chromium-'));
  if (!dir) return undefined;
  return path.join(browsersDir, dir.name, 'chrome-linux', 'chrome');
}

// Two more real, reproducible infra gaps hit on the first run against this
// origin (on top of the dedicated authorizer-webauthn service above), both
// fixed here (must be top-level, not inside describe() - launchOptions is a
// worker-scoped option):
//
// 1. authorizer-js's isWebauthnSupported() gates on window.PublicKeyCredential
//    existing at all, which browsers only expose in a secure context (https,
//    or the special-cased "localhost"). The containerized playwright run
//    navigates to a docker-internal hostname over plain http, which is
//    neither - so without the unsafely-treat-insecure-origin-as-secure flag
//    the Passkey "Set up" button in Settings stays permanently disabled.
// 2. Playwright's default `headless: true` launches the stripped-down
//    "headless_shell" binary (confirmed via `DEBUG=pw:browser`), which does
//    NOT honor that flag (window.isSecureContext stays false even with it
//    set) - only the full `chrome` binary does. executablePath forces that
//    full binary; --headless is still passed by Playwright's defaults, so
//    this stays headless.
test.use({
  launchOptions: {
    executablePath: findFullChromePath(),
    args: [`--unsafely-treat-insecure-origin-as-secure=${BASE_URL}`],
  },
});

test.describe('WebAuthn / Passkey', () => {
  test('register a passkey in Settings, then log back in with only the passkey', async ({ browser }) => {
    // Real WebAuthn ceremony via Playwright's CDP-backed virtual authenticator
    // (no mock server involved, unlike the social-login specs) - this drives
    // the actual browser WebAuthn JSON APIs the SDK/authorizer-js call.
    const context = await browser.newContext();
    const page = await context.newPage();
    const cdp = await context.newCDPSession(page);
    await cdp.send('WebAuthn.enable');
    const { authenticatorId } = await cdp.send('WebAuthn.addVirtualAuthenticator', {
      options: {
        protocol: 'ctap2',
        transport: 'internal',
        hasResidentKey: true,
        hasUserVerification: true,
        isUserVerified: true,
      },
    });

    const email = randomEmail();
    const password = 'Str0ngPassw0rd!';
    // Created directly via the admin/public GraphQL API (not the signup UI
    // form) - this spec is testing the WebAuthn ceremony, not the signup
    // form, same rationale as tests/oidc-provider.spec.ts and
    // tests/saml-idp.spec.ts.
    await signupUser(email, password);

    // Real rendered /app login form (web/app/src/pages/login.tsx ->
    // AuthorizerBasicAuthLogin in authorizer-react), same locators as
    // tests/oidc-provider.spec.ts's PKCE test.
    await page.goto('/app');
    await page.locator('#authorizer-login-email-or-phone-number').fill(email);
    await page.locator('#authorizer-login-password').fill(password);
    await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();

    // Brand-new user, first login: withheld-token MFA offer screen (commit
    // 992b3bf4). "Skip for now" issues the token without enrolling any
    // factor yet - the passkey gets registered afterwards, from Settings.
    await page
      .getByRole('button', { name: 'Skip for now' })
      .click({ timeout: 10_000 })
      .catch(() => {});

    // web/app/src/pages/dashboard.tsx only renders "Signed in as <email>"
    // once useAuthorizer's token is actually set - a real proof of session,
    // not a "page didn't say error" tautology.
    await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });

    // web/app/src/pages/dashboard.tsx:30 - the link text is "Manage MFA" in
    // this repo's wrapper (NOT "Manage sign-in methods" - that's a different
    // consumer app in authorizer-react's own examples).
    await page.getByRole('link', { name: 'Manage MFA' }).click();
    await expect(page.getByRole('heading', { name: 'Multi-factor authentication' })).toBeVisible();

    // Settings list -> Passkey row -> "Set up" (AuthorizerMFASetup renders
    // "Set up" when passkeyRegistered is false, "Manage" once true - verified
    // against node_modules/@authorizerdev/authorizer-react/src/components/
    // AuthorizerMFASetup.tsx). This matches the brief's guessed locator.
    await page
      .getByRole('listitem')
      .filter({ hasText: 'Passkey' })
      .getByRole('button', { name: 'Set up' })
      .click();
    // AuthorizerPasskeyRegister.tsx's button text, confirmed from source.
    await page.getByRole('button', { name: 'Add a passkey' }).click();

    // AuthorizerMFASetup has no stable success message on this path: its
    // onSuccess callback is `backToList` (source-confirmed), which
    // immediately unmounts AuthorizerPasskeyRegister (and its own transient
    // "Passkey added..." message) and re-renders the top-level method list.
    // So the real success signal is: no error, and we're back on that list.
    // NOTE: the brief's suggested error regex (/failed to verify passkey/i)
    // does not match the real error text - AuthorizerPasskeyRegister.tsx
    // sets "Could not add passkey." on failure, verified from source.
    await expect(page.getByText('Could not add passkey.')).toHaveCount(0);
    await expect(page.getByText('Add a second step to sign in')).toBeVisible({ timeout: 10_000 });

    // Log out (the button only exists on the dashboard, not Settings) and
    // log back in using ONLY the passkey - no password.
    await page.getByRole('link', { name: 'Back to dashboard' }).click();
    await page.getByRole('button', { name: 'Logout' }).click();

    // AuthorizerPasskeyLogin's usernameless/passwordless login button
    // (web/app/src/pages/login.tsx), confirmed from source.
    await page.getByRole('button', { name: 'Sign in with a passkey' }).click();

    // Real session established via the passkey alone - same proof as the
    // initial password login above.
    await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });
    await expect(page.locator(`a[href="mailto:${email}"]`)).toBeVisible();

    await cdp.send('WebAuthn.removeVirtualAuthenticator', { authenticatorId });
    await context.close();
  });
});
