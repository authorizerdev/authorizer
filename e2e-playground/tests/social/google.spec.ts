// e2e-playground/tests/social/google.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import {
  runSocialLoginHappyPath,
  runConsentDeniedNegativePath,
  runSocialLoginExpectingRejection,
} from './helpers';
import { getUserByEmail, signupUser } from '../../fixtures/adminClient';

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
      profile: { sub: `google-${crypto.randomUUID()}`, email, email_verified: true, given_name: 'Ada', family_name: 'Lovelace' },
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

// --- Email-attestation contract (nOAuth defence, AUDIT-01/AUDIT-02) ---------
//
// These live in this provider's own spec file on purpose. mock-oauth stores ONE
// profile per provider globally, so two spec FILES driving the same provider
// race under parallel workers. One provider per file is the convention that
// keeps the suite order-independent; see docs/email-verification-contract.md
// for what the contract itself says.

test.describe('Social login — Google — email-attestation contract', () => {
  test('email_verified true is imported and the account is created verified', async ({ page, request }) => {
    const email = `evc-google-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'google',
      buttonName: /google/i,
      profile: {
        sub: `google-${crypto.randomUUID()}`,
        email,
        email_verified: true,
        given_name: 'Ada',
        family_name: 'Lovelace',
      },
      expectedEmail: email,
    });

    const user = await getUserByEmail(email);
    expect(user.signup_methods).toContain('google');
    // Auth0 parity: a provider that vouches means no separate verification
    // round-trip — the account is verified from the moment it is created.
    expect(user.email_verified).toBe(true);
  });

  test('an explicit email_verified:false is refused just like an absent claim', async ({
    request,
    baseURL,
  }) => {
    const email = `evc-explicit-false-${crypto.randomUUID()}@example.com`;
    const { status, body } = await runSocialLoginExpectingRejection(request, baseURL!, {
      provider: 'google',
      profile: {
        sub: `google-${crypto.randomUUID()}`,
        email,
        email_verified: false,
        given_name: 'Grace',
        family_name: 'Hopper',
      },
    });

    expect(status).toBe(400);
    expect(body.error).toBe('email_not_verified');
    // The refusal happens before any local lookup, so no account exists.
    await expect(getUserByEmail(email)).rejects.toThrow(/user not found/);
  });

  test('an attested federated email may still link to an existing verified account', async ({
    page,
    request,
  }) => {
    // The control for the nOAuth refusal in microsoft.spec.ts: the guard must
    // block the attack without breaking legitimate linking, or it is an outage.
    const email = `evc-link-${crypto.randomUUID()}@example.com`;
    await signupUser(email, 'Password@123');
    const before = await getUserByEmail(email);
    expect(before.signup_methods).toContain('basic_auth');

    await runSocialLoginHappyPath(page, request, {
      provider: 'google',
      buttonName: /google/i,
      profile: {
        sub: `google-${crypto.randomUUID()}`,
        email,
        email_verified: true,
        given_name: 'Ada',
        family_name: 'Lovelace',
      },
      expectedEmail: email,
    });

    const after = await getUserByEmail(email);
    expect(after.id).toBe(before.id);
    expect(after.signup_methods).toContain('basic_auth');
    expect(after.signup_methods).toContain('google');
  });
});
