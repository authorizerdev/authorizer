"""Shared configuration and mock-polling helpers for the SDK e2e suite.

Everything here is deliberately transport-level (raw ``httpx``): these are the
parts a real integration exercises around the SDK — reading a delivered SMS
from ``sms-sink``, a magic-link email from Mailpit, a webhook from
``webhook-sink`` — none of which the ``authorizer-py`` SDK wraps (nor should
it; they are third-party sinks). Anything that *is* SDK surface goes through
the SDK in the test files, not here.
"""

from __future__ import annotations

import base64
import os
import re
import time
import uuid
from typing import Any

import httpx

# --- base URLs -------------------------------------------------------------- #
# Host-port defaults let the suite run against a locally-exposed stack; the
# compose `python-sdk` service overrides each with the docker-internal
# hostname (mirroring the `playwright` service's env block) so social-OAuth
# and WebAuthn resolve/validate correctly.


def _env(name: str, default: str) -> str:
    return os.environ.get(name, default)


AUTHORIZER_BASE_URL = _env("AUTHORIZER_BASE_URL", "http://localhost:8080")
AUTHORIZER_MFA_ENFORCED_BASE_URL = _env("AUTHORIZER_MFA_ENFORCED_BASE_URL", "http://localhost:8084")
AUTHORIZER_MFA_MAGIC_LINK_BASE_URL = _env(
    "AUTHORIZER_MFA_MAGIC_LINK_BASE_URL", "http://localhost:8085"
)
AUTHORIZER_WEBAUTHN_BASE_URL = _env("AUTHORIZER_WEBAUTHN_BASE_URL", "http://localhost:8082")

MOCK_OAUTH_BASE_URL = _env("MOCK_OAUTH_BASE_URL", "http://localhost:4000")
SMS_SINK_BASE_URL = _env("SMS_SINK_BASE_URL", "http://localhost:4100")
WEBHOOK_SINK_BASE_URL = _env("WEBHOOK_SINK_BASE_URL", "http://localhost:4200")
MAILPIT_BASE_URL = _env("MAILPIT_BASE_URL", "http://localhost:8025")

ADMIN_SECRET = _env("AUTHORIZER_ADMIN_SECRET", "e2e-admin-secret")
CLIENT_ID = _env("AUTHORIZER_CLIENT_ID", "e2e-client-id")
# Matches --client-secret in docker-compose.yml; webhook HMAC signing key.
CLIENT_SECRET = _env("AUTHORIZER_CLIENT_SECRET", "e2e-client-secret")

PASSWORD = "Str0ngPassw0rd!"


# --- random identities ------------------------------------------------------ #
def random_email(prefix: str) -> str:
    return f"{prefix}-{uuid.uuid4()}@example.com"


def random_phone() -> str:
    # E.164-ish, wide enough range to avoid collisions across runs.
    return f"+1555{uuid.uuid4().int % 9000000 + 1000000}"


# --- OTP extraction --------------------------------------------------------- #
# utils.GenerateOTP draws from "ABCDEFGHJKLMNPQRSTUVWXYZ123456789" (ambiguous
# I/O/0/1 excluded) — a naive \d{6} would never match, so split on the fixed
# "code is: " prefix, charset-agnostic. Same reasoning as the Playwright suite.
_OTP_RE = re.compile(r"code is:\s*([A-Z0-9]{6})")


def extract_otp(message: str) -> str:
    m = _OTP_RE.search(message)
    if not m:
        raise AssertionError(f"could not find OTP in SMS body: {message!r}")
    return m.group(1)


# --- mock polling ----------------------------------------------------------- #
def wait_for_sms(phone: str, *, attempts: int = 40, delay: float = 0.25) -> str:
    """Poll sms-sink's GET /sms/:phone until a message lands; return its body."""
    url = f"{SMS_SINK_BASE_URL}/sms/{phone}"
    with httpx.Client(timeout=5.0) as c:
        for _ in range(attempts):
            r = c.get(url)
            if r.status_code == 200:
                return str(r.json()["message"])
            time.sleep(delay)
    raise AssertionError(f"no SMS received for {phone} within timeout")


def wait_for_magic_link(email: str, *, attempts: int = 40, delay: float = 0.25) -> str:
    """Poll Mailpit for a verify_email magic link addressed to *email*."""
    link_re = re.compile(r"(https?://\S+verify_email\?\S+)")
    with httpx.Client(timeout=5.0) as c:
        for _ in range(attempts):
            msgs = c.get(f"{MAILPIT_BASE_URL}/api/v1/messages").json().get("messages", [])
            match = next(
                (m for m in msgs if any(t["Address"] == email for t in m["To"])),
                None,
            )
            if match:
                detail = c.get(f"{MAILPIT_BASE_URL}/api/v1/message/{match['ID']}").json()
                found = link_re.search(detail.get("Text", ""))
                if found:
                    return found.group(1).rstrip(")")
            time.sleep(delay)
    raise AssertionError(f"no magic-link email received for {email} within timeout")


def wait_for_webhook_events(
    email: str, *, attempts: int = 40, delay: float = 0.25
) -> dict[str, dict[str, Any]]:
    """Poll webhook-sink's GET /webhook/:email; return its keyed events map."""
    url = f"{WEBHOOK_SINK_BASE_URL}/webhook/{email}"
    with httpx.Client(timeout=5.0) as c:
        for _ in range(attempts):
            r = c.get(url)
            if r.status_code == 200:
                events: dict[str, dict[str, Any]] = r.json().get("events", {})
                if events:
                    return events
            time.sleep(delay)
    raise AssertionError(f"no webhook deliveries for {email} within timeout")


def configure_mock_profile(provider: str, profile: dict[str, Any]) -> None:
    """Set the profile mock-oauth returns for the next login against *provider*."""
    with httpx.Client(timeout=5.0) as c:
        r = c.post(f"{MOCK_OAUTH_BASE_URL}/{provider}/__configure", json={"profile": profile})
        if r.status_code != 204:
            raise AssertionError(
                f"failed to configure mock profile for {provider}: {r.status_code}"
            )


# --- base64url (WebAuthn wire codec, matches go-webauthn RawURLEncoding) ----- #
def b64url_decode(data: str) -> bytes:
    return base64.urlsafe_b64decode(data + "=" * (-len(data) % 4))


def b64url_encode(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).decode("ascii").rstrip("=")
