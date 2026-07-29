// e2e-playground/tests/social/helpers.ts
//
// Shared driver for the 10 OAuth2 social-provider specs (this file is
// consumed by tests/social/google.spec.ts and, per the plan, every other
// tests/social/<provider>.spec.ts). Keeps the mock-configure + button-click +
// session-established sequence in one place so each per-provider spec only
// supplies its own provider name, button label, and mock profile shape.
import type { APIRequestContext, Page } from '@playwright/test';
import { expect } from '@playwright/test';

const MOCK_OAUTH_BASE = process.env.MOCK_OAUTH_BASE_URL || 'http://localhost:4000';

// configureProviderProfile sets the profile mock-oauth's /:provider/token and
// /:provider/userinfo (and provider-specific alias) endpoints will return for
// the next login against that provider (e2e-playground/mocks/mock-oauth/server.ts
// __configure handler). Profile shape is provider-specific: the 4 OIDC-verified
// providers (google/apple/microsoft/twitch) get an id_token built from these
// claims directly (sub/email/given_name/family_name...); the 6 REST-profile
// providers (github/facebook/linkedin/discord/twitter/roblox) get it back
// verbatim from their own userinfo/alias route, in whatever shape that
// provider's real API uses (see server.ts's defaultProfile for each shape).
export async function configureProviderProfile(
  request: APIRequestContext,
  provider: string,
  profile: Record<string, unknown>
): Promise<void> {
  const res = await request.post(`${MOCK_OAUTH_BASE}/${provider}/__configure`, { data: { profile } });
  if (res.status() !== 204) {
    throw new Error(`failed to configure mock profile for ${provider}: ${res.status()}`);
  }
}

// dismissMfaSetupOffer handles the "withheld-token first-time setup" screen
// (AuthorizerMFASetup, web/app/src/Root.tsx) that a brand-new user's first
// login lands on: EnableMFA is true by default (TOTP needs no external
// provider - internal/config/config.go Finalize) and this stack doesn't
// disable it, so a fresh signup with no enrolled factor and no prior skip
// hits mfaGateOfferAll (internal/service/mfa_gate.go resolveMFAGate) on
// EVERY social provider, not just Google - shared here rather than
// duplicated per provider spec, same pattern tests/oidc-provider.spec.ts
// uses for the password-login path. click()'s own actionability wait means
// this also tolerates a future world where the screen doesn't appear.
async function dismissMfaSetupOffer(page: Page): Promise<void> {
  await page
    .getByRole('button', { name: 'Skip for now' })
    .click({ timeout: 10_000 })
    .catch(() => {});
}

// runSocialLoginHappyPath drives a real login through the actual rendered
// /app social button (window.location.href = /oauth_login/:provider - see
// authorizer-react's AuthorizerSocialLogin.tsx), through mock-oauth's real
// authorize/token/userinfo roundtrip, and asserts a session was actually
// established: web/app/src/pages/dashboard.tsx only renders "Signed in as
// <email>" once useAuthorizer's token is set, so this is a real proof of a
// completed login (and of email-field mapping), not a "page didn't say
// error" tautology. Callers that also want to assert given_name/family_name/
// signup_methods mapping should follow up with adminClient.ts's
// getUserByEmail(opts.expectedEmail) after this resolves.
//
// expectedEmail is optional solely for Twitter/X (processTwitterUserInfo,
// internal/http_handlers/oauth_callback.go): unlike every other provider in
// this plan, real Twitter's API never returns an email address at all (no
// mock quirk - the mock's default/documented profile shape omits it on
// purpose, matching the real API), so there is no address to assert a
// mailto link against. Every other provider spec keeps passing a real
// expectedEmail and gets the exact same mailto assertion as before.
export async function runSocialLoginHappyPath(
  page: Page,
  request: APIRequestContext,
  opts: { provider: string; buttonName: RegExp; profile: Record<string, unknown>; expectedEmail?: string }
): Promise<void> {
  await configureProviderProfile(request, opts.provider, opts.profile);
  // Not /app/login - that route doesn't exist (web/app/src/Root.tsx only
  // registers "/app" for the unauthenticated Login view); /app/login renders
  // an empty container with zero inputs/buttons.
  await page.goto('/app');
  await page.getByRole('button', { name: opts.buttonName }).click();
  // The whole redirect chain (/oauth_login/:provider -> mock-oauth
  // /:provider/authorize -> /oauth_callback/:provider) always lands back on
  // /app (web/app/src/Root.tsx's redirectURL default), whether or not the
  // MFA-offer screen also fires - query string varies, path doesn't. Gin's
  // router.Group("/app") redirects the bare path to a trailing slash
  // (internal/server/http_routes.go), so the browser actually ends up on
  // "/app/", not "/app" - match both.
  await page.waitForURL((url) => url.pathname === '/app' || url.pathname === '/app/', { timeout: 15_000 });
  await dismissMfaSetupOffer(page);
  await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });
  if (opts.expectedEmail) {
    await expect(page.locator(`a[href="mailto:${opts.expectedEmail}"]`)).toBeVisible();
  }
}

// runConsentDeniedNegativePath mirrors what every provider actually sends
// when a user denies consent: the browser is redirected back with
// `error=access_denied` and no `code`, carrying the exact `state` this
// Authorizer instance issued for the attempt. internal/http_handlers/
// oauth_login.go and oauth_callback.go route `provider` as a generic path
// param and reject `error=access_denied` before any provider-specific logic
// runs, so this ~50-line flow is identical across all 10 social providers -
// shared here rather than hand-copied per provider spec (same reasoning as
// runSocialLoginHappyPath above).
export async function runConsentDeniedNegativePath(
  request: APIRequestContext,
  baseURL: string,
  provider: string
): Promise<void> {
  // Get a real state by driving the real login-initiation endpoint directly
  // (same route the rendered "Continue with <provider>" button hits) rather
  // than guessing its format - the compound state string is generated
  // server-side.
  const redirectUri = `${baseURL}/app`;
  const loginRes = await request.get(`/oauth_login/${provider}?redirect_uri=${encodeURIComponent(redirectUri)}`, {
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
  // is rejected before any provider call is made, so no user is ever looked
  // up or created - there is no "partial account" state to leave behind
  // here by construction, not just by assertion.
  const deniedRes = await request.get(
    `/oauth_callback/${provider}?error=access_denied&state=${encodeURIComponent(state!)}`,
    { maxRedirects: 0 }
  );
  expect(deniedRes.status()).toBe(400);
  const deniedBody = await deniedRes.json();
  expect(deniedBody.error).toBe('invalid oauth code');

  // The callback handler removes the state from the store as soon as it's
  // read, regardless of outcome (single-use, same RFC 6749 §4.1.2 discipline
  // as authorization codes - see the replay test in
  // tests/oidc-provider.spec.ts) - a retried request with the exact same
  // state must fail differently: unrecognized state, not "no code" again.
  const replayRes = await request.get(
    `/oauth_callback/${provider}?error=access_denied&state=${encodeURIComponent(state!)}`,
    { maxRedirects: 0 }
  );
  expect(replayRes.status()).toBe(400);
  const replayBody = await replayRes.json();
  expect(replayBody.error).toBe('invalid oauth state');
}
