"""Social OAuth (10 providers) — precise SDK vs. raw-HTTP split.

What the SDK does NOT wrap: login *initiation*. A social login is a browser
redirect — authorizer-react does ``window.location.href = /oauth_login/:provider``
— so there is no SDK method that starts it. The server-side ceremony
(``/oauth_login/:provider`` -> mock-oauth ``/authorize`` -> ``/token`` ->
``/userinfo`` -> ``/oauth_callback/:provider``) is therefore driven here with a
RAW httpx client following the redirect chain (no browser needed — mock-oauth
auto-approves and 302s straight back, the same technique
tests/oidc-provider.spec.ts's PKCE test and social/helpers.ts's consent-denied
path use). This exercises real server-side callback validation, code exchange,
userinfo fetch and user creation.

What the SDK DOES do here (its genuine, if minimal, role):
  * admin ``users`` query — verify the provider's profile claims were mapped
    onto a real stored user (given/family/nickname + signup_methods). This is
    the assertion that would catch a wire-shape drift in the admin users query.
  * ``skip_mfa_setup`` + ``validate_jwt_token`` + ``get_profile`` — for one
    representative provider, complete the withheld-MFA login and validate the
    resulting token THROUGH the SDK, proving the SDK parses a token minted for
    a social-originated session.

A brand-new social user lands on the withheld MFA-setup redirect
(``mfa_required=1&mfa_gate=offer``) — the callback issuing that redirect is
itself proof the whole exchange succeeded server-side.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any
from urllib.parse import parse_qs, urlparse

import httpx
import pytest
from authorizer import (
    AuthorizerAdminClient,
    ListUsersRequest,
    SkipMfaSetupRequest,
    TokenType,
    ValidateJWTTokenRequest,
)

import helpers

pytestmark = pytest.mark.live

BASE = helpers.AUTHORIZER_BASE_URL


@dataclass
class Case:
    provider: str
    profile: dict[str, Any]
    lookup_kind: str  # "email" | "nickname"
    lookup_value: str
    expected_given: str
    expected_family: str | None
    email: str | None


def _case(provider: str) -> Case:
    """Build a per-run provider case matching mock-oauth's expected profile shape."""
    uid = helpers.random_email(provider).split("@")[0]  # unique token per run
    email = f"{provider}-{uid}@example.com"
    p: dict[str, Any]
    if provider == "google":
        p = {"sub": f"google-{uid}", "email": email, "given_name": "Ada", "family_name": "Lovelace"}
        return Case(provider, p, "email", email, "Ada", "Lovelace", email)
    if provider == "github":
        p = {"name": "Grace Hopper", "email": email, "avatar_url": "https://example.com/a.png"}
        return Case(provider, p, "email", email, "Grace", "Hopper", email)
    if provider == "facebook":
        p = {
            "first_name": "Katherine",
            "last_name": "Johnson",
            "email": email,
            "picture": {"data": {"url": "https://example.com/a.png"}},
        }
        return Case(provider, p, "email", email, "Katherine", "Johnson", email)
    if provider == "linkedin":
        p = {"localizedFirstName": "Margaret", "localizedLastName": "Hamilton", "email": email}
        return Case(provider, p, "email", email, "Margaret", "Hamilton", email)
    if provider == "apple":
        p = {"sub": f"apple-{uid}", "email": email, "given_name": "Alan", "family_name": "Turing"}
        return Case(provider, p, "email", email, "Alan", "Turing", email)
    if provider == "discord":
        p = {"id": f"discord-{uid}", "username": "gracehopper", "avatar": "abc", "email": email}
        return Case(provider, p, "email", email, "gracehopper", None, email)
    if provider == "microsoft":
        p = {
            "sub": f"microsoft-{uid}",
            "email": email,
            "given_name": "Katherine",
            "family_name": "Johnson",
        }
        return Case(provider, p, "email", email, "Katherine", "Johnson", email)
    if provider == "twitch":
        p = {"sub": f"twitch-{uid}", "email": email, "given_name": "Sally", "family_name": "Ride"}
        return Case(provider, p, "email", email, "Sally", "Ride", email)
    if provider == "roblox":
        p = {
            "name": "Ada Lovelace",
            "nickname": "ada",
            "picture": "https://example.com/a.png",
            "email": email,
        }
        return Case(provider, p, "email", email, "Ada ", "Lovelace", email)
    if provider == "twitter":
        # Twitter/X never returns an email (real API parity) -> nickname lookup.
        nickname = f"ada-{uid}"
        p = {
            "data": {
                "id": f"twitter-{uid}",
                "name": "Ada Lovelace",
                "username": nickname,
                "profile_image_url": "https://example.com/a.png",
            }
        }
        return Case(provider, p, "nickname", nickname, "Ada ", "Lovelace", None)
    raise ValueError(provider)


