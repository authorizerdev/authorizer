# SDK-driven e2e-playground suite (Go)

A second test surface alongside the Playwright suite in `../../tests`. Where the
Playwright specs drive the same live stack via **raw GraphQL/REST + a browser**,
this suite drives it through the **actually-published `authorizer-go` SDK**
(`github.com/authorizerdev/authorizer-go/v2@v2.2.0-rc.4`, a real `go get`
dependency — no `replace` onto local source). The point is to catch wire-shape
drift between what the SDK sends/parses and what the server does, in feature
areas the SDK's own integration suite never touches.

## How to run

Inside the compose network (canonical — WebAuthn's pinned origin and the
docker-internal hostnames only resolve here):

```bash
make e2e-playground-sdk          # from repo root: up --build → run → down -v (always)
```

Equivalent manual invocation:

```bash
docker compose -f e2e-playground/docker-compose.yml up -d --wait --build \
  authorizer authorizer-webauthn authorizer-mfa-enforced authorizer-mfa-magic-link \
  mock-oauth mock-saml-idp mailpit sms-sink webhook-sink
docker compose -f e2e-playground/docker-compose.yml run --rm --build go-sdk-tests
docker compose -f e2e-playground/docker-compose.yml down -v
```

The suite also runs from the host (`go test ./...`) against the published ports
for everything **except** WebAuthn, which skips off-network (see below).

## What is genuinely SDK-driven, and what is not

| Area | Verdict | How |
|---|---|---|
| **MFA enforcement routing** | ✅ fully SDK | `Login` / `SkipMfaSetup` typed methods assert the withheld-token response shape (`mfa_routing_test.go`). |
| **SAML IdP admin** | ✅ fully SDK | SP CRUD + IdP signing-key rotation/listing via the admin client's typed proto methods (`saml_idp_admin_test.go`). |
| **SCIM admin + provisioning webhooks** | ✅ SDK admin + raw SCIM | Org / SCIM-endpoint / token-rotation / webhook registration through the SDK admin client; the RFC 7644 `/scim/v2/Users` CRUD itself stays raw HTTP **by design** (it is a standardized protocol an IdP hits directly, not an SDK concern) (`scim_test.go`). |
| **WebAuthn registration** | ✅ full ceremony via SDK | `WebauthnRegistrationOptions`/`Verify` (typed, cookie threaded) completed by a pure-Go software authenticator (`descope/virtualwebauthn`) — no browser (`webauthn_test.go`). |
| **WebAuthn login** | ⚠️ options only | `WebauthnLoginOptions` shape asserted via SDK; completion blocked by the cookie gap below. |
| **TOTP** | ⚠️ SDK setup + raw verify | `TotpMfaSetup` via SDK (cookie threaded); `verify_otp` + lockout via raw jar client (cookie gap) (`totp_test.go`, `otp_lockout_test.go`). |
| **SMS-OTP / WebOTP** | ⚠️ SDK setup + raw verify | `SmsOtpMfaSetup` via SDK (triggers the real code to `sms-sink`); `verify_otp` + lockout raw (`sms_otp_test.go`, `otp_lockout_test.go`). |
| **OTP brute-force lockout** | ⚠️ raw verify | 5 wrong `verify_otp` → distinct lockout error; enrollment via SDK (`otp_lockout_test.go`). |
| **Social OAuth (10 providers)** | ❌ not SDK-testable | The SDK has **no** login-initiation surface — social login is a `/oauth_login/:provider` browser redirect chain. There is nothing for the SDK to drive; it belongs to the Playwright suite (`tests/social/*`). |
| **SAML SP login (assertion flow)** | ❌ not SDK-testable | AuthnRequest redirect + IdP form-POST + signed assertion to the SP ACS is inherently a browser/form flow with no SDK surface. The SP *config* is admin-API and lives in the SAML IdP admin test; the *login* stays in Playwright (`tests/saml-sp.spec.ts`). |

## Confirmed SDK gaps (v2.2.0-rc.4) — follow-up recommendations

These are real, not test-harness limitations. The MFA-session flow is
cookie-based server-side (`mfa_session` cookie, required unconditionally by
`verify_otp` and by the OTP/WebAuthn setup calls), but the SDK cannot carry it
end to end:

1. **`Login` (and `SignUp`) discard `Set-Cookie`.** The typed methods return only
   the parsed body, so the `mfa_session` cookie a login arms is unreachable
   through the SDK. A caller cannot obtain it to continue the withheld-token MFA
   challenge. *Suggested fix:* expose response cookies (or an opt-in cookie jar)
   on the client.
2. **`VerifyOTP` and `WebauthnLoginVerify` accept no per-call headers.** Even with
   the cookie in hand, there is no way to send it on the one call that most needs
   it. (By contrast `TotpMfaSetup`, `SmsOtpMfaSetup`, and the WebAuthn
   registration methods *do* take a `headers map[string]string`, which is why the
   setup half of each flow is genuinely SDK-driven here.) *Suggested fix:* add a
   `headers` parameter to `VerifyOTP` / `WebauthnLoginVerify`, matching the setup
   methods.

Where a gap forced raw HTTP, it is a single labelled call (a cookie-jar
`verify_otp` / login), never a silent workaround — see the per-file headers.
Every step that *can* go through the SDK does.
