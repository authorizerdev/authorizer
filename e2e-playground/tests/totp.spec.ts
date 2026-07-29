// e2e-playground/tests/totp.spec.ts
import { test, expect, APIRequestContext } from '@playwright/test';
import * as OTPAuth from 'otpauth';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';

const PASSWORD = 'Str0ngPassw0rd!';

function randomEmail() {
  return `totp-${crypto.randomUUID()}@example.com`;
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
