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
  // ---------------------------------------------------------------------
  // What this file does NOT (and cannot honestly) test: real WebOTP
  // auto-fill via a mocked navigator.credentials.get().
  //
  // Investigated directly against source, not the plan's assumption:
  // AuthorizerVerifyOtp.tsx (node_modules/@authorizerdev/authorizer-react/
  // src/components/AuthorizerVerifyOtp.tsx) never calls
  // navigator.credentials.get() anywhere. Confirmed by grepping the full
  // authorizer-react source tree plus web/app/src for "credentials.get",
  // "OTPCredential", and "WebOTP": zero matches. authorizer-js ships no
  // source in node_modules (compiled lib/ only); its compiled output DOES
  // contain one navigator.credentials.get({ publicKey: ... }) call, inside
  // loginWithPasskey - WebAuthn/passkey login, a different UI element and a
  // different Credential Management API mode (publicKey, not otp) from the
  // #authorizer-verify-otp SMS-OTP input this file exercises. Unrelated to
  // WebOTP, doesn't change the conclusion below.
  //
  // The only WebOTP-related surface that exists anywhere in this stack is
  // the declarative autoComplete="one-time-code" attribute on the
  // #authorizer-verify-otp <input> (asserted below) - a purely browser/OS-
  // native mechanism (the native SMS Retriever / keyboard-suggestion chip)
  // with no JS-observable hook. A page only gets programmatic WebOTP
  // autofill by calling navigator.credentials.get({ otp: {...} }) itself;
  // browsers do not wire the declarative attribute to that API internally
  // on the page's behalf.
  //
  // So mocking navigator.credentials.get() via page.addInitScript, as the
  // brief suggested, would have zero effect on any real code path here -
  // nothing in the product ever calls it. A "passing" test built around
  // that mock would necessarily contain test-authored glue that calls the
  // mock itself and manually fills/dispatches the input's value, which
  // only proves the test's own polyfill works, not that the product does
  // anything with WebOTP. That is not a test-authoring mistake to correct;
  // it is a genuine absence of the programmatic WebOTP integration in the
  // product (spans authorizer, authorizer-js, and authorizer-react - not
  // fixable from this repo alone). Documented here, as the brief's own
  // escape hatch calls for, instead of shipping a fake assertion.
  test('auto-fill via mocked navigator.credentials.get()', async () => {
    test.skip(
      true,
      'authorizer-react never calls navigator.credentials.get({otp:...}) anywhere for the SMS-OTP verify input ' +
        '(grep-verified against source; authorizer-js\'s only call to that API is an unrelated WebAuthn/passkey ' +
        'login) - only the declarative autoComplete="one-time-code" hint exists on #authorizer-verify-otp, and it ' +
        'has no JS-observable hook for a mock to attach to. See the describe-block comment above for the full investigation.'
    );
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
