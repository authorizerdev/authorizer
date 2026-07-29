// e2e-playground/tests/otp-lockout.spec.ts
import { test, expect, APIRequestContext } from '@playwright/test';
import * as OTPAuth from 'otpauth';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';

// #698 (internal/service/verify_otp.go): 5 failed verify_otp attempts within
// a 15-minute sliding window lock the user out with a distinct
// "too many failed attempts, please try again later" error - increment-then-
// check, so the 6th call (even with a correct code) is the one that's
// rejected. A successful verification clears the counter
// (DeleteCacheByPrefix), so the lockout isn't a one-way ratchet. The lock
// key is `totp_failed_attempts:<user.ID>` / `otp_failed_attempts:<user.ID>` -
// keyed by user, not by session - so it persists across logins, which is
// what makes the "reset" test below meaningful (a second login's failed
// attempts would inherit the first login's count if the reset didn't work).

const PASSWORD = 'Str0ngPassw0rd!';
// Any value outside the OTP charset window still round-trips through
// crypto.VerifyOTPHash as a mismatch - reusing sms-otp.spec.ts's documented
// charset (ABCDEFGHJKLMNPQRSTUVWXYZ123456789, I/O/0/1 excluded) purely so
// this wrong code looks plausible; the hash comparison is what actually
// rejects it.
const WRONG_SMS_CODE = 'ZZZZZZ';
const SMS_SINK_BASE = process.env.SMS_SINK_BASE_URL || 'http://localhost:4100';

function randomEmail(prefix: string) {
  return `${prefix}-${crypto.randomUUID()}@example.com`;
}

function randomPhone() {
  return `+1555${crypto.randomInt(1000000, 9999999)}`;
}

