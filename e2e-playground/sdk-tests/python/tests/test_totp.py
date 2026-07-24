"""TOTP MFA — enroll + verify, driven entirely through authorizer-py.

Reproduces tests/totp.spec.ts's flow, but every server interaction goes
through the SDK's own methods (signup / login / totp_mfa_setup / verify_otp)
instead of raw GraphQL. Fully SDK-drivable: these are plain request/response
calls with no browser ceremony.
"""

from __future__ import annotations

import time

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


def _reach_totp_setup(client: AuthorizerClient, email: str) -> str:
    """signup -> login (token withheld, TOTP screen) -> totp_mfa_setup; return secret."""
    client.signup(
        SignUpRequest(email=email, password=helpers.PASSWORD, confirm_password=helpers.PASSWORD)
    )
    login = client.login(LoginRequest(email=email, password=helpers.PASSWORD))
    # Brand-new user, MFA on by default, not yet enrolled -> mfaGateOfferAll:
    # token withheld, should_show_totp_screen true.
    assert login.should_show_totp_screen is True
    assert login.access_token is None

    setup = client.totp_mfa_setup(OtpMfaSetupRequest(email=email))
    assert setup.authenticator_secret
    return setup.authenticator_secret


def test_totp_enroll_and_complete_login(client: AuthorizerClient) -> None:
    email = helpers.random_email("totp")
    secret = _reach_totp_setup(client, email)

    code = pyotp.TOTP(secret).now()
    token = client.verify_otp(VerifyOTPRequest(email=email, otp=code, is_totp=True))

    assert token.access_token, "a valid TOTP code must complete the challenge"
    assert token.user is not None
    assert token.user.email == email


def test_totp_expired_code_rejected(client: AuthorizerClient) -> None:
    email = helpers.random_email("totp-expired")
    secret = _reach_totp_setup(client, email)

    # A code from 10 minutes ago is well outside the validator's ~90s skew.
    stale = pyotp.TOTP(secret).at(int(time.time()) - 600)

    with pytest.raises(AuthorizerError) as exc:
        client.verify_otp(VerifyOTPRequest(email=email, otp=stale, is_totp=True))
    assert "invalid otp" in exc.value.message.lower()