def _collect_params(resp: httpx.Response) -> dict[str, str]:
    """Merge query params from every redirect Location and the final URL."""
    params: dict[str, str] = {}
    for hop in [*resp.history, resp]:
        for source in (hop.headers.get("location", ""), str(hop.url)):
            for k, v in parse_qs(urlparse(source).query).items():
                if v:
                    params[k] = v[0]
    return params


def _run_oauth_chain(c: httpx.Client, provider: str) -> dict[str, str]:
    """Drive the full server-side social login chain; return the final query params."""
    redirect_uri = f"{BASE}/app"
    resp = c.get(f"{BASE}/oauth_login/{provider}", params={"redirect_uri": redirect_uri})
    assert resp.status_code == 200, f"chain did not land on the app page: {resp.status_code}"
    return _collect_params(resp)


@pytest.mark.parametrize(
    "provider",
    [
        "google",
        "github",
        "facebook",
        "linkedin",
        "apple",
        "discord",
        "twitter",
        "microsoft",
        "twitch",
        "roblox",
    ],
)
def test_social_login_creates_mapped_user(provider: str, admin: AuthorizerAdminClient) -> None:
    case = _case(provider)
    helpers.configure_mock_profile(provider, case.profile)

    with httpx.Client(follow_redirects=True, timeout=20.0) as c:
        params = _run_oauth_chain(c, provider)

    # New user, MFA on by default -> callback withholds the token and routes to
    # the MFA-setup offer. That redirect existing at all proves the code
    # exchange + userinfo fetch + user creation all succeeded server-side.
    assert params.get("mfa_required") == "1"
    assert "access_token" not in params

    # SDK role: the provider's claims landed on a real stored user.
    users = admin.users(ListUsersRequest(query=case.lookup_value)).users
    if case.lookup_kind == "email":
        user = next((u for u in users if u.email == case.lookup_value), None)
    else:
        user = next((u for u in users if u.nickname == case.lookup_value), None)
    assert user is not None, f"{provider}: user not created / not found via admin SDK"
    assert user.given_name == case.expected_given
    if case.expected_family is not None:
        assert user.family_name == case.expected_family
    assert user.signup_methods is not None and provider in user.signup_methods


def test_social_login_token_validates_through_sdk(
    admin: AuthorizerAdminClient, make_client: Any
) -> None:
    """Complete a withheld social login via the SDK and validate the token via the SDK."""
    case = _case("google")
    helpers.configure_mock_profile("google", case.profile)

    with httpx.Client(follow_redirects=True, timeout=20.0) as c:
        params = _run_oauth_chain(c, "google")
        assert params.get("mfa_required") == "1"
        # The callback set the mfa_session cookie on this jar; hand it to the SDK.
        cookie_header = "; ".join(f"{k}={v}" for k, v in c.cookies.items())

    sdk = make_client()
    token = sdk.skip_mfa_setup(
        SkipMfaSetupRequest(email=case.email), headers={"Cookie": cookie_header}
    )
    assert token.access_token, "skip_mfa_setup must complete the social login"

    validated = sdk.validate_jwt_token(
        ValidateJWTTokenRequest(token=token.access_token, token_type=TokenType.ACCESS_TOKEN)
    )
    assert validated.is_valid is True

    profile = sdk.get_profile(headers={"Authorization": f"Bearer {token.access_token}"})
    assert profile.email == case.email
    assert profile.signup_methods is not None and "google" in profile.signup_methods


def test_social_consent_denied_rejected_and_state_single_use() -> None:
    """Raw-HTTP negative path: denied consent -> 400, and the state is single-use."""
    provider = "google"
    with httpx.Client(follow_redirects=False, timeout=10.0) as c:
        login = c.get(
            f"{BASE}/oauth_login/{provider}", params={"redirect_uri": f"{BASE}/app"}
        )
        assert login.status_code == 307
        state = parse_qs(urlparse(login.headers["location"]).query)["state"][0]

        denied = c.get(
            f"{BASE}/oauth_callback/{provider}",
            params={"error": "access_denied", "state": state},
        )
        assert denied.status_code == 400
        assert denied.json()["error"] == "invalid oauth code"

        # State is single-use — a replay is rejected differently (unknown state).
        replay = c.get(
            f"{BASE}/oauth_callback/{provider}",
            params={"error": "access_denied", "state": state},
        )
        assert replay.status_code == 400
        assert replay.json()["error"] == "invalid oauth state"
