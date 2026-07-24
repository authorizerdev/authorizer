"""WebAuthn / Passkey — FULL ceremony through authorizer-py + a software authenticator.

Unlike the Playwright suite (which uses Chrome's CDP virtual authenticator),
Python has no built-in authenticator on the calling side. We use the
``soft-webauthn`` package's ``SoftWebauthnDevice`` — a real software FIDO2
authenticator — to actually create the attestation and sign the assertion.
Every server interaction goes through the SDK's own webauthn_* methods; the
only glue code is the WebAuthn wire codec (base64url <-> bytes) a browser would
otherwise handle, which is exactly the layer most prone to SDK/server drift and
therefore the most valuable to exercise.

Runs against the dedicated authorizer-webauthn instance: go-webauthn's RPID
validation requires a dotted hostname, so the RP origin/ID must be
``webauthn.e2e-playground.test`` (reachable in-network via that alias). The
soft authenticator signs clientDataJSON against that same origin.
"""

from __future__ import annotations

import json
from base64 import urlsafe_b64encode
from struct import pack
from typing import Any, Callable

import pytest
from authorizer import (
    AuthorizerClient,
    LoginRequest,
    SignUpRequest,
    SkipMfaSetupRequest,
    TokenType,
    ValidateJWTTokenRequest,
    WebauthnLoginVerifyRequest,
    WebauthnRegistrationVerifyRequest,
)
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec
from fido2 import cbor
from fido2.cose import ES256
from fido2.utils import sha256
from soft_webauthn import SoftWebauthnDevice

import helpers

pytestmark = pytest.mark.live

BASE = helpers.AUTHORIZER_WEBAUTHN_BASE_URL
ORIGIN = BASE  # RP origin == the instance's --url


class UvSoftWebauthnDevice(SoftWebauthnDevice):  # type: ignore[misc]  # untyped base
    """soft-webauthn, but with the User-Verified (UV) flag set.

    Authorizer's RP is configured ``UserVerification: VerificationRequired``
    (internal/authenticators/webauthn/webauthn.go newRP), so go-webauthn
    rejects any attestation/assertion whose authenticator-data UV flag is
    unset. Stock ``SoftWebauthnDevice`` hardcodes UP-only flags (0x41 create /
    0x01 get). This overrides both to also set UV (0x45 / 0x05) — the software
    equivalent of the Playwright CDP authenticator's ``isUserVerified: true``.
    The bodies are otherwise identical to the upstream methods.
    """

    def create(self, options: dict[str, Any], origin: str) -> dict[str, Any]:
        if {"alg": -7, "type": "public-key"} not in options["publicKey"]["pubKeyCredParams"]:
            raise ValueError("Requested pubKeyCredParams does not contain supported type")
        self.cred_init(options["publicKey"]["rp"]["id"], options["publicKey"]["user"]["id"])
        client_data = {
            "type": "webauthn.create",
            "challenge": urlsafe_b64encode(options["publicKey"]["challenge"])
            .decode("ascii")
            .rstrip("="),
            "origin": origin,
        }
        rp_id_hash = sha256(self.rp_id.encode("ascii"))
        flags = b"\x45"  # UP | UV | AT
        pub = ES256.from_cryptography_key(self.private_key.public_key())  # type: ignore[no-untyped-call]
        cose_key = cbor.encode(pub)
        auth_data = (
            rp_id_hash
            + flags
            + pack(">I", self.sign_count)
            + self.aaguid
            + pack(">H", len(self.credential_id))
            + self.credential_id
            + cose_key
        )
        attestation_object = {"authData": auth_data, "fmt": "none", "attStmt": {}}
        return {
            "id": urlsafe_b64encode(self.credential_id),
            "rawId": self.credential_id,
            "response": {
                "clientDataJSON": json.dumps(client_data).encode("utf-8"),
                "attestationObject": cbor.encode(attestation_object),
            },
            "type": "public-key",
        }

    def get(self, options: dict[str, Any], origin: str) -> dict[str, Any]:
        if self.rp_id != options["publicKey"]["rpId"]:
            raise ValueError("Requested rpID does not match current credential")
        self.sign_count += 1
        client_data = json.dumps(
            {
                "type": "webauthn.get",
                "challenge": urlsafe_b64encode(options["publicKey"]["challenge"])
                .decode("ascii")
                .rstrip("="),
                "origin": origin,
            }
        ).encode("utf-8")
        rp_id_hash = sha256(self.rp_id.encode("ascii"))
        flags = b"\x05"  # UP | UV
        authenticator_data = rp_id_hash + flags + pack(">I", self.sign_count)
        signature = self.private_key.sign(
            authenticator_data + sha256(client_data), ec.ECDSA(hashes.SHA256())
        )
        return {
            "id": urlsafe_b64encode(self.credential_id),
            "rawId": self.credential_id,
            "response": {
                "authenticatorData": authenticator_data,
                "clientDataJSON": client_data,
                "signature": signature,
                "userHandle": self.user_handle,
            },
            "type": "public-key",
        }


