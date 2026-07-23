// e2e-playground/tests/totp.spec.ts
import { test, expect, APIRequestContext } from '@playwright/test';
import * as OTPAuth from 'otpauth';
import crypto from 'node:crypto';

const PASSWORD = 'Str0ngPassw0rd!';

function randomEmail() {
  return `totp-${crypto.randomUUID()}@example.com`;
}

// This spec is pure GraphQL (no browser UI) but CANNOT use a plain
// graphql-request GraphQLClient the way tests/oidc-provider.spec.ts does:
// the MFA flow is cookie-based (mfa_session / mfa_session_domain, set by
// login's Set-Cookie header and read back by totp_mfa_setup/verify_otp -
// see internal/cookie/mfa_session.go and internal/service/otp_mfa_setup.go's
// resolveOTPSetupCaller), and neither graphql-request nor Node's fetch
// maintain a cookie jar across separate client.request() calls. Playwright's
// own `request` fixture (APIRequestContext) does maintain a cookie jar
// automatically across calls made through it within one test - confirmed by
// running this spec - so login's Set-Cookie is replayed on the following
// totp_mfa_setup/verify_otp calls with no manual Cookie passthrough needed.
async function graphql<T = any>(
  request: APIRequestContext,
  baseURL: string,
  query: string,
  variables: Record<string, unknown>
): Promise<T> {
  const res = await request.post('/graphql', {
    data: { query, variables },
    // CSRF middleware requires Origin/Referer on state-changing requests
    // (internal/http_handlers/csrf.go) - same rationale as the GraphQLClient
    // Origin header in tests/oidc-provider.spec.ts. Playwright's request
    // fixture doesn't send Origin the way a real browser would, so it's set
    // explicitly here. The other CSRF requirement (Content-Type:
    // application/json) is satisfied automatically: passing a plain object
    // as `data` makes Playwright JSON-encode the body and set that header.
    headers: { Origin: baseURL },
  });
  const body = await res.json();
  if (body.errors) {
    throw new Error(`GraphQL error: ${JSON.stringify(body.errors)}`);
  }
  return body.data as T;
}

async function signupAndReachTotpChallenge(request: APIRequestContext, baseURL: string, email: string) {
  const signup = `
    mutation ($params: SignUpRequest!) {
      signup(params: $params) { message }
    }
  `;
  await graphql(request, baseURL, signup, {
    params: { email, password: PASSWORD, confirm_password: PASSWORD },
  });

  // Brand-new user, MFA enabled server-wide by default (EnableTOTPLogin) and
  // not yet enrolled/skipped -> mfaGateOfferAll (internal/service/mfa_gate.go):
  // the token is withheld and should_show_totp_screen comes back true,
  // alongside the mfa_session cookie login.go's setMFASession arms.
  const login = `
    mutation ($params: LoginRequest!) {
      login(params: $params) { message should_show_totp_screen }
    }
  `;
  const loginRes = await graphql<{ login: { message: string; should_show_totp_screen: boolean | null } }>(
    request,
    baseURL,
    login,
    { params: { email, password: PASSWORD } }
  );
  expect(loginRes.login.should_show_totp_screen).toBe(true);
}

async function setupTotp(request: APIRequestContext, baseURL: string, email: string): Promise<string> {
  // No bearer token yet (login withheld it) - totp_mfa_setup resolves the
  // caller via the mfa_session cookie, using params.email only to identify
  // whose session cookie this is (OtpMfaSetupRequest doc comment).
  const setup = `
    mutation ($params: OtpMfaSetupRequest) {
      totp_mfa_setup(params: $params) { authenticator_secret }
    }
  `;
  const setupRes = await graphql<{ totp_mfa_setup: { authenticator_secret: string } }>(request, baseURL, setup, {
    params: { email },
  });
  const secret = setupRes.totp_mfa_setup.authenticator_secret;
  expect(secret).toBeTruthy();
  return secret;
}

test.describe('TOTP', () => {
  test('enroll and complete login challenge with a computed code', async ({ request, baseURL }) => {
    const email = randomEmail();
    await signupAndReachTotpChallenge(request, baseURL!, email);
    const secret = await setupTotp(request, baseURL!, email);

    const totp = new OTPAuth.TOTP({ secret: OTPAuth.Secret.fromBase32(secret), digits: 6, period: 30 });
    const code = totp.generate();

    const verify = `
      mutation ($params: VerifyOTPRequest!) {
        verify_otp(params: $params) { message }
      }
    `;
    const verifyRes = await graphql<{ verify_otp: { message: string } }>(request, baseURL!, verify, {
      params: { email, otp: code, is_totp: true },
    });
    expect(verifyRes.verify_otp.message).toBeTruthy();
  });

  test('expired code is rejected', async ({ request, baseURL }) => {
    const email = randomEmail();
    await signupAndReachTotpChallenge(request, baseURL!, email);
    const secret = await setupTotp(request, baseURL!, email);

    // A code computed 10 minutes away from now is well outside the
    // validator's tolerance (github.com/pquerna/otp/totp.Validate, called
    // from internal/authenticators/totp/totp.go's Validate, defaults to
    // Period:30/Skew:1 - i.e. accepts the current 30s step plus one step
    // either side, ~90s total), so it must be rejected as invalid.
    const totp = new OTPAuth.TOTP({ secret: OTPAuth.Secret.fromBase32(secret), digits: 6, period: 30 });
    const staleCode = totp.generate({ timestamp: Date.now() - 10 * 60 * 1000 });

    const verify = `
      mutation ($params: VerifyOTPRequest!) {
        verify_otp(params: $params) { message }
      }
    `;
    const res = await request.post('/graphql', {
      data: { query: verify, variables: { params: { email, otp: staleCode, is_totp: true } } },
      headers: { Origin: baseURL! },
    });
    const body = await res.json();
    expect(body.errors).toBeTruthy();
    expect(JSON.stringify(body.errors)).toMatch(/invalid otp/i);
  });
});
