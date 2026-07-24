// e2e-playground/tests/social/discord.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { runSocialLoginHappyPath, runConsentDeniedNegativePath } from './helpers';
import { getUserByEmail } from '../../fixtures/adminClient';

test.describe('Social login — Discord', () => {
  test('first-time signup via Discord creates an account with mapped profile fields', async ({ page, request }) => {
    // defaultDiscordScopes (cmd/root.go) already requests `identify email`,
    // and processDiscordUserInfo (internal/http_handlers/oauth_callback.go)
    // now reads GET /users/@me's flat id/username/avatar/email fields
    // directly - a real, deliverable email, unlike Twitter/X's synthetic
    // fallback. mock-oauth's /discord/userinfo route (server.ts) returns
    // this profile back verbatim.
    const discordId = `discord-1-${crypto.randomUUID()}`;
    const email = `discord-user-${crypto.randomUUID()}@example.com`;
    await runSocialLoginHappyPath(page, request, {
      provider: 'discord',
      buttonName: /discord/i,
      profile: { id: discordId, username: 'gracehopper', avatar: 'abc123', email },
      expectedEmail: email,
    });

    // processDiscordUserInfo maps `username` straight into GivenName (no
    // name-splitting, no FamilyName) and builds Picture from
    // cdn.discordapp.com/avatars/<id>/<avatar>.png.
    const user = await getUserByEmail(email);
    expect(user.given_name).toBe('gracehopper');
    expect(user.signup_methods).toContain('discord');
  });

  test('repeat login with the same Discord identity resolves to the same account, not a duplicate', async ({
    browser,
    page,
    request,
    baseURL,
  }) => {
    // Regression test for the real production bug fixed in
    // processDiscordUserInfo: it used to call GET /oauth2/@me (Discord's
    // "current authorization" endpoint), whose `user` sub-object never
    // includes `email` regardless of granted scopes, and never read an
    // email field even from what it did get - so user.Email was always nil.
    // OAuthCallbackHandler's signup-vs-login check (GetUserByEmail(ctx,
    // refs.StringValue(user.Email))) then always ran as
    // GetUserByEmail(ctx, ""), which never matches a NULL email column -
    // every Discord login created a brand-new duplicate account. The fix
    // switches to GET /users/@me (which does return a real email once the
    // user grants the `email` scope, already Authorizer's default) and sets
    // it as user.Email, so the normal GetUserByEmail path now works
    // correctly with no synthetic-email machinery needed (unlike Twitter,
    // which never gets a real email at all).
    const email = `discord-repeat-${crypto.randomUUID()}@example.com`;
    const profile = { id: `discord-stable-${crypto.randomUUID()}`, username: 'gracehopper', avatar: 'def456', email };

    // First login (fresh browser context = `page`/`request` from the test
    // fixture): creates the account.
    await runSocialLoginHappyPath(page, request, {
      provider: 'discord',
      buttonName: /discord/i,
      profile,
      expectedEmail: email,
    });
    const firstLoginUser = await getUserByEmail(email);

    // Second login, from a brand-new, cookie-less browser context (two
    // independent people sitting down and logging into Discord, not one
    // continuous session), against the SAME mock-oauth Discord profile/id.
    // Before the fix this would create a second, orphaned account; after
    // the fix it must recognize the same account via GetUserByEmail.
    const context2 = await browser.newContext({ baseURL });
    const page2 = await context2.newPage();
    try {
      await runSocialLoginHappyPath(page2, context2.request, {
        provider: 'discord',
        buttonName: /discord/i,
        profile,
        expectedEmail: email,
      });
    } finally {
      await context2.close();
    }

    const secondLoginUser = await getUserByEmail(email);
    expect(secondLoginUser.id).toBe(firstLoginUser.id);
  });

  test('consent denied at provider is rejected without a session, and the state cannot be replayed', async ({
    request,
    baseURL,
  }) => {
    await runConsentDeniedNegativePath(request, baseURL!, 'discord');
  });
});