async function waitForSMS(request: APIRequestContext, phone: string): Promise<string> {
  for (let i = 0; i < 20; i++) {
    const res = await request.get(`${SMS_SINK_BASE}/sms/${encodeURIComponent(phone)}`);
    if (res.status() === 200) {
      const body = await res.json();
      return body.message as string;
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`no SMS received for ${phone} within timeout`);
}

function extractOTP(message: string): string {
  const match = message.match(/code is:\s*([A-Z0-9]{6})/);
  if (!match) throw new Error(`could not find OTP in SMS body: ${message}`);
  return match[1];
}

const LOGIN_MUTATION = `
  mutation ($params: LoginRequest!) {
    login(params: $params) { message should_show_totp_screen should_offer_sms_otp_mfa_setup }
  }
`;
const VERIFY_MUTATION = `
  mutation ($params: VerifyOTPRequest!) {
    verify_otp(params: $params) { message access_token }
  }
`;

async function login(request: APIRequestContext, baseURL: string, email: string) {
  return graphql<{
    login: { message: string; should_show_totp_screen: boolean | null; should_offer_sms_otp_mfa_setup: boolean | null };
  }>(request, baseURL, LOGIN_MUTATION, { params: { email, password: PASSWORD } });
}

async function verifyOTPRaw(
  request: APIRequestContext,
  baseURL: string,
  variables: Record<string, unknown>
): Promise<{ errors?: Array<{ message: string }>; data?: { verify_otp: { message: string; access_token: string | null } } }> {
  const res = await request.post('/graphql', {
    data: { query: VERIFY_MUTATION, variables },
    headers: { Origin: baseURL },
  });
  return res.json();
}

test.describe('OTP brute-force lockout (#698)', () => {
  test('TOTP: 5 wrong codes lock verification, distinct from a plain invalid-code error', async ({ request, baseURL }) => {
    const email = randomEmail('totp-lockout');
    await graphql(request, baseURL!, `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`, {
      params: { email, password: PASSWORD, confirm_password: PASSWORD },
    });
    const loginRes = await login(request, baseURL!, email);
    expect(loginRes.login.should_show_totp_screen).toBe(true);

    const setupRes = await graphql<{ totp_mfa_setup: { authenticator_secret: string } }>(
      request,
      baseURL!,
      `mutation ($params: OtpMfaSetupRequest) { totp_mfa_setup(params: $params) { authenticator_secret } }`,
      { params: { email } }
    );
    const secret = setupRes.totp_mfa_setup.authenticator_secret;
    const totp = new OTPAuth.TOTP({ secret: OTPAuth.Secret.fromBase32(secret), digits: 6, period: 30 });
    const wrongCode = String((parseInt(totp.generate(), 10) + 1) % 1_000_000).padStart(6, '0');

    for (let i = 0; i < 5; i++) {
      const body = await verifyOTPRaw(request, baseURL!, { params: { email, otp: wrongCode, is_totp: true } });
      expect(JSON.stringify(body.errors)).toMatch(/invalid otp/i);
    }

    // The 6th attempt - even with a WRONG code - must now return the
    // distinct lockout error, not the generic invalid-otp error.
    const lockedBody = await verifyOTPRaw(request, baseURL!, { params: { email, otp: wrongCode, is_totp: true } });
    expect(JSON.stringify(lockedBody.errors)).toMatch(/too many failed attempts/i);

    // The CORRECT code is also refused while locked - lockout blocks
    // verification outright, it doesn't just keep rejecting wrong guesses.
    const correctCode = totp.generate();
    const stillLockedBody = await verifyOTPRaw(request, baseURL!, { params: { email, otp: correctCode, is_totp: true } });
    expect(JSON.stringify(stillLockedBody.errors)).toMatch(/too many failed attempts/i);
  });

  test('TOTP: a successful verification resets the failed-attempt counter', async ({ request, baseURL }) => {
    const email = randomEmail('totp-reset');
    await graphql(request, baseURL!, `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`, {
      params: { email, password: PASSWORD, confirm_password: PASSWORD },
    });
    await login(request, baseURL!, email);
    const setupRes = await graphql<{ totp_mfa_setup: { authenticator_secret: string } }>(
      request,
      baseURL!,
      `mutation ($params: OtpMfaSetupRequest) { totp_mfa_setup(params: $params) { authenticator_secret } }`,
      { params: { email } }
    );
    const secret = setupRes.totp_mfa_setup.authenticator_secret;
    const totp = new OTPAuth.TOTP({ secret: OTPAuth.Secret.fromBase32(secret), digits: 6, period: 30 });
    const wrongCode = String((parseInt(totp.generate(), 10) + 1) % 1_000_000).padStart(6, '0');

    // 3 failed attempts, then succeed - well under the 5-attempt budget.
    for (let i = 0; i < 3; i++) {
      await verifyOTPRaw(request, baseURL!, { params: { email, otp: wrongCode, is_totp: true } });
    }
    const verifyRes = await graphql<{ verify_otp: { access_token: string | null } }>(request, baseURL!, VERIFY_MUTATION, {
      params: { email, otp: totp.generate(), is_totp: true },
    });
    expect(verifyRes.verify_otp.access_token).toBeTruthy();

    // Log in again (fresh mfa_session, but the lock key is keyed by user.ID
    // so it would still carry the prior count if the reset hadn't happened)
    // and run a full 5 wrong attempts - if the counter wasn't reset, this
    // batch would lock out well before the 5th (3 carried over + 3 new = 6).
    const secondLogin = await login(request, baseURL!, email);
    expect(secondLogin.login.should_show_totp_screen).toBe(true);
    for (let i = 0; i < 5; i++) {
      const body = await verifyOTPRaw(request, baseURL!, { params: { email, otp: wrongCode, is_totp: true } });
      expect(JSON.stringify(body.errors)).toMatch(/invalid otp/i);
    }
  });

  test('SMS-OTP: 5 wrong codes lock verification, correct code still refused while locked', async ({ request, baseURL }) => {
    const email = randomEmail('sms-otp-lockout');
    const phone = randomPhone();
    await graphql(request, baseURL!, `mutation ($params: SignUpRequest!) { signup(params: $params) { message } }`, {
      params: { email, phone_number: phone, password: PASSWORD, confirm_password: PASSWORD },
    });
    const loginRes = await login(request, baseURL!, email);
    expect(loginRes.login.should_offer_sms_otp_mfa_setup).toBe(true);

    await graphql(request, baseURL!, `mutation ($params: OtpMfaSetupRequest) { sms_otp_mfa_setup(params: $params) { message } }`, {
      params: { phone_number: phone },
    });
    const smsBody = await waitForSMS(request, phone);
    const correctCode = extractOTP(smsBody);

    for (let i = 0; i < 5; i++) {
      const body = await verifyOTPRaw(request, baseURL!, { params: { phone_number: phone, otp: WRONG_SMS_CODE, is_totp: false } });
      expect(JSON.stringify(body.errors)).toMatch(/otp/i);
    }

    const lockedBody = await verifyOTPRaw(request, baseURL!, { params: { phone_number: phone, otp: WRONG_SMS_CODE, is_totp: false } });
    expect(JSON.stringify(lockedBody.errors)).toMatch(/too many failed attempts/i);

    const stillLockedBody = await verifyOTPRaw(request, baseURL!, { params: { phone_number: phone, otp: correctCode, is_totp: false } });
    expect(JSON.stringify(stillLockedBody.errors)).toMatch(/too many failed attempts/i);
  });
});
