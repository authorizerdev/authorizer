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
      // Sign In with LinkedIn (OIDC): mock-oauth's /linkedin/userinfo route
      // returns this JSON verbatim, and processLinkedInUserInfo
      // (internal/http_handlers/oauth_callback.go) reads
      // given_name/family_name/picture/email straight off it - one call, no
      // separate /v2/emailAddress hop (the legacy /v2/me pair needed
      // r_liteprofile/r_emailaddress, which OIDC-onboarded apps never get).
      // `email` only arrives when the member granted the email scope; without
      // it the handler hard-errors, since LinkedIn's `sub` is pairwise per-app
      // and there is no other identity key.
      profile: {
        sub: 'mock-linkedin-sub',
        given_name: 'Margaret',
        family_name: 'Hamilton',
        picture: 'https://example.com/a.png',
        email,
        email_verified: true,
      },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves a real session; this
    // proves the userinfo given_name/family_name actually landed on the stored
    // user, the userinfo email resolved to the right address, and "linkedin"
    // was recorded as the signup method.
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
