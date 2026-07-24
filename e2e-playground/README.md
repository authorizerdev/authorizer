# Live-Playground E2E Suite

A real, live-server end-to-end test suite for Authorizer. Every spec drives a real `authorizer` binary (built from this repo's source) plus a real browser (Playwright) — no unit-level mocking of Authorizer itself. Third-party services (Google/GitHub/Discord/etc. OAuth, SMS delivery, an external SAML IdP) are stood in for by small local mock servers under `mocks/`, wired in via `authorizer`'s own test-only CLI overrides (`--test-oauth-*-base-url`, `--test-sms-webhook-url`, `--test-allow-private-*-hosts`) — every one of those flags is a documented no-op when unset, so none of this affects production behavior.

## Coverage

- OIDC provider (signup/login/PKCE/token issuance) and OIDC SSO relying-party (home-realm discovery)
- SAML SP and SAML IdP
- SCIM (CRUD, filter operators, webhooks — including real webhook delivery with HMAC verification)
- 10 social OAuth providers (Google, GitHub, Facebook, LinkedIn, Apple, Discord, Twitter, Microsoft, Twitch, Roblox)
- WebAuthn/passkeys (via Playwright's CDP-backed virtual authenticator)
- TOTP, SMS-OTP, WebOTP auto-fill, magic-link
- MFA enforcement routing matrix
- OTP brute-force lockout
- Dashboard verified-domains UI

## Prerequisites

- Docker + Docker Compose (everything else — Node, Playwright browsers, the `authorizer` binary — is built into containers; nothing needs to be installed on the host).

## Run the full suite

From the repository root:

```bash
make e2e-playground
```

This builds and starts the full docker-compose stack (six `authorizer` instances configured for different scenarios, the mock OAuth/SAML/SMS/webhook servers, Mailpit for email), runs the entire Playwright suite in a containerized runner, and **always** tears the stack down afterward (`docker compose down -v`) — including on failure or if the stack fails to start.

Equivalent to, run manually:

```bash
docker compose -f e2e-playground/docker-compose.yml up -d --wait authorizer authorizer-sso mock-oauth mock-saml-idp mailpit sms-sink
docker compose -f e2e-playground/docker-compose.yml run --rm playwright npx playwright test
docker compose -f e2e-playground/docker-compose.yml down -v
```

(The `playwright` service's own `depends_on` brings up the remaining instances — `authorizer-webauthn`, `authorizer-magic-link`, `authorizer-mfa-enforced`, `authorizer-mfa-magic-link`, `webhook-sink` — automatically; they don't need to be named in the `up` step.)

## Run a subset

Any Playwright CLI filter works. From `e2e-playground/`, after starting the stack:

```bash
# One spec file
docker compose run --rm --build playwright npx playwright test totp.spec.ts

# By name pattern
docker compose run --rm --build playwright npx playwright test -g "SCIM"

# A whole directory
docker compose run --rm --build playwright npx playwright test social/
```

**`--build` is required whenever a spec file (or anything else under `e2e-playground/`) has changed** — the `playwright` service's Dockerfile bakes test files into the image at build time rather than mounting them live; without `--build` you'll run stale tests, or Playwright will report "No tests found" if a new file was added since the image was last built.

Don't forget to tear down afterward: `docker compose -f e2e-playground/docker-compose.yml down -v`.

## Verify results after a run

Playwright's HTML report is written to `e2e-playground/playwright-report/` on the host (mounted, survives container teardown). To view it:

```bash
npx playwright show-report e2e-playground/playwright-report
```

or open `e2e-playground/playwright-report/index.html` directly in a browser. It shows every test's pass/fail status, timing, and — for failures — a full trace (screenshots, network log, action-by-action replay).

Raw output (traces, videos on retry, etc.) is under `e2e-playground/test-results/`, also host-mounted.

A clean run's terminal output ends with a summary line, e.g.:

```
XX passed (NNs)
```

Any skip is intentional and self-documenting — read the `test.skip(...)` call's message in the relevant spec file for the reason. As of this writing there are no skips tied to real product gaps.

## Notes

- Every spec seeds its own state (org, users, connections) — specs are independently runnable and safe to run in parallel or in any order.
- `authorizer-sso` runs a second, separately-configured instance (port 8081) specifically because `--enable-org-discovery=true` is a global login-UX toggle that would otherwise change behavior for every other spec sharing the default instance.
- Test-only secrets (SAML certs, JWT signing keys) live under `fixtures/certs/`, are checked in, and sign nothing real — the same pattern `make dev`'s embedded dev RSA keys already use.
