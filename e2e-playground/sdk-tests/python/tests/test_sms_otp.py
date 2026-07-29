"""SMS-OTP MFA — enroll + verify, driven through authorizer-py.

Reproduces tests/sms-otp.spec.ts: signup with a phone number, login (token
withheld, SMS-OTP offered), sms_otp_mfa_setup, read the delivered code from
the sms-sink mock, verify_otp. Fully SDK-drivable — the only non-SDK step is
reading the mock's captured SMS (a third-party delivery sink the SDK does not,
and should not, wrap).

WebOTP note: "WebOTP" is not a distinct server flow — it is the browser
WebOTP autofill API reading this exact same SMS code out of the message and
pre-filling the OTP field (see tests/web-otp.spec.ts, which asserts the
`@` origin-bound autofill hint in the SMS body). The wire flow the SDK drives
is identical to SMS-OTP; the autofill itself is a browser-only UX layer with
no SDK surface, so it is intentionally not reproduced here.
"""

from __future__ import annotations

import pytest
from authorizer import (
    AuthorizerClient,
    LoginRequest,
    OtpMfaSetupRequest,
    SignUpRequest,
    VerifyOTPRequest,
)

import helpers

pytestmark = pytest.mark.live


def test_sms_otp_enroll_and_complete_login(client: AuthorizerClient) -> None:
    email = helpers.random_email("sms-otp")
    phone = helpers.random_phone()

    client.signup(
        SignUpRequest(
            email=email,
            phone_number=phone,
            password=helpers.PASSWORD,
            confirm_password=helpers.PASSWORD,
        )
    )

    login = client.login(LoginRequest(email=email, password=helpers.PASSWORD))
    # SMS OTP enabled + test SMS webhook configured -> setup offered, token withheld.
    assert login.should_offer_sms_otp_mfa_setup is True
    assert login.access_token is None

    # Resolved via the mfa_session cookie (same httpx client), keyed by phone.
    client.sms_otp_mfa_setup(OtpMfaSetupRequest(phone_number=phone))

    code = helpers.extract_otp(helpers.wait_for_sms(phone))
    token = client.verify_otp(
        VerifyOTPRequest(phone_number=phone, otp=code, is_totp=False)
    )

    assert token.access_token, "a valid SMS OTP must complete the challenge"
    assert token.user is not None
    assert token.user.email == email
