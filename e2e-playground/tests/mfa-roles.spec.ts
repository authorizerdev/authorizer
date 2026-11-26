// e2e-playground/tests/mfa-roles.spec.ts
//
// Guards #925: the roles a login requests must survive the MFA step. The token
// is minted by verify_otp, which never sees login's `roles`, and used to fall
// back to every role stored on the user. Both MFA completion paths are covered:
// verifying an already-enrolled TOTP, and enrolling TOTP during the login
// (setup offered with the MFA session cookie, then verified).
//
// Runs in the `mfa-off` project against the shared `authorizer` instance
// (TOTP login on by default, MFA optional), which configures
// --roles=user,editor,viewer for this spec.
import { test, expect, APIRequestContext } from '@playwright/test';
import * as OTPAuth from 'otpauth';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';
import { jwtClaims } from '../fixtures/oauth';

const PASSWORD = 'Str0ngPassw0rd!';
const ALL_ROLES = ['user', 'editor', 'viewer'];

const signupMutation = `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`;
const loginMutation = `mutation ($params: LoginRequest!) { login(params: $params) { message should_show_totp_screen access_token } }`;
const setupMutation = `mutation ($params: OtpMfaSetupRequest) { totp_mfa_setup(params: $params) { authenticator_secret } }`;
const verifyMutation = `mutation ($params: VerifyOTPRequest!) { verify_otp(params: $params) { access_token } }`;

type LoginRes = { login: { should_show_totp_screen: boolean | null; access_token: string | null } };

function totpFor(secret: string) {
  return new OTPAuth.TOTP({ secret: OTPAuth.Secret.fromBase32(secret), digits: 6, period: 30 });
}

async function signupWithAllRoles(request: APIRequestContext, baseURL: string): Promise<string> {
  const email = `mfa-roles-${crypto.randomUUID()}@example.com`;
  await graphql(request, baseURL, signupMutation, {
    params: { email, password: PASSWORD, confirm_password: PASSWORD, roles: ALL_ROLES },
  });
  return email;
}

// loginEnrolAndVerify: login (token withheld) → TOTP setup on the MFA cookie →
// verify. Returns the access token and the secret.
async function loginEnrolAndVerify(request: APIRequestContext, baseURL: string, email: string, roles?: string[]) {
  const login = await graphql<LoginRes>(request, baseURL, loginMutation, { params: { email, password: PASSWORD, roles } });
  expect(login.login.should_show_totp_screen).toBe(true);
  expect(login.login.access_token).toBeFalsy();
  const setup = await graphql<{ totp_mfa_setup: { authenticator_secret: string } }>(request, baseURL, setupMutation, {
    params: { email },
  });
  const secret = setup.totp_mfa_setup.authenticator_secret;
  const verify = await graphql<{ verify_otp: { access_token: string } }>(request, baseURL, verifyMutation, {
    params: { email, otp: totpFor(secret).generate(), is_totp: true },
  });
  return { accessToken: verify.verify_otp.access_token, secret };
}

test.describe('MFA carries login roles (#925)', () => {
  test('enrolled TOTP: login with roles=[editor] → verify → token roles == [editor]', async ({ playwright, baseURL }) => {
    // Enrol once in its own cookie jar, without a roles request.
    const enrol = await playwright.request.newContext();
    const login2 = await playwright.request.newContext();
    try {
      const email = await signupWithAllRoles(enrol, baseURL!);
      const { accessToken: enrolToken, secret } = await loginEnrolAndVerify(enrol, baseURL!, email);
      // Baseline: with no roles requested, login resolves the default roles
      // (--default-roles=user), not every role stored on the user — the same
      // as a login with no MFA step.
      expect(jwtClaims(enrolToken).roles).toEqual(['user']);

      // Fresh jar = fresh MFA session for the second login.
      const login = await graphql<LoginRes>(login2, baseURL!, loginMutation, {
        params: { email, password: PASSWORD, roles: ['editor'] },
      });
      expect(login.login.should_show_totp_screen).toBe(true);
      expect(login.login.access_token).toBeFalsy();

      // The next 30s step: inside the validator's ±1 step skew, and never
      // the step just spent on enrolment, so replay protection cannot reject
      // it and no wait is needed.
      const code = totpFor(secret).generate({ timestamp: Date.now() + 30_000 });
      const verify = await graphql<{ verify_otp: { access_token: string } }>(login2, baseURL!, verifyMutation, {
        params: { email, otp: code, is_totp: true },
      });
      expect(jwtClaims(verify.verify_otp.access_token).roles).toEqual(['editor']);
    } finally {
      await enrol.dispose();
      await login2.dispose();
    }
  });

  test('enrol during login: roles=[editor] → TOTP setup on the MFA cookie → verify → token roles == [editor]', async ({
    request,
    baseURL,
  }) => {
    const email = await signupWithAllRoles(request, baseURL!);
    const { accessToken } = await loginEnrolAndVerify(request, baseURL!, email, ['editor']);
    expect(jwtClaims(accessToken).roles).toEqual(['editor']);
  });
});
