# Enterprise OIDC IdP Integration Guide

Authorizer can act as an **Enterprise OpenID Connect Identity Provider (IdP)** for platforms like Auth0, Okta, and Keycloak. This document covers the technical details, configuration, and known requirements for this integration.

---

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Auth0 Configuration](#auth0-configuration)
- [Okta Configuration](#okta-configuration)
- [Keycloak Configuration](#keycloak-configuration)
- [OIDC Endpoints](#oidc-endpoints)
- [Authorization Code Flow with PKCE](#authorization-code-flow-with-pkce)
- [Nonce Handling](#nonce-handling)
- [Response Modes](#response-modes)
- [Prompt Parameter](#prompt-parameter)
- [Session Management](#session-management)
- [Troubleshooting](#troubleshooting)
- [RFC Compliance](#rfc-compliance)

---

## Overview

When configured as an Enterprise OIDC IdP, Authorizer handles:

1. **Discovery** — RP reads `/.well-known/openid-configuration` to discover endpoints
2. **Authorization** — RP redirects users to `/authorize` with OIDC parameters
3. **Authentication** — Authorizer shows its login UI, user authenticates
4. **Code Exchange** — RP exchanges the authorization code at `/oauth/token`
5. **UserInfo** — RP optionally fetches user claims from `/userinfo`

## How It Works

```
┌─────────┐     1. /authorize?client_id=...&nonce=...     ┌────────────┐
│  Auth0   │ ─────────────────────────────────────────────▶│ Authorizer │
│  (RP)    │                                               │   (IdP)    │
│          │     2. User authenticates at /app              │            │
│          │                                               │            │
│          │     3. 302 redirect_uri?code=...&state=...    │            │
│          │ ◀─────────────────────────────────────────────│            │
│          │                                               │            │
│          │     4. POST /oauth/token (code + verifier)    │            │
│          │ ─────────────────────────────────────────────▶│            │
│          │                                               │            │
│          │     5. { access_token, id_token }             │            │
│          │ ◀─────────────────────────────────────────────│            │
└─────────┘                                               └────────────┘
```

### Key Parameters Forwarded Through Login UI

When the user is not yet authenticated, Authorizer redirects to its login UI (`/app`). The following parameters are forwarded via the `authState` URL so the React login app can send them back to `/authorize` after authentication:

| Parameter | Purpose | Required |
|-----------|---------|----------|
| `state` | CSRF protection, RP state preservation | Yes |
| `nonce` | Replay protection — echoed in `id_token` | Required for backchannel (code) flow by most RPs |
| `scope` | Requested claims (`openid profile email`) | Yes |
| `redirect_uri` | RP's callback URL | Yes |
| `response_type` | Flow type (`code`, `id_token`, etc.) | Yes |
| `response_mode` | Delivery method (`query`, `fragment`, `form_post`, `web_message`) | Yes |
| `client_id` | RP's client identifier | Yes |
| `code_challenge` | PKCE challenge (hash of verifier) | When PKCE is used |
| `code_challenge_method` | `S256` or `plain` | When PKCE is used |
| `login_hint` | Pre-fill email in login form | Optional |
| `ui_locales` | Preferred UI language | Optional |

**Important:** `prompt` is intentionally NOT forwarded to prevent redirect loops. See [Prompt Parameter](#prompt-parameter).

---

## Auth0 Configuration

### Setting Up Enterprise Connection

1. In Auth0 Dashboard → **Authentication** → **Enterprise** → **OpenID Connect**
2. Create new connection with:
   - **Issuer URL**: `https://your-authorizer-domain.com`
   - **Client ID**: Your Authorizer `client_id`
   - **Client Secret**: Your Authorizer `client_secret`
   - **Scopes**: `openid profile email`
3. Auth0 auto-discovers endpoints from `/.well-known/openid-configuration`

### Auth0-Specific Behavior

- **PKCE**: Auth0 may send `code_challenge` without `code_challenge_method`. Per RFC 7636 §4.2, Authorizer defaults to `plain` when the method is absent.
- **Nonce**: Auth0 **always** sends a `nonce` parameter, even for the `code` flow (where OIDC Core makes it optional). Authorizer must echo this nonce back in the `id_token`.
- **Response Mode**: Auth0 uses `form_post` for the authorization response. Authorizer sets a CSP header that allows the form to POST to Auth0's callback URL.
- **Token Exchange**: Auth0's server calls `/oauth/token` with the authorization code. This is a server-to-server call — Authorizer does not set a browser cookie on this response.

### Testing

Use Auth0's connection tester:
```
https://YOUR_TENANT.us.auth0.com/authorize?client_id=YOUR_CLIENT_ID&response_type=code&connection=YOUR_CONNECTION_NAME&prompt=login&scope=openid%20profile%20email&redirect_uri=https://YOUR_TENANT.us.auth0.com/login/callback
```

---

## Okta Configuration

1. In Okta Admin → **Security** → **Identity Providers** → **Add Identity Provider** → **OpenID Connect**
2. Configure:
   - **Issuer**: `https://your-authorizer-domain.com`
   - **Client ID / Secret**: Authorizer credentials
   - **Scopes**: `openid profile email`
3. Okta reads the discovery document automatically

### Okta-Specific Behavior

- Okta typically uses S256 PKCE and sends `code_challenge_method=S256` explicitly
- Okta validates the `auth_time` claim in the `id_token` when `max_age` is sent

---

## Keycloak Configuration

1. In Keycloak Admin → **Identity Providers** → **OpenID Connect v1.0**
2. Configure:
   - **Authorization URL**: Discovered from `/.well-known/openid-configuration`
   - **Token URL**: Discovered automatically
   - **Client ID / Secret**: Authorizer credentials
3. Enable **PKCE** in the provider settings (recommended)

---

## OIDC Endpoints

All endpoints are discoverable via `/.well-known/openid-configuration`.

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/.well-known/openid-configuration` | GET | OIDC Discovery document |
| `/.well-known/jwks.json` | GET | Public signing keys (JWKS) |
| `/authorize` | GET | Authorization endpoint |
| `/oauth/token` | POST | Token endpoint |
| `/userinfo` | GET | UserInfo endpoint (Bearer token) |
| `/logout` | GET/POST | RP-Initiated Logout |
| `/oauth/revoke` | POST | Token Revocation (RFC 7009) |
| `/oauth/introspect` | POST | Token Introspection (RFC 7662) |

### Endpoint Middleware

- `/.well-known/*` and `/authorize` — exempt from `client_id` header middleware (client_id is passed as query/body param)
- `/oauth/token`, `/oauth/revoke`, `/oauth/introspect` — exempt from CSRF middleware (use bearer/client credentials, not cookies)
- `/.well-known/*` — exempt from rate limiting

---

## Authorization Code Flow with PKCE

### Standard Flow (S256)

```
RP generates:
  code_verifier = random(43-128 chars)
  code_challenge = BASE64URL(SHA256(code_verifier))

1. GET /authorize?
     response_type=code
     &client_id=...
     &redirect_uri=...
     &scope=openid profile email
     &state=RANDOM
     &nonce=RANDOM                          ← echoed in id_token
     &code_challenge=BASE64URL(SHA256(v))
     &code_challenge_method=S256

2. User authenticates → 302 redirect_uri?code=CODE&state=STATE

3. POST /oauth/token
     grant_type=authorization_code
     &client_id=...
     &code=CODE
     &code_verifier=ORIGINAL_VERIFIER       ← server hashes and compares
     &redirect_uri=...                      ← must match step 1

4. Response: { access_token, id_token, token_type: "Bearer", expires_in }
```

### Plain PKCE

When `code_challenge_method` is omitted, Authorizer defaults to `plain` per RFC 7636 §4.2. In this case `code_verifier == code_challenge` (direct comparison, no hashing).

### Client Authentication

The token endpoint supports three authentication methods (advertised in discovery):

| Method | How |
|--------|-----|
| `client_secret_basic` | HTTP Basic: `Authorization: Basic base64(client_id:client_secret)` |
| `client_secret_post` | Form body: `client_id=...&client_secret=...` |
| `none` | Public client with PKCE (no secret required) |

---

## Nonce Handling

### Why Nonce Matters

The `nonce` parameter prevents token replay attacks. The RP generates a random nonce, sends it to `/authorize`, and expects it back as a claim in the `id_token`. If the nonce doesn't match, the RP rejects the token.

### Nonce in Code Flow (Backchannel)

**Nonce is required for backchannel (code) flow when using Enterprise IdP integrations.** While OIDC Core §3.1.2.1 makes nonce optional for the code flow, Auth0, Okta, and Keycloak all send and validate it.

Flow:
1. RP sends `nonce=ABC` to `/authorize`
2. Authorizer stores `nonce=ABC` with the authorization code
3. If user needs to log in, `nonce` is forwarded through the login UI via `authState`
4. After login, React app redirects back to `/authorize` with `nonce=ABC`
5. `/authorize` stores `nonce=ABC` in the code state data
6. RP exchanges code at `/oauth/token`
7. Token endpoint reads `nonce=ABC` from stored state → embeds it in `id_token`
8. RP validates `id_token.nonce == ABC`

### Nonce in Implicit Flow

For `response_type=id_token` or `response_type=id_token token`, nonce is **required** per OIDC Core §3.2.2.1. Authorizer enforces this and returns `invalid_request` if nonce is missing.

---

## Response Modes

| Mode | Description | Token Delivery |
|------|-------------|---------------|
| `query` | Redirect with params in query string | Only for `code` flow (no tokens in URLs) |
| `fragment` | Redirect with params in URL fragment | Default for implicit/hybrid flows |
| `form_post` | Auto-submitting HTML form POST | Used by Auth0 for Enterprise connections |
| `web_message` | `window.postMessage()` to RP | For SPA integrations |

### form_post CSP

When using `form_post`, Authorizer overrides the Content-Security-Policy header to allow the form to POST to the RP's redirect_uri:

```
form-action 'self' https://rp-domain.com https://rp-domain.com/callback/path
```

The CSP includes both the origin and the full path (without query params) for maximum browser compatibility.

---

## Prompt Parameter

### prompt=login

Forces re-authentication. Authorizer keeps the existing session for the immediate code generation (since the React SDK would auto-detect the session and redirect immediately), but the RP can enforce its own re-authentication logic.

### prompt=none

Checks for an existing session without showing the login UI. If no valid session exists, Authorizer returns `login_required` error to the RP's `redirect_uri` via the configured `response_mode`.

### prompt=consent / prompt=select_account

Accepted but not yet implemented — proceeds normally with a debug log.

---

## Session Management

### Token Endpoint Session Behavior

| Grant Type | Session Behavior |
|------------|-----------------|
| `authorization_code` | **Does NOT** create a new browser session or delete the existing one. The `/authorize` endpoint already created the session. Token endpoint is called server-to-server by the RP. |
| `refresh_token` | **Does** create a new browser session (session rollover). Old session is deleted for security. |

This distinction is critical: for `authorization_code` grant, the token endpoint caller is the RP's server (Auth0/Okta/Keycloak), not the user's browser. Setting cookies or deleting sessions here would affect the RP's HTTP client, not the user.

### auth_time Claim

When `max_age` is included in the authorization request, the `id_token` includes the `auth_time` claim (Unix timestamp of when the user authenticated). This is populated from the session's `IssuedAt` field.

---

## Troubleshooting

### "unexpected ID Token nonce claim value"

**Cause:** The nonce from the RP was lost during the login UI round-trip.
**Fix:** Ensure the `nonce` parameter is included in `authState` (fixed in this release).

### "code_verifier does not match code_challenge"

**Cause:** Either (a) PKCE parameters lost during login UI round-trip, or (b) `code_challenge_method` defaulting to wrong value.
**Fix:** `code_challenge` and `code_challenge_method` are now forwarded through `authState`. Default method is `plain` per RFC 7636 §4.2.

### "Sending form data violates Content Security Policy"

**Cause:** CSP `form-action` didn't include the RP's callback URL.
**Fix:** CSP now includes both the origin and full path of the redirect_uri.

### Session not found after OIDC login

**Cause:** `/oauth/token` was deleting the browser session during code exchange.
**Fix:** Token endpoint no longer deletes/replaces the browser session for `authorization_code` grant.

---

## RFC Compliance

| Standard | Status | Notes |
|----------|--------|-------|
| RFC 6749 (OAuth 2.0) | Compliant | Authorization code, refresh token, implicit grants |
| RFC 6750 (Bearer Token) | Compliant | Token type "Bearer", WWW-Authenticate on 401 |
| RFC 7636 (PKCE) | Compliant | S256 and plain methods, default plain per §4.2 |
| RFC 7009 (Token Revocation) | Compliant | Always returns 200, constant-time comparison |
| RFC 7662 (Token Introspection) | Compliant | active/inactive, validates iss/aud/exp |
| OIDC Core 1.0 | Compliant | id_token, nonce, auth_time, at_hash, c_hash |
| OIDC Discovery 1.0 | Compliant | All required + recommended fields |
| OIDC RP-Initiated Logout | Compliant | post_logout_redirect_uri validation |

### Error Response Format

All error responses use the RFC 6749 §5.2 format:
```json
{
  "error": "invalid_request",
  "error_description": "Human-readable description"
}
```

Standard error codes used: `invalid_request`, `invalid_client`, `invalid_grant`, `unauthorized_client`, `unsupported_grant_type`, `unsupported_response_type`, `login_required`.

### Authorization Code TTL

Authorization codes expire after **10 minutes** (RFC 6749 §4.1.2 recommendation). This is enforced in:
- **Redis**: Native TTL (`stateTTL = 10 * time.Minute`)
- **In-memory**: Entry-level expiration on read
- **Database**: `CreatedAt`-based expiration on read (600 seconds)