def _prepare_creation_options(options_json: str) -> dict[str, Any]:
    """Server creation options (JSON, base64url) -> soft_webauthn input (bytes)."""
    pk = json.loads(options_json)
    pk["challenge"] = helpers.b64url_decode(pk["challenge"])
    pk["user"]["id"] = helpers.b64url_decode(pk["user"]["id"])
    for cred in pk.get("excludeCredentials", []) or []:
        cred["id"] = helpers.b64url_decode(cred["id"])
    # soft_webauthn only produces 'none' attestation; drop any other request.
    pk.pop("attestation", None)
    return {"publicKey": pk}


def _prepare_request_options(options_json: str) -> dict[str, Any]:
    """Server request (login) options (JSON, base64url) -> soft_webauthn input (bytes)."""
    pk = json.loads(options_json)
    pk["challenge"] = helpers.b64url_decode(pk["challenge"])
    for cred in pk.get("allowCredentials", []) or []:
        cred["id"] = helpers.b64url_decode(cred["id"])
    return {"publicKey": pk}


def _attestation_to_json(att: dict[str, Any]) -> str:
    return json.dumps(
        {
            "id": helpers.b64url_encode(att["rawId"]),
            "rawId": helpers.b64url_encode(att["rawId"]),
            "type": "public-key",
            "response": {
                "clientDataJSON": helpers.b64url_encode(att["response"]["clientDataJSON"]),
                "attestationObject": helpers.b64url_encode(att["response"]["attestationObject"]),
            },
        }
    )


def _assertion_to_json(assertion: dict[str, Any]) -> str:
    r = assertion["response"]
    return json.dumps(
        {
            "id": helpers.b64url_encode(assertion["rawId"]),
            "rawId": helpers.b64url_encode(assertion["rawId"]),
            "type": "public-key",
            "response": {
                "authenticatorData": helpers.b64url_encode(r["authenticatorData"]),
                "clientDataJSON": helpers.b64url_encode(r["clientDataJSON"]),
                "signature": helpers.b64url_encode(r["signature"]),
                "userHandle": helpers.b64url_encode(r["userHandle"]),
            },
        }
    )


def _authenticated_token(client: AuthorizerClient, email: str) -> str:
    """signup -> login (withheld) -> skip MFA -> a real bearer access token."""
    client.signup(
        SignUpRequest(email=email, password=helpers.PASSWORD, confirm_password=helpers.PASSWORD)
    )
    client.login(LoginRequest(email=email, password=helpers.PASSWORD))
    token = client.skip_mfa_setup(SkipMfaSetupRequest(email=email))
    assert token.access_token
    return token.access_token


def test_webauthn_full_registration_and_passwordless_login(
    make_client: Callable[..., AuthorizerClient],
) -> None:
    email = helpers.random_email("webauthn")
    device = UvSoftWebauthnDevice()

    # --- register a passkey (settings-page path: bearer-token authenticated) --- #
    reg_client = make_client(BASE)
    access_token = _authenticated_token(reg_client, email)
    auth = {"Authorization": f"Bearer {access_token}"}

    reg_options = reg_client.webauthn_registration_options(headers=auth)
    attestation = device.create(_prepare_creation_options(reg_options.options), ORIGIN)
    reg_client.webauthn_registration_verify(
        WebauthnRegistrationVerifyRequest(
            credential=_attestation_to_json(attestation), name="e2e-passkey"
        ),
        headers=auth,
    )

    creds = reg_client.webauthn_credentials(headers=auth)
    assert len(creds) == 1
    assert creds[0].name == "e2e-passkey"

    # --- log back in with ONLY the passkey (usernameless / discoverable) --- #
    login_client = make_client(BASE)
    login_options = login_client.webauthn_login_options()  # no email -> discoverable
    assertion = device.get(_prepare_request_options(login_options.options), ORIGIN)
    token = login_client.webauthn_login_verify(
        WebauthnLoginVerifyRequest(credential=_assertion_to_json(assertion))
    )

    assert token.access_token, "passkey-only login must issue a token"
    assert token.user is not None and token.user.email == email

    validated = login_client.validate_jwt_token(
        ValidateJWTTokenRequest(token=token.access_token, token_type=TokenType.ACCESS_TOKEN)
    )
    assert validated.is_valid is True
