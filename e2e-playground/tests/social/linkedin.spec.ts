// e2e-playground/tests/social/linkedin.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — LinkedIn', () => {
  test('first-time signup via LinkedIn creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `linkedin-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'linkedin',
      buttonName: /linkedin/i,
      // OIDC userinfo shape (api.linkedin.com/v2/userinfo), which replaced the
      // legacy /v2/me + /v2/emailAddress pair. mock-oauth's /linkedin/userinfo
      // route returns this JSON verbatim and processLinkedInUserInfo
      // (internal/http_handlers/oauth_callback.go) reads given_name/family_name
      // straight into GivenName/FamilyName (no name-splitting like GitHub).
      //
      // `email` arrives in THIS payload, not a second call. It is documented as
      // optional — present only when the member granted the `email` scope — and
      // the handler treats its absence as a hard error rather than synthesizing
      // one, because LinkedIn's `sub` is pairwise per-app and is therefore
      // useless as an identity key.
      //
      // Keep this in the provider's real shape: __configure REPLACES the mock's
      // default profile wholesale, so a stale field name here silently sends a
      // payload the handler cannot read, and the assertions below fail with an
      // empty string rather than anything that names the cause.
      profile: {
        sub: 'mock-linkedin-sub',
        name: 'Margaret Hamilton',
        given_name: 'Margaret',
        family_name: 'Hamilton',
        picture: 'https://example.com/a.png',
        email,
        email_verified: true,
      },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves a real session; this
    // proves given_name/family_name actually landed on the stored user, the
    // email in the userinfo payload resolved to the right address, and
    // "linkedin" was recorded as the signup method.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Margaret');
    expect(user.family_name).toBe('Hamilton');
    expect(user.signup_methods).toContain('linkedin');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'linkedin');
  });
});
