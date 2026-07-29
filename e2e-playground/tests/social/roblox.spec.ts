// e2e-playground/tests/social/roblox.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Roblox', () => {
  test('first-time signup via Roblox creates an account with mapped profile fields', async ({ page, request }) => {
    // Roblox is a REST-profile provider (not OIDC-verified, unlike
    // Microsoft/Twitch): mock-oauth's /roblox/userinfo route returns this
    // profile back verbatim, and processRobloxUserInfo (internal/
    // http_handlers/oauth_callback.go) reads it into map[string]interface{}.
    const email = `roblox-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'roblox',
      buttonName: /roblox/i,
      profile: { name: 'Ada Lovelace', nickname: 'ada', picture: 'https://example.com/a.png', email },
      expectedEmail: email,
    });

    // processRobloxUserInfo splits `name` with strings.SplitAfterN(name, " ", 2)
    // - like processTwitterUserInfo, NOT like GitHub's strings.Split - which
    // keeps the separator on the first piece: "Ada Lovelace" becomes
    // given_name "Ada " (trailing space) and family_name "Lovelace".
    // Asserting the literal trailing space proves this is exercising the
    // real Roblox-specific split, not a copy-pasted GitHub-style assertion.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('Ada ');
    expect(user.family_name).toBe('Lovelace');
    expect(user.signup_methods).toContain('roblox');
  });

  // Regression test for the real production bug in processRobloxUserInfo:
  // defaultRobloxScopes (cmd/root.go) is ["openid", "profile"] - no `email` -
  // so real Roblox userinfo (an OIDC-standard endpoint per
  // constants.RobloxUserInfoURL) returns the mandatory `sub` claim but omits
  // `email` under this default config. The mock's /roblox/userinfo route
  // returns whatever profile is configured verbatim with no scope gating of
  // its own, so configuring a profile with `sub` set and no `email` key here
  // reproduces the real no-email-scope condition exactly. Before the fix,
  // processRobloxUserInfo fell into the `else if sub` branch and stored the
  // bare numeric sub directly as user.Email - not email-shaped at all. The
  // fix synthesizes "roblox-<sub>@roblox.oauth.internal" instead, mirroring
  // twitterSyntheticEmail/discordSyntheticEmail.
  test('signup with no email scope granted falls back to a synthetic email, not the raw sub', async ({
    page,
    request,
  }) => {
    const sub = `${Date.now()}${crypto.randomInt(1000, 9999)}`;
    const syntheticEmail = `roblox-${sub}@roblox.oauth.internal`;

    await runSocialLoginHappyPath(page, request, {
      provider: 'roblox',
      buttonName: /roblox/i,
      profile: { sub, name: 'Grace Hopper', nickname: 'grace', picture: 'https://example.com/a.png' },
      expectedEmail: syntheticEmail,
    });

    const user = await getUserByEmail(syntheticEmail);
    expect(user.email).toBe(syntheticEmail);
    // The raw sub must never land directly in the email column.
    expect(user.email).not.toBe(sub);
    expect(user.signup_methods).toContain('roblox');
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'roblox');
  });
});
