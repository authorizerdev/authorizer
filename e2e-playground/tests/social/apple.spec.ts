// e2e-playground/tests/social/apple.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath, configureProviderProfile } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Apple', () => {
  test('first-time signup via Apple with a private relay email creates an account with mapped profile fields', async ({
    page,
    request,
  }) => {
    const email = `relay-${crypto.randomUUID()}@privaterelay.appleid.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'apple',
      buttonName: /apple/i,
      // Apple is one of the 4 "OIDC-verified" mock-oauth providers, but
      // unlike Google/Microsoft/Twitch, given_name/family_name do NOT come
      // off the id_token for Apple - processAppleUserInfo (internal/
      // http_handlers/oauth_callback.go) only ever reads them from the
      // Apple-specific `user` form field (AppleUserInfo.Name.FirstName/
      // LastName), which real Apple's hosted consent page constructs
      // client-side and POSTs alongside the code on first authorization -
      // it is not part of the id_token and mock-oauth's authorize/token
      // exchange doesn't build one by default. mock-oauth's /apple/authorize
      // handler (server.ts) mirrors that real behavior by attaching a `user`
      // query param built from this profile's given_name/family_name, which
      // Authorizer's ctx.Request.FormValue("user") resolves identically to a
      // POST body. Email still comes off the id_token, matching Apple's real
      // private relay pattern.
      profile: { sub: `apple-${crypto.randomUUID()}`, email, given_name: 'Alan', family_name: 'Turing' },
      expectedEmail: email,
    });

    // As with google.spec.ts, the dashboard assertion inside the helper
    // proves email mapping + a real session; this proves given_name/
    // family_name actually landed on the stored user via the `user` field
    // path (not the id_token), and that "apple" was recorded as the signup
    // method.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Alan');
    expect(user.family_name).toBe('Turing');
    expect(user.signup_methods).toContain('apple');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'apple');
  });

  // REGRESSION (production bug): real Apple sends the `user` field
  // (given/family name) only on an account's very first authorization -
  // every later login omits it entirely (a one-time grant, not re-sent).
  // Before the fix, Authorizer's OAuthCallbackHandler (processAppleUserInfo,
  // internal/http_handlers/oauth_callback.go) unconditionally
  // json.Unmarshal'd the `user` field, so its absence 400'd the whole
  // callback - rejecting every returning Apple user outright. This drives a
  // real first signup, then a real second login for the same account with
  // mock-oauth configured to omit the `user` field (via `omit_user_field`,
  // server.ts's /apple/authorize handler), and proves the second login still
  // establishes a session.
  test('returning Apple user (no `user` field on second login) signs in successfully', async ({ page, request }) => {
    const email = `relay-${crypto.randomUUID()}@privaterelay.appleid.com`;

    await runSocialLoginHappyPath(page, request, {
      provider: 'apple',
      buttonName: /apple/i,
      profile: { sub: `apple-${crypto.randomUUID()}`, email, given_name: 'Grace', family_name: 'Hopper' },
      expectedEmail: email,
    });

    // Log out and reconfigure mock-oauth to omit the `user` field, then log
    // back in as the same Apple account - the returning-user path.
    await page.getByRole('button', { name: 'Logout' }).click();
    await configureProviderProfile(request, 'apple', {
      sub: `apple-${crypto.randomUUID()}`,
      email,
      omit_user_field: true,
    });
    await page.getByRole('button', { name: /apple/i }).click();
    await page.waitForURL((url) => url.pathname === '/app' || url.pathname === '/app/', { timeout: 15_000 });
    await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });
    await expect(page.locator(`a[href="mailto:${email}"]`)).toBeVisible();
  });
});
