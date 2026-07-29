// e2e-playground/tests/sms-otp.spec.ts
import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';

const PASSWORD = 'Str0ngPassw0rd!';
const SMS_SINK_BASE = process.env.SMS_SINK_BASE_URL || 'http://localhost:4100';

function randomEmail() {
  return `sms-otp-${crypto.randomUUID()}@example.com`;
}

function randomPhone() {
  // E.164-ish, random enough to avoid collisions across parallel test runs
  // (crypto.randomInt, not Date.now() - same reasoning as randomEmail).
  return `+1555${crypto.randomInt(1000000, 9999999)}`;
}

// waitForSMS polls mocks/sms-sink's GET /sms/:phone (server.ts) for the
// latest message sent to `phone`. The mock 404s until a message has been
// recorded, and returns { phone, message } once it has (confirmed against
// mocks/sms-sink/server.ts - not the plan's untested guess).
async function waitForSMS(request: import('@playwright/test').APIRequestContext, phone: string): Promise<string> {
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

// extractOTP pulls the code out of "Your verification code is: XXXXXX"
// (internal/service/otp_mfa_setup.go's SMSOTPMFASetup). The code itself is
// NOT purely numeric - utils.GenerateOTP (internal/utils/generate_otp.go)
// draws from charset "ABCDEFGHJKLMNPQRSTUVWXYZ123456789" (ambiguous
// I/O/0/1 excluded), so a naive \d{6} regex (the plan's untested guess)
// would never match. Splitting on the fixed "code is: " prefix is exact and
// charset-agnostic.
function extractOTP(message: string): string {
  const match = message.match(/code is:\s*([A-Z0-9]{6})/);
  if (!match) throw new Error(`could not find OTP in SMS body: ${message}`);
  return match[1];
}

test.describe('SMS-OTP', () => {
  test('enroll, receive code via sms-sink, complete login challenge', async ({ request, baseURL }) => {
    const email = randomEmail();
    const phone = randomPhone();

    const signup = `
      mutation ($params: SignUpRequest!) {
        signup(params: $params) { message }
      }
    `;
    await graphql(request, baseURL!, signup, {
      params: { email, phone_number: phone, password: PASSWORD, confirm_password: PASSWORD },
    });

    // Brand-new user, MFA enabled server-wide by default and not yet
    // enrolled/skipped -> mfaGateOfferAll (internal/service/mfa_gate.go):
    // the token is withheld and should_offer_sms_otp_mfa_setup comes back
    // true (SMS OTP is enabled + the test webhook counts as the SMS service
    // being configured - internal/config/config.go's IsSMSServiceEnabled),
    // alongside the mfa_session cookie login.go's setMFASession arms.
    const login = `
      mutation ($params: LoginRequest!) {
        login(params: $params) { message should_offer_sms_otp_mfa_setup }
      }
    `;
    const loginRes = await graphql<{
      login: { message: string; should_offer_sms_otp_mfa_setup: boolean | null };
    }>(request, baseURL!, login, { params: { email, password: PASSWORD } });
    expect(loginRes.login.should_offer_sms_otp_mfa_setup).toBe(true);

    // No bearer token yet (login withheld it) - sms_otp_mfa_setup resolves
    // the caller via the mfa_session cookie, using params.phone_number to
    // identify whose session cookie this is (OtpMfaSetupRequest doc
    // comment), same pattern as totp.spec.ts's setupTotp but keyed on phone
    // instead of email.
    const setup = `
      mutation ($params: OtpMfaSetupRequest) {
        sms_otp_mfa_setup(params: $params) { message }
      }
    `;
    await graphql(request, baseURL!, setup, { params: { phone_number: phone } });

    const smsBody = await waitForSMS(request, phone);
    const code = extractOTP(smsBody);

    const verify = `
      mutation ($params: VerifyOTPRequest!) {
        verify_otp(params: $params) { message access_token }
      }
    `;
    const verifyRes = await graphql<{ verify_otp: { message: string; access_token: string | null } }>(
      request,
      baseURL!,
      verify,
      { params: { phone_number: phone, otp: code, is_totp: false } }
    );
    expect(verifyRes.verify_otp.access_token).toBeTruthy();
  });
});
