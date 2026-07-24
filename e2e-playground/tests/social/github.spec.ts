// e2e-playground/tests/social/github.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — GitHub', () => {
  test('first-time signup via GitHub creates an account with mapped profile fields', async ({ page, request }) => {
    const email = `github-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'github',
      buttonName: /github/i,
      // GitHub is a REST-profile provider (not OIDC-verified): mock-oauth's
      // /github/userinfo route returns this JSON verbatim, and
      // processGithubUserInfo (internal/http_handlers/oauth_callback.go)
      // decodes it into map[string]string, then splits `name` on the first
      // space into given_name/family_name - keep every field a plain string.
      profile: { name: 'Grace Hopper', email, avatar_url: 'https://example.com/a.png' },
      expectedEmail: email,
    });

    // The dashboard assertion inside the helper proves a real session; this
    // proves the given_name/family_name split actually landed on the stored
    // user, and that "github" was recorded as the signup method.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Grace');
    expect(user.family_name).toBe('Hopper');
    expect(user.signup_methods).toContain('github');
  });

  test('empty primary email falls back to the /user/emails endpoint', async ({ page, request }) => {
    // Real GitHub behavior: profile.email is "" when the user has their
    // email set to private, and processGithubUserInfo falls back to
    // GET /user/emails for the primary address. mock-oauth's
    // /:provider/user/emails handler (server.ts) ignores whatever email was
    // configured and always serves a hardcoded fallback address, so
    // configuring email: '' here exercises the real fallback code path
    // deterministically rather than the happy-path email-passthrough above.
    const fallbackEmail = 'mock-user@github.example.com';
    await runSocialLoginHappyPath(page, request, {
      provider: 'github',
      buttonName: /github/i,
      profile: { name: 'Grace Hopper', email: '', avatar_url: '' },
      expectedEmail: fallbackEmail,
    });

    // Proves the fallback /user/emails roundtrip actually happened - the
    // stored user has the fallback address, not an empty one, and the name
    // split still landed correctly off the same /userinfo response.
    const user = await getUserByEmail(fallbackEmail);
    expect(user.email).toBe(fallbackEmail);
    expect(user.given_name).toBe('Grace');
    expect(user.family_name).toBe('Hopper');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'github');
  });
});
