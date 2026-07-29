// e2e-playground/tests/mfa-routing-matrix.spec.ts
import { test, expect, APIRequestContext } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';

// This spec runs against authorizer-mfa-enforced (docker-compose.yml, `mfa-on`
// project in playwright.config.ts) - the only instance with --enforce-mfa=true.
//
// EnforceMFA turned out NOT to be a runtime-toggleable admin setting, despite
// fixtures/adminClient.ts exporting a setEnforceMFA helper that calls the
// `_update_env` mutation: verified live against a running instance that this
// mutation is a stub which always errors -
// "deprecated. please configure env via cli args"
// (internal/graph/schema.resolvers.go's UpdateEnv resolver) - consistent with
// this repo's v2 CLI-flags-only config model (AGENTS.md: "no .env or OS env
// vars"). So, same as WebAuthn/magic-link/SSO before it, --enforce-mfa needs
// its own dedicated static-CLI-flag instance rather than a beforeAll/afterAll
// toggle against the shared `authorizer` service - setEnforceMFA is not used
// here at all.
const PASSWORD = 'Str0ngPassw0rd!';

// Dedicated second instance (also docker-compose.yml) for the one test below
// needing BOTH --enforce-mfa=true and magic-link login - see that service's
// comment in docker-compose.yml for why it can't be either of
// authorizer-magic-link (pinned to --disable-mfa=true, which forces
// EnforceMFA off) or authorizer-mfa-enforced (no magic-link/email-verification
// wiring, and turning that on there would break its own password-login tests
// below).
const MAGIC_LINK_MFA_BASE_URL = process.env.AUTHORIZER_MFA_MAGIC_LINK_BASE_URL || 'http://localhost:8085';
const MAILPIT_BASE = process.env.MAILPIT_BASE_URL || 'http://localhost:8025';

function randomEmail(prefix: string) {
  return `${prefix}-${crypto.randomUUID()}@example.com`;
}

const signupMutation = `
  mutation ($params: SignUpRequest!) {
    signup(params: $params) { message }
  }
`;

const loginMutation = `
  mutation ($params: LoginRequest!) {
    login(params: $params) { message access_token should_show_totp_screen }
  }
`;

// waitForMagicLinkURL mirrors tests/magic-link.spec.ts's helper (Task 27) -
// not shared/exported from there either, so duplicated here rather than
// introducing a new cross-spec fixture for one function. Bounded to 40
// tries * 250ms = 10s, matching that spec's own timeout.
async function waitForMagicLinkURL(request: APIRequestContext, email: string): Promise<string> {
  for (let i = 0; i < 40; i++) {
    const res = await request.get(`${MAILPIT_BASE}/api/v1/messages`);
    const { messages } = await res.json();
    const msg = messages.find((m: { To: { Address: string }[] }) => m.To.some((t) => t.Address === email));
    if (msg) {
      const detail = await (await request.get(`${MAILPIT_BASE}/api/v1/message/${msg.ID}`)).json();
      const match = /(https?:\/\/\S+verify_email\?\S+)/.exec(detail.Text);
      if (match) return match[1].replace(/\)$/, '');
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`no magic link email received for ${email} within 10s`);
}

test.describe('EnforceMFA=true routing matrix', () => {
  test('password login, no factor enrolled: token withheld, routed to mfa enrollment', async ({ request, baseURL }) => {
    const email = randomEmail('mfa-matrix');
    await graphql(request, baseURL!, signupMutation, {
      params: { email, password: PASSWORD, confirm_password: PASSWORD },
    });

    // mfaGateBlockEnroll (internal/service/mfa_gate.go, resolveMFAGate):
    // EnforceMFA is absolute for an unenrolled user, so login.go's
    // mfaGateBlockEnroll branch (lines ~435-454) withholds access_token
    // entirely - this is the real enforcement signal, not just a truthy
    // message. TOTP is enabled by default in this stack (no
    // --disable-totp-login flag set), so should_show_totp_screen comes back
    // true alongside the withheld token.
    const loginRes = await graphql<{
      login: { message: string; access_token: string | null; should_show_totp_screen: boolean | null };
    }>(request, baseURL!, loginMutation, { params: { email, password: PASSWORD } });

    expect(loginRes.login.access_token).toBeFalsy();
    expect(loginRes.login.message).toBe('Proceed to mfa setup');
    expect(loginRes.login.should_show_totp_screen).toBe(true);
  });

  test('skip_mfa_setup is rejected while enforcement is active', async ({ request, baseURL }) => {
    const email = randomEmail('mfa-matrix-skip');
    await graphql(request, baseURL!, signupMutation, {
      params: { email, password: PASSWORD, confirm_password: PASSWORD },
    });
    // Login withholds the token and arms the mfa_session cookie
    // (login.go's setMFASession) - Playwright's request fixture replays it
    // automatically on the skip_mfa_setup call below (same cookie-jar
    // behavior tests/totp.spec.ts and tests/sms-otp.spec.ts rely on).
    await graphql(request, baseURL!, loginMutation, { params: { email, password: PASSWORD } });

    // internal/service/skip_mfa_setup.go recomputes the gate server-side and
    // only allows the skip when gate === mfaGateOfferAll. Under enforcement
    // the gate is mfaGateBlockEnroll, so this must be rejected even though
    // the caller holds a real mfa_session cookie from a genuine login -
    // proves enforcement can't be routed around by calling skip directly,
    // not just that the login response *offers* setup.
    const skipMutation = `
      mutation ($params: SkipMfaSetupRequest!) {
        skip_mfa_setup(params: $params) { message }
      }
    `;
    const res = await request.post('/graphql', {
      data: { query: skipMutation, variables: { params: { email } } },
      headers: { Origin: baseURL! },
    });
    const body = await res.json();
    expect(body.errors).toBeTruthy();
    expect(JSON.stringify(body.errors)).toMatch(/cannot skip/i);
  });

  test('magic-link login still routes through the mfa challenge under enforcement', async ({ request }) => {
    const email = randomEmail('mfa-matrix-magic');
    const magicLinkLoginMutation = `
      mutation ($params: MagicLinkLoginRequest!) {
        magic_link_login(params: $params) { message }
      }
    `;
    await graphql(request, MAGIC_LINK_MFA_BASE_URL, magicLinkLoginMutation, { params: { email } });

    const link = await waitForMagicLinkURL(request, email);

    // VerifyEmailHandler (internal/http_handlers/verify_email.go) runs the
    // same gate as login.go via EvaluateMFAGateForOAuth
    // (internal/service/oauth_mfa_gate.go) before ever minting a token: for
    // mfaGateBlockEnroll it redirects with mfa_required=1&mfa_gate=offer
    // instead of access_token=... - assert on the real redirect target/query
    // params (not just "URL isn't /app"), per Task 13's established
    // mfa_required/mfa_gate/mfa_methods param handling in
    // web/app/src/Root.tsx.
    const res = await request.get(link, { maxRedirects: 0 });
    expect(res.status()).toBe(307);
    const location = res.headers()['location'];
    expect(location).not.toContain('access_token=');
    expect(location).toContain('mfa_required=1');
    expect(location).toContain('mfa_gate=offer');
  });
});
