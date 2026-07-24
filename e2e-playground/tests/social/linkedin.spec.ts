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
      // LinkedIn is a two-URL REST-profile provider (like GitHub): mock-oauth's
      // /linkedin/userinfo route returns this JSON verbatim, and
      // processLinkedInUserInfo (internal/http_handlers/oauth_callback.go)
      // reads localizedFirstName/localizedLastName straight into
      // GivenName/FamilyName (no name-splitting like GitHub). Unlike GitHub's
      // email fallback, LinkedIn's /userinfo response never carries an email
      // at all - the handler unconditionally fetches mock-oauth's
      // /linkedin/emailAddress route (real LinkedIn's separate email API) and
      // errors out if that lookup fails, so `email` here only ever reaches
      // Authorizer through that second call, not the userinfo payload.
      profile: { localizedFirstName: 'Margaret', localizedLastName: 'Hamilton', email },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves a real session; this
    // proves localizedFirstName/localizedLastName actually landed on the
    // stored user as given_name/family_name, the separate emailAddress call
    // resolved to the right address, and "linkedin" was recorded as the
    // signup method.
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
