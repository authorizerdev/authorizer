// e2e-playground/tests/social/microsoft.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Microsoft', () => {
  test('first-time signup via Microsoft creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `microsoft-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'microsoft',
      buttonName: /microsoft/i,
      // Microsoft is one of the 4 "OIDC-verified" mock-oauth providers: this
      // profile is signed into a real id_token (server.ts), and
      // processMicrosoftUserInfo (internal/http_handlers/oauth_callback.go)
      // reads given_name/family_name/email/sub straight off its claims.
      profile: { sub: `microsoft-${crypto.randomUUID()}`, email, given_name: 'Katherine', family_name: 'Johnson' },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves email mapping + a
    // real session; this proves the rest of the id_token claims (given_name/
    // family_name) actually landed on the stored user, and that "microsoft"
    // was recorded as the signup method - not just that login "looked"
    // successful.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Katherine');
    expect(user.family_name).toBe('Johnson');
    expect(user.signup_methods).toContain('microsoft');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'microsoft');
  });
});
