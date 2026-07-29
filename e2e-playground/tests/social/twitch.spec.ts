// e2e-playground/tests/social/twitch.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Twitch', () => {
  test('first-time signup via Twitch creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `twitch-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'twitch',
      buttonName: /twitch/i,
      // Twitch is one of the 4 "OIDC-verified" mock-oauth providers: this
      // profile is signed into a real id_token (server.ts), and
      // processTwitchUserInfo (internal/http_handlers/oauth_callback.go) does
      // idToken.Claims(&user) directly off it - identical mechanism to
      // processMicrosoftUserInfo, so given_name/family_name map through the
      // same way despite real Twitch's OIDC token not normally carrying them.
      profile: { sub: `twitch-${crypto.randomUUID()}`, email, given_name: 'Sally', family_name: 'Ride' },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves email mapping + a
    // real session; this proves the rest of the id_token claims (given_name/
    // family_name) actually landed on the stored user, and that "twitch"
    // was recorded as the signup method - not just that login "looked"
    // successful.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Sally');
    expect(user.family_name).toBe('Ride');
    expect(user.signup_methods).toContain('twitch');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'twitch');
  });
});
