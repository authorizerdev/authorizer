// e2e-playground/tests/social/google.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath } from './helpers';
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
    // Mirrors what Google actually sends when a user denies consent: the
    // browser is redirected back with `error=access_denied` and no `code`,
    // carrying the exact `state` this Authorizer instance issued for the
    // attempt (internal/http_handlers/oauth_login.go). Get a real one by
    // driving the real login-initiation endpoint directly (same route the
    // rendered "Continue with Google" button hits) rather than guessing its
    // format - the compound state string is generated server-side.
    const redirectUri = `${baseURL}/app`;
    const loginRes = await request.get(`/oauth_login/google?redirect_uri=${encodeURIComponent(redirectUri)}`, {
      maxRedirects: 0,
    });
    // internal/http_handlers/oauth_login.go issues http.StatusTemporaryRedirect.
    expect(loginRes.status()).toBe(307);
    const authorizeLocation = loginRes.headers()['location'];
    expect(authorizeLocation).toBeTruthy();
    const state = new URL(authorizeLocation!).searchParams.get('state');
    expect(state).toBeTruthy();

    // Real behavior (internal/http_handlers/oauth_callback.go
    // OAuthCallbackHandler): a recognized state with no `code` in the request
    // is rejected before any provider call is made, so no user is ever
    // looked up or created - there is no "partial account" state to leave
    // behind here by construction, not just by assertion.
    const deniedRes = await request.get(
      `/oauth_callback/google?error=access_denied&state=${encodeURIComponent(state!)}`,
      { maxRedirects: 0 }
    );
    expect(deniedRes.status()).toBe(400);
    const deniedBody = await deniedRes.json();
    expect(deniedBody.error).toBe('invalid oauth code');

    // The callback handler removes the state from the store as soon as it's
    // read, regardless of outcome (single-use, same RFC 6749 §4.1.2
    // discipline as authorization codes - see the replay test in
    // tests/oidc-provider.spec.ts) - a retried request with the exact same
    // state must fail differently: unrecognized state, not "no code" again.
    const replayRes = await request.get(
      `/oauth_callback/google?error=access_denied&state=${encodeURIComponent(state!)}`,
      { maxRedirects: 0 }
    );
    expect(replayRes.status()).toBe(400);
    const replayBody = await replayRes.json();
    expect(replayBody.error).toBe('invalid oauth state');
  });
});
