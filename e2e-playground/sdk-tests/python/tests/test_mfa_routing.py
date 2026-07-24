"""MFA enforcement routing matrix — driven through authorizer-py.

Runs against the --enforce-mfa=true instances (authorizer-mfa-enforced and
authorizer-mfa-magic-link). Mirrors tests/mfa-routing-matrix.spec.ts.

SDK coverage:
  * password login under enforcement -> token withheld, routed to enrollment
  * skip_mfa_setup rejected under enforcement (can't be routed around)
  * magic_link_login initiation under enforcement

The one non-SDK step is following the magic-link email URL: that link is an
HTTP redirect endpoint (GET /verify_email) whose whole purpose is to 307 a
*browser* onward, carrying the mfa_required/mfa_gate query params this test
asserts. It is browser-facing, not SDK surface — so it's driven with a raw,
redirect-suppressed httpx GET, exactly as the Playwright suite does.
"""

from __future__ import annotations

from collections.abc import Callable

import httpx
import pytest
from authorizer import (
    AuthorizerClient,
    AuthorizerError,
    LoginRequest,
    MagicLinkLoginRequest,
    SignUpRequest,
    SkipMfaSetupRequest,
)

import helpers

pytestmark = pytest.mark.live

ClientFactory = Callable[..., AuthorizerClient]


def test_password_login_no_factor_withholds_token(make_client: ClientFactory) -> None:
    client = make_client(helpers.AUTHORIZER_MFA_ENFORCED_BASE_URL)
    email = helpers.random_email("mfa-matrix")
    client.signup(
        SignUpRequest(email=email, password=helpers.PASSWORD, confirm_password=helpers.PASSWORD)
    )

    login = client.login(LoginRequest(email=email, password=helpers.PASSWORD))
    # mfaGateBlockEnroll: EnforceMFA is absolute for an unenrolled user.
    assert login.access_token is None
    assert login.message == "Proceed to mfa setup"
    assert login.should_show_totp_screen is True


def test_skip_mfa_setup_rejected_under_enforcement(make_client: ClientFactory) -> None:
    client = make_client(helpers.AUTHORIZER_MFA_ENFORCED_BASE_URL)
    email = helpers.random_email("mfa-matrix-skip")
    client.signup(
        SignUpRequest(email=email, password=helpers.PASSWORD, confirm_password=helpers.PASSWORD)
    )
    # Login arms the mfa_session cookie on this client's jar; skip_mfa_setup
    # replays it — proving enforcement can't be bypassed even with a genuine
    # session, not just that login *offers* setup.
    client.login(LoginRequest(email=email, password=helpers.PASSWORD))

    with pytest.raises(AuthorizerError) as exc:
        client.skip_mfa_setup(SkipMfaSetupRequest(email=email))
    assert "cannot skip" in exc.value.message.lower()


def test_magic_link_login_routes_through_mfa_challenge(make_client: ClientFactory) -> None:
    base = helpers.AUTHORIZER_MFA_MAGIC_LINK_BASE_URL
    client = make_client(base)
    email = helpers.random_email("mfa-matrix-magic")

    # SDK initiates the magic-link login.
    client.magic_link_login(MagicLinkLoginRequest(email=email))

    link = helpers.wait_for_magic_link(email)

    # Non-SDK: the email link is a browser-facing redirect endpoint. Under
    # enforcement, VerifyEmailHandler routes an unenrolled user to the MFA
    # gate (mfa_required=1&mfa_gate=offer) instead of minting a token.
    with httpx.Client(follow_redirects=False, timeout=10.0) as raw:
        resp = raw.get(link)
    assert resp.status_code == 307
    location = resp.headers["location"]
    assert "access_token=" not in location
    assert "mfa_required=1" in location
    assert "mfa_gate=offer" in location
