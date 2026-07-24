"""Pytest fixtures: SDK client factories wired to the live e2e-playground stack.

Every test is marked ``live`` (see pyproject) — the whole suite talks to a
running stack. There is no mock of Authorizer itself.
"""

from __future__ import annotations

from collections.abc import Callable, Iterator

import httpx
import pytest
from authorizer import AuthorizerAdminClient, AuthorizerClient

import helpers


class _CookieShim:
    """Keeps the MFA session cookie flowing across SDK calls on single-label hosts.

    The MFA flow is cookie-based: login sets an ``mfa_session`` cookie that
    ``totp_mfa_setup`` / ``verify_otp`` / ``skip_mfa_setup`` read back. The
    server sets ``Domain=<request-host>``; in this docker stack that host is a
    single label (``authorizer``, ``authorizer-mfa-enforced``, ...). Two
    layers of Python's stdlib cookie handling then drop it, and neither is a
    product bug — both are artifacts of single-label hostnames that a real
    dotted domain (auth.example.com) and the browser jar the Playwright suite
    uses never hit:

      1. ``http.cookiejar`` rewrites a single-label request host to
         ``<host>.local`` (``eff_request_host``), so the stored ``Domain``
         stops matching at send time.
      2. httpx 0.28's ``_merge_cookies`` rebuilds a fresh *default-policy*
         jar per request (``Cookies(self.cookies)``), so any relaxed jar
         policy is discarded before the request is sent.

    Rather than fight both, this shim captures Set-Cookie off responses and
    re-injects a plain ``Cookie`` request header — exactly the bytes a browser
    would send — via httpx event hooks. It never overrides a ``Cookie`` header
    the caller set explicitly (so the social-login skip flow, which passes its
    own captured cookie, is left untouched).
    """

    def __init__(self) -> None:
        self._cookies: dict[str, str] = {}

    def on_response(self, response: httpx.Response) -> None:
        for name, value in response.cookies.items():
            self._cookies[name] = value

    def on_request(self, request: httpx.Request) -> None:
        if not self._cookies or "cookie" in request.headers:
            return
        request.headers["Cookie"] = "; ".join(f"{k}={v}" for k, v in self._cookies.items())


@pytest.fixture(scope="session")
def admin() -> Iterator[AuthorizerAdminClient]:
    """Session-wide admin client (x-authorizer-admin-secret) on the default instance."""
    c = AuthorizerAdminClient(
        authorizer_url=helpers.AUTHORIZER_BASE_URL,
        admin_secret=helpers.ADMIN_SECRET,
    )
    try:
        yield c
    finally:
        c.close()


@pytest.fixture
def make_client() -> Iterator[Callable[..., AuthorizerClient]]:
    """Factory for public SDK clients.

    Each call returns a fresh ``AuthorizerClient`` (its own httpx cookie jar —
    so a login's mfa_session cookie is replayed across subsequent setup/verify
    calls on the SAME client, exactly as the MFA flow requires). All clients
    handed out are closed at test teardown.
    """
    created: list[AuthorizerClient] = []

    def _make(base_url: str = helpers.AUTHORIZER_BASE_URL) -> AuthorizerClient:
        c = AuthorizerClient(client_id=helpers.CLIENT_ID, authorizer_url=base_url)
        # See _CookieShim: keep the mfa_session cookie flowing across the
        # login -> setup/verify calls despite single-label docker hosts.
        shim = _CookieShim()
        c._http.event_hooks["request"].append(shim.on_request)
        c._http.event_hooks["response"].append(shim.on_response)
        created.append(c)
        return c

    try:
        yield _make
    finally:
        for c in created:
            c.close()


@pytest.fixture
def client(make_client: Callable[..., AuthorizerClient]) -> AuthorizerClient:
    """A public SDK client on the default `authorizer` instance."""
    return make_client()
