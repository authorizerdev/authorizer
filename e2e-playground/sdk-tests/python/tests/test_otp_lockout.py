"""OTP brute-force lockout (#698) — driven through authorizer-py.

Mirrors tests/otp-lockout.spec.ts through the SDK: 5 failed verify_otp calls
inside the sliding window lock the user out with a distinct
"too many failed attempts" error (increment-then-check, so the 6th call — even
with a correct code — is the one rejected). A successful verification clears
the counter. The lock key is per-user, so it persists across logins. Fully
SDK-drivable: every attempt is an SDK verify_otp; the distinct error surfaces
as AuthorizerError.message.
"""

from __future__ import annotations

import pyotp
import pytest
from authorizer import (
    AuthorizerClient,
    AuthorizerError,
    LoginRequest,
    OtpMfaSetupRequest,
    SignUpRequest,
    VerifyOTPRequest,
)

import helpers

pytestmark = pytest.mark.live

LOCKOUT_MSG = "too many failed attempts"
INVALID_MSG = "invalid otp"


def _verify_error(client: AuthorizerClient, req: VerifyOTPRequest) -> str:
    with pytest.raises(AuthorizerError) as exc:
        client.verify_otp(req)
    return exc.value.message.lower()


def _totp_setup(client: AuthorizerClient, email: str) -> pyotp.TOTP:
    client.signup(
        SignUpRequest(email=email, password=helpers.PASSWORD, confirm_password=helpers.PASSWORD)
    )
    client.login(LoginRequest(email=email, password=helpers.PASSWORD))
    secret = client.totp_mfa_setup(OtpMfaSetupRequest(email=email)).authenticator_secret
    assert secret
    return pyotp.TOTP(secret)


def _wrong_totp(totp: pyotp.TOTP) -> str:
    return str((int(totp.now()) + 1) % 1_000_000).zfill(6)


def test_totp_lockout_after_five_failures(client: AuthorizerClient) -> None:
    email = helpers.random_email("totp-lockout")
    totp = _totp_setup(client, email)
    wrong = _wrong_totp(totp)

    for _ in range(5):
        assert INVALID_MSG in _verify_error(
            client, VerifyOTPRequest(email=email, otp=wrong, is_totp=True)
        )

    # 6th attempt (still wrong) -> distinct lockout error, not invalid-otp.
    assert LOCKOUT_MSG in _verify_error(
        client, VerifyOTPRequest(email=email, otp=wrong, is_totp=True)
    )

    # The CORRECT code is also refused while locked — lockout blocks
    # verification outright, it doesn't just keep rejecting wrong guesses.
    assert LOCKOUT_MSG in _verify_error(
        client, VerifyOTPRequest(email=email, otp=totp.now(), is_totp=True)
    )


def test_totp_success_resets_failure_counter(client: AuthorizerClient) -> None:
    email = helpers.random_email("totp-reset")
    totp = _totp_setup(client, email)
    wrong = _wrong_totp(totp)

    # 3 failures, then succeed — under the 5-attempt budget.
    for _ in range(3):
        _verify_error(client, VerifyOTPRequest(email=email, otp=wrong, is_totp=True))
    ok = client.verify_otp(VerifyOTPRequest(email=email, otp=totp.now(), is_totp=True))
    assert ok.access_token

    # Log in again; the lock key is per-user, so if the reset hadn't happened
    # the prior 3 + 5 new = 8 would lock out before the 5th. Assert all 5 are
    # plain invalid-otp, never the lockout error.
    client.login(LoginRequest(email=email, password=helpers.PASSWORD))
    for _ in range(5):
        assert LOCKOUT_MSG not in _verify_error(
            client, VerifyOTPRequest(email=email, otp=wrong, is_totp=True)
        )


def test_sms_otp_lockout_correct_code_refused_while_locked(client: AuthorizerClient) -> None:
    email = helpers.random_email("sms-lockout")
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
    assert login.should_offer_sms_otp_mfa_setup is True

    client.sms_otp_mfa_setup(OtpMfaSetupRequest(phone_number=phone))
    correct = helpers.extract_otp(helpers.wait_for_sms(phone))
    wrong = "ZZZZZZ"  # outside the OTP charset window -> guaranteed mismatch

    for _ in range(5):
        assert "otp" in _verify_error(
            client, VerifyOTPRequest(phone_number=phone, otp=wrong, is_totp=False)
        )

    assert LOCKOUT_MSG in _verify_error(
        client, VerifyOTPRequest(phone_number=phone, otp=wrong, is_totp=False)
    )
    # Correct code refused while locked.
    assert LOCKOUT_MSG in _verify_error(
        client, VerifyOTPRequest(phone_number=phone, otp=correct, is_totp=False)
    )
