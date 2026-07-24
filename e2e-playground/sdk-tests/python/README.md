# Python SDK e2e suite

Live e2e tests that drive the `e2e-playground` stack's enterprise/MFA feature
set **through the published `authorizer-py` SDK** (PyPI `authorizer-py==0.3.0rc3`),
not raw GraphQL/REST. The sibling `../../tests/*.spec.ts` Playwright suite
already covers these features at the wire level and through a browser; this
suite re-drives the same flows through the SDK's own methods to catch
SDK↔server wire-shape drift (pagination shapes, stale/renamed fields, request
encodings) that a raw-HTTP or browser test would never surface.

It depends on the **published** SDK as a normal dependency — no local/editable
install. That is the point: it exercises exactly what a consumer's `pip install
authorizer-py` gets.

## Why it runs inside the compose network

Like the `playwright` service, this suite runs as a compose service
(`python-sdk`) on the docker-internal network. Two feature areas require it:

- **Social OAuth** — the server redirects the login chain to `mock-oauth:4000`
  (a docker-internal hostname), so the redirect-follow must resolve it.
- **WebAuthn** — go-webauthn validates the RP origin against the instance's
  `--url` (`http://webauthn.e2e-playground.test:8080`); the software
  authenticator must sign against that exact origin.

Everything else (TOTP, SMS-OTP, MFA routing, SCIM, SAML admin, OTP lockout)
would also run against host-exposed ports, but one runner keeps it uniform.

## Run

From the repository root, with the stack up (see `../../README.md`):

```bash
docker compose -f e2e-playground/docker-compose.yml up -d --wait \
  authorizer authorizer-webauthn authorizer-mfa-enforced authorizer-mfa-magic-link \
  mock-oauth sms-sink webhook-sink mailpit
docker compose -f e2e-playground/docker-compose.yml run --rm --build python-sdk
docker compose -f e2e-playground/docker-compose.yml down -v
```

`--build` is required after any change under `sdk-tests/python/` (test files
are baked into the image, same as the Playwright runner).

Subset / single file:

```bash
docker compose -f e2e-playground/docker-compose.yml run --rm --build python-sdk tests/test_totp.py -v
```

## Lint / type-check

Config in `pyproject.toml` mirrors `authorizer-py`'s own (ruff line-length 100,
mypy strict):

```bash
pip install -e '.[dev]'
ruff check .
mypy .
```

## Coverage & the SDK/architecture boundary

| Area | Through the SDK | Raw HTTP (labeled) — and why |
|---|---|---|
| TOTP | signup/login/`totp_mfa_setup`/`verify_otp` | — |
| SMS-OTP | signup/login/`sms_otp_mfa_setup`/`verify_otp` | read code from `sms-sink` (delivery sink) |
| MFA routing | login/`skip_mfa_setup`/`magic_link_login` | follow the magic-link email URL (browser redirect endpoint) |
| SCIM | endpoint CRUD, token rotation, `add_webhook`, `webhook_logs` | `/scim/v2/*` protocol (inbound RFC 7644 REST — not SDK surface) |
| Social OAuth | admin `users` verify; `skip_mfa_setup`+`validate_jwt_token`+`get_profile` | the `/oauth_login`→callback redirect chain (login initiation is a browser redirect, not wrapped by the SDK) |
| SAML | SP registry CRUD, IdP key rotate/retire, metadata import | — (the SSO ceremony is browser + HTML-form-POST; out of scope by construction) |
| WebAuthn | full ceremony via `webauthn_*` + `soft-webauthn` authenticator | — |
| OTP lockout | 5× `verify_otp` → `AuthorizerError` lockout message | read code from `sms-sink` |

WebOTP has no distinct server flow — it is the browser WebOTP autofill reading
the same SMS-OTP code; the wire flow is covered by the SMS-OTP tests, the
autofill itself is browser-only.
