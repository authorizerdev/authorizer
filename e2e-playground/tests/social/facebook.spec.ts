// e2e-playground/tests/social/facebook.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Facebook', () => {
  test('first-time signup via Facebook creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `facebook-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'facebook',
      buttonName: /facebook/i,
      // Facebook is a REST-profile provider (not OIDC-verified): mock-oauth's
      // /facebook/userinfo route returns this JSON verbatim, and
      // processFacebookUserInfo (internal/http_handlers/oauth_callback.go)
      // reads first_name/last_name straight into GivenName/FamilyName (no
      // name-splitting like GitHub) and picture.data.url into Picture.
      profile: {
        first_name: 'Katherine',
        last_name: 'Johnson',
        email,
        picture: { data: { url: 'https://example.com/a.png' } },
      },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves a real session; this
    // proves first_name/last_name actually landed on the stored user
    // unmodified, and that "facebook" was recorded as the signup method.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Katherine');
    expect(user.family_name).toBe('Johnson');
    expect(user.signup_methods).toContain('facebook');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'facebook');
  });
});
