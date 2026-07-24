// e2e-playground/tests/web-otp.spec.ts
import { test, expect, APIRequestContext, Page } from '@playwright/test';
import crypto from 'node:crypto';
import { graphql } from '../fixtures/graphqlRequest';

const PASSWORD = 'Str0ngPassw0rd!';
const SMS_SINK_BASE = process.env.SMS_SINK_BASE_URL || 'http://localhost:4100';

function randomEmail() {
  return `web-otp-${crypto.randomUUID()}@example.com`;
}

function randomPhone() {
  return `+1555${crypto.randomInt(1000000, 9999999)}`;
}

// Same bounded poll as tests/sms-otp.spec.ts's waitForSMS (mocks/sms-sink's
// GET /sms/:phone 404s until a message has been recorded for that phone,
// server.ts's `latestByPhone` map). Capped at 20 * 250ms = 5s so a missing
// SMS fails fast instead of hanging - never an unbounded wait.
//
// `excluding`: this spec sends TWO SMS messages to the same phone number
// (the enrollment code, then a fresh code on the second login - see
// loginToOtpVerifyScreen's comment), and sms-sink only keeps the latest
// message per phone. Without this, a call made right after the second
// login could observe the stale enrollment-code message (already recorded
// before the new one lands, since login.go sends it via an async
// goroutine) and grab an already-consumed code instead of waiting for the
// new one.
async function waitForSMS(request: APIRequestContext, phone: string, opts?: { excluding?: string }): Promise<string> {
  for (let i = 0; i < 20; i++) {
    const res = await request.get(`${SMS_SINK_BASE}/sms/${encodeURIComponent(phone)}`);
    if (res.status() === 200) {
      const body = await res.json();
      const message = body.message as string;
      if (!opts?.excluding || message !== opts.excluding) {
        return message;
      }
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`no (new) SMS received for ${phone} within timeout`);
}

// Same exact regex as tests/sms-otp.spec.ts's extractOTP - the code is
// alphanumeric (internal/utils/generate_otp.go's charset
// "ABCDEFGHJKLMNPQRSTUVWXYZ123456789"), never purely numeric.
function extractOTP(message: string): string {
  const match = message.match(/code is:\s*([A-Z0-9]{6})/);
  if (!match) throw new Error(`could not find OTP in SMS body: ${message}`);
  return match[1];
}

// Enrolls `email`/`phone` in SMS-OTP MFA via GraphQL only - same sequence
// as tests/sms-otp.spec.ts: signup -> login (withheld token,
// should_offer_sms_otp_mfa_setup true, mfa_session cookie armed) ->
// sms_otp_mfa_setup -> verify_otp completes enrollment. This spec is about
// the *verify* screen reached on a later login, not the enrollment UI, so
// enrollment stays on the GraphQL fast path like Task 25 established.
// Returns the enrollment SMS body, so callers can pass it as `excluding`
// to a later waitForSMS call (see that function's comment).
async function enrollSmsOtp(request: APIRequestContext, baseURL: string, email: string, phone: string): Promise<string> {
  const signup = `
    mutation ($params: SignUpRequest!) {
      signup(params: $params) { message }
    }
  `;
  await graphql(request, baseURL, signup, {
    params: { email, phone_number: phone, password: PASSWORD, confirm_password: PASSWORD },
  });

  const login = `
    mutation ($params: LoginRequest!) {
      login(params: $params) { message should_offer_sms_otp_mfa_setup }
    }
  `;
  const loginRes = await graphql<{ login: { should_offer_sms_otp_mfa_setup: boolean | null } }>(
    request,
    baseURL,
    login,
    { params: { email, password: PASSWORD } }
  );
  expect(loginRes.login.should_offer_sms_otp_mfa_setup).toBe(true);

  const setup = `
    mutation ($params: OtpMfaSetupRequest) {
      sms_otp_mfa_setup(params: $params) { message }
    }
  `;
  await graphql(request, baseURL, setup, { params: { phone_number: phone } });

  const enrollMessage = await waitForSMS(request, phone);
  const enrollCode = extractOTP(enrollMessage);

  const verify = `
    mutation ($params: VerifyOTPRequest!) {
      verify_otp(params: $params) { access_token }
    }
  `;
  const verifyRes = await graphql<{ verify_otp: { access_token: string | null } }>(request, baseURL, verify, {
    params: { phone_number: phone, otp: enrollCode, is_totp: false },
  });
  expect(verifyRes.verify_otp.access_token).toBeTruthy();

  return enrollMessage;
}

// Drives a REAL second login through the rendered /app login form (same
// locators as tests/webauthn.spec.ts, confirmed against
// AuthorizerBasicAuthLogin.tsx source: #authorizer-login-email-or-phone-number,
// #authorizer-login-password, form[name="authorizer-login-form"]) for a
// user who is already enrolled in SMS-OTP from enrollSmsOtp above. Because
// a *verified* SMS-OTP authenticator now exists, this login lands on the
// verify screen (AuthorizerVerifyOtp, #authorizer-verify-otp input) instead
// of the first-time setup screen: internal/service/login.go's
// smsOTPEnrolled branch short-circuits straight past the MFA-gate/offer
// logic, generates+stores a fresh OTP, fires SMSProvider.SendSMS
// asynchronously (asyncutil.Go - it lands in sms-sink shortly after, not
// necessarily before this function returns), and returns
// should_show_mobile_otp_screen: true - which authorizer-react's
// resolveAuthStep (mfaTriage.ts) turns into step.kind === 'verify', mobile:
// true, rendering AuthorizerVerifyOtp inline (confirmed from source).
async function loginToOtpVerifyScreen(page: Page, email: string): Promise<void> {
  // Defensive: the `request` fixture used for enrollment and `page` are
  // separate contexts, but clearing cookies removes any doubt that a stray
  // enrollment-flow cookie could auto-authenticate the page and skip the
  // login form entirely.
  await page.context().clearCookies();
  await page.goto('/app');
  await page.locator('#authorizer-login-email-or-phone-number').fill(email);
  await page.locator('#authorizer-login-password').fill(PASSWORD);
  await page.locator('form[name="authorizer-login-form"] button[type="submit"]').click();
  await expect(page.locator('#authorizer-verify-otp')).toBeVisible({ timeout: 10_000 });
}

test.describe('WebOTP (SMS-OTP code-entry screen)', () => {
  // REGRESSION/FEATURE COVERAGE: this was originally skipped because
  // authorizer-react never called navigator.credentials.get() for the
  // SMS-OTP screen at all (verified against source at the time) - only the
  // declarative autoComplete="one-time-code" hint existed, with no
  // JS-observable hook for a mock to attach to. authorizer-react's
  // WebOTP PR (shipped in the 2.2.0-rc.3 dependency bump) added a real
  // navigator.credentials.get({otp:{transport:['sms']}}) call, gated on a
  // `hasSmsOtp` prop and feature-detected via `'OTPCredential' in window`
  // (AuthorizerVerifyOtp.tsx). AuthorizerBasicAuthLogin.tsx (the internal
  // component that renders AuthorizerVerifyOtp for this exact
  // should_show_mobile_otp_screen flow, via loginToOtpVerifyScreen below)
  // already passes hasSmsOtp: otpData.has_sms_otp - so this path is
  // wired end-to-end without any web/app change. (web/app's Root.tsx
  // needed its own separate hasSmsOtp fix for the mfa-redirect/query-param
  // flow specifically, a different code path from the one this test
  // exercises - see Root.tsx's <AuthorizerVerifyOtp>.)
  test('auto-fill via mocked navigator.credentials.get()', async ({ page, request, baseURL }) => {
    const email = randomEmail();
    const phone = randomPhone();
    const enrollMessage = await enrollSmsOtp(request, baseURL!, email, phone);

    // Bridges into Node so the stubbed navigator.credentials.get() below can
    // wait for the real login-time SMS the same way a real WebOTP-capable
    // browser would (it never resolves with a canned value - only with an
    // actually-received matching message), reusing the exact same
    // sms-sink poll + excluding-the-enrollment-code logic as the manual-entry
    // test below.
    await page.exposeFunction('__e2eWebOtpCode', async () => {
      const message = await waitForSMS(request, phone, { excluding: enrollMessage });
      return extractOTP(message);
    });

    // Feature-detect + stub navigator.credentials.get before any page script
    // runs, so AuthorizerVerifyOtp's `'OTPCredential' in window` check
    // passes and its real navigator.credentials.get({otp:...}) call
    // resolves via the real SMS code fetched above - exercising the
    // product's actual WebOTP code path, not test-authored glue standing in
    // for it.
    await page.addInitScript(() => {
      // Minimal marker - the product only checks `'OTPCredential' in
      // window`; it never constructs or inspects the class itself.
      (window as any).OTPCredential = function OTPCredential() {};
      const nav = window.navigator as any;
      nav.credentials = nav.credentials || {};
      nav.credentials.get = (opts: any) => {
        if (!opts || !opts.otp) {
          return Promise.reject(new Error('e2e stub only implements otp credential requests'));
        }
        return (window as any).__e2eWebOtpCode().then((code: string) => ({ code }));
      };
    });

    await loginToOtpVerifyScreen(page, email);

    // The product's WebOTP effect races the stubbed navigator.credentials.get()
    // against manual entry - assert its resolution actually reached the
    // controlled input, with nobody ever calling .fill() on it.
    await expect(page.locator('#authorizer-verify-otp')).toHaveValue(/^[A-Z0-9]{6}$/, { timeout: 10_000 });

    // The auto-filled value is real and submittable - completes login.
    await page.locator('form[name="authorizer-mfa-otp-form"] button[type="submit"]').click();
    await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });
  });

  test('code-entry input declares the browser-native WebOTP hint (autocomplete="one-time-code")', async ({
    page,
    request,
    baseURL,
  }) => {
    const email = randomEmail();
    const phone = randomPhone();
    await enrollSmsOtp(request, baseURL!, email, phone);
    await loginToOtpVerifyScreen(page, email);

    // The one real, verifiable WebOTP contract this product ships - see
    // the skipped test above for what does not exist.
    await expect(page.locator('#authorizer-verify-otp')).toHaveAttribute('autocomplete', 'one-time-code');
  });

  test('manual entry: typing the SMS code in still completes login', async ({ page, request, baseURL }) => {
    const email = randomEmail();
    const phone = randomPhone();
    const enrollMessage = await enrollSmsOtp(request, baseURL!, email, phone);
    await loginToOtpVerifyScreen(page, email);

    // login.go's smsOTPEnrolled branch (see loginToOtpVerifyScreen's
    // comment) sent a fresh SMS as part of the login above - poll
    // sms-sink for it, excluding the already-consumed enrollment code.
    const code = extractOTP(await waitForSMS(request, phone, { excluding: enrollMessage }));

    await page.locator('#authorizer-verify-otp').fill(code);
    await page.locator('form[name="authorizer-mfa-otp-form"] button[type="submit"]').click();

    // web/app/src/pages/dashboard.tsx only renders this once useAuthorizer's
    // token is actually set - a real proof of session, same assertion
    // tests/webauthn.spec.ts uses.
    await expect(page.getByText('Signed in as')).toBeVisible({ timeout: 10_000 });
  });
});
