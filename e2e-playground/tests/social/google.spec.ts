// e2e-playground/tests/social/google.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Google', () => {
  test('first-time signup via Google creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `google-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'google',
      buttonName: /google/i,
      // Google is one of the 4 "OIDC-verified" mock-oauth providers: this
      // profile is signed into a real id_token (server.ts), and
      // processGoogleUserInfo (internal/http_handlers/oauth_callback.go)
      // reads given_name/family_name/email/sub straight off its claims.
      profile: { sub: `google-${crypto.randomUUID()}`, email, given_name: 'Ada', family_name: 'Lovelace' },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves email mapping + a
    // real session; this proves the rest of the id_token claims (given_name/
    // family_name) actually landed on the stored user, and that "google" was
    // recorded as the signup method - not just that login "looked" successful.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Ada');
    expect(user.family_name).toBe('Lovelace');
    expect(user.signup_methods).toContain('google');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'google');
  });
});
