// e2e-playground/tests/social/microsoft.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import {
  runSocialLoginHappyPath,
  runConsentDeniedNegativePath,
  runSocialLoginExpectingRejection,
} from './helpers';
import { getUserByEmail, signupUser } from '../../fixtures/adminClient';

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
      profile: { sub: `microsoft-${crypto.randomUUID()}`, email, email_verified: true, given_name: 'Katherine', family_name: 'Johnson' },
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

// --- Email-attestation contract (nOAuth defence, AUDIT-01/AUDIT-02) ---------
//
// These live in this provider's own spec file on purpose. mock-oauth stores ONE
// profile per provider globally, so two spec FILES driving the same provider
// race under parallel workers. One provider per file is the convention that
// keeps the suite order-independent; see docs/email-verification-contract.md
// for what the contract itself says.

test.describe('Social login — Microsoft — enterprise directories do not attest', () => {
  test('an id_token with no email attestation is refused and creates no account', async ({
    request,
    baseURL,
  }) => {
    const email = `evc-entra-${crypto.randomUUID()}@example.com`;
    // A real Entra v2 id_token: no email_verified, no xms_edov. This is exactly
    // what the multi-tenant "common" endpoint hands back.
    const { status, body } = await runSocialLoginExpectingRejection(request, baseURL!, {
      provider: 'microsoft',
      profile: {
        sub: `microsoft-${crypto.randomUUID()}`,
        email,
        given_name: 'Katherine',
        family_name: 'Johnson',
      },
    });

    expect(status).toBe(400);
    expect(body.error).toBe('email_not_verified');
    await expect(getUserByEmail(email)).rejects.toThrow(/user not found/);
  });

  test('an unattested federated email cannot take over an existing verified account', async ({
    request,
    baseURL,
  }) => {
    // The victim: an ordinary password account, email already verified.
    const victimEmail = `evc-victim-${crypto.randomUUID()}@example.com`;
    await signupUser(victimEmail, 'Password@123');
    const before = await getUserByEmail(victimEmail);
    expect(before.email_verified).toBe(true);
    expect(before.signup_methods).toContain('basic_auth');
    expect(before.signup_methods).not.toContain('microsoft');

    // The attack: a tenant the operator does not control asserts the victim's
    // address with no attestation behind it. The pre-hijack guard in
    // oauth_callback.go does NOT cover this — it only removes *unverified*
    // local accounts, and a verified account is precisely the target.
    const { status, body } = await runSocialLoginExpectingRejection(request, baseURL!, {
      provider: 'microsoft',
      profile: {
        sub: `microsoft-attacker-${crypto.randomUUID()}`,
        email: victimEmail,
        given_name: 'Not',
        family_name: 'Katherine',
      },
    });

    expect(status).toBe(400);
    expect(body.error).toBe('email_not_verified');

    // The victim's account must be untouched: no microsoft signup method
    // grafted on, no profile fields overwritten, still verified.
    const after = await getUserByEmail(victimEmail);
    expect(after.id).toBe(before.id);
    expect(after.signup_methods).toBe(before.signup_methods);
    expect(after.signup_methods).not.toContain('microsoft');
    expect(after.given_name).toBe(before.given_name);
    expect(after.email_verified).toBe(true);
  });
});
