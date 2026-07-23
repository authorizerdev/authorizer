// e2e-playground/tests/social/apple.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
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
});
