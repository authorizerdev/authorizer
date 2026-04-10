# OAuth 2.0 & OpenID Connect Endpoints Reference

Authorizer implements the following industry-standard OAuth 2.0 and OpenID Connect endpoints. This document describes each endpoint, its parameters, and the relevant RFCs/specs it complies with.

## Table of Contents

- [Discovery](#openid-connect-discovery)
- [Authorization](#authorization-endpoint)
- [Token](#token-endpoint)
- [UserInfo](#userinfo-endpoint)
- [Token Revocation](#token-revocation-endpoint)
- [JWKS](#json-web-key-set-endpoint)
- [Logout](#logout-endpoint)

---

## OpenID Connect Discovery

**Endpoint:** `GET /.well-known/openid-configuration`

**Spec:** [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)

Returns metadata about the Authorizer instance so clients can auto-configure themselves.

### Response Fields

| Field | Description |
|-------|-------------|
| `issuer` | Base URL of the Authorizer instance |
| `authorization_endpoint` | URL for `/authorize` |
| `token_endpoint` | URL for `/oauth/token` |
| `userinfo_endpoint` | URL for `/userinfo` |
| `jwks_uri` | URL for `/.well-known/jwks.json` |
| `revocation_endpoint` | URL for `/oauth/revoke` |
| `end_session_endpoint` | URL for `/logout` |
| `response_types_supported` | `["code", "token", "id_token"]` |
| `grant_types_supported` | `["authorization_code", "refresh_token", "implicit"]` |
| `scopes_supported` | `["openid", "email", "profile", "offline_access"]` |
| `code_challenge_methods_supported` | `["S256"]` |
| `token_endpoint_auth_methods_supported` | `["client_secret_basic", "client_secret_post"]` |

> **Phase 1 conformance note:** `grant_types_supported` now includes `implicit` to honestly reflect that `/authorize` accepts `response_type=token` and `response_type=id_token`. The previously advertised `registration_endpoint` field has been removed because it pointed to the signup UI, not an RFC 7591 dynamic client registration endpoint; it will return when RFC 7591 is implemented.

### Usage

```bash
curl https://your-authorizer.example/.well-known/openid-configuration
```

Most OIDC client libraries will automatically fetch this to discover all other endpoints.

---

## Authorization Endpoint

**Endpoint:** `GET /authorize`

**Specs:** [RFC 6749 (OAuth 2.0)](https://www.rfc-editor.org/rfc/rfc6749) | [RFC 7636 (PKCE)](https://www.rfc-editor.org/rfc/rfc7636) | [OIDC Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)

Initiates the OAuth 2.0 authorization flow. Supports Authorization Code (with PKCE), Implicit Token, and Implicit ID Token flows.

### Request Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | Yes | Your application's client ID |
| `response_type` | Yes | `code`, `token`, or `id_token` |
| `state` | Yes | Anti-CSRF token (opaque string) |
| `redirect_uri` | No | Where to redirect after auth (defaults to `/app`) |
| `scope` | No | Space-separated scopes (default: `openid profile email`) |
| `response_mode` | No | `query`, `fragment`, `form_post`, or `web_message` |
| `code_challenge` | Recommended | PKCE challenge. Required for public clients; confidential clients may use `client_secret` instead |
| `code_challenge_method` | No | `S256` (default) or `plain` per RFC 7636 |
| `nonce` | Recommended | Binds ID token to session; REQUIRED for implicit flows per OIDC |
| `screen_hint` | No | Set to `signup` to show the signup page |

### Authorization Code Flow (Recommended)

```
GET /authorize?
  client_id=YOUR_CLIENT_ID
  &response_type=code
  &state=RANDOM_STATE
  &code_challenge=BASE64URL_SHA256_OF_VERIFIER
  &code_challenge_method=S256
  &redirect_uri=https://yourapp.com/callback
  &scope=openid profile email
```

**Success response:** Redirects to `redirect_uri?code=AUTH_CODE&state=RANDOM_STATE`

The `code` is single-use and short-lived per RFC 6749 Section 4.1.2.

### Implicit Flow

```
GET /authorize?
  client_id=YOUR_CLIENT_ID
  &response_type=token
  &state=RANDOM_STATE
  &nonce=RANDOM_NONCE
  &redirect_uri=https://yourapp.com/callback
```

**Success response:** Redirects to `redirect_uri#access_token=...&id_token=...&token_type=Bearer&state=...`

> **Phase 1 conformance note:** ID tokens issued from any flow now compute `at_hash` correctly as `base64url(sha256(access_token)[:16])` per OIDC Core §3.2.2.10, and echo the request's `nonce` (OIDC Core §2) when one was supplied. Previously the implicit/token branch set `at_hash` to the nonce value.

---

## Token Endpoint

**Endpoint:** `POST /oauth/token`

**Specs:** [RFC 6749 Section 3.2](https://www.rfc-editor.org/rfc/rfc6749#section-3.2) | [RFC 7636 Section 4.6](https://www.rfc-editor.org/rfc/rfc7636#section-4.6)

Exchanges an authorization code or refresh token for access/ID tokens.

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

### Authorization Code Grant

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `authorization_code` |
| `code` | Yes | The authorization code from `/authorize` |
| `code_verifier` | Yes* | The PKCE code verifier (43-128 chars) |
| `client_id` | Yes | Your application's client ID |
| `client_secret` | Yes* | Required if `code_verifier` is not provided |

*Either `code_verifier` or `client_secret` is required.

**Client authentication** can also be sent via HTTP Basic Auth (`Authorization: Basic base64(client_id:client_secret)`).

```bash
curl -X POST https://your-authorizer.example/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTH_CODE" \
  -d "code_verifier=YOUR_CODE_VERIFIER" \
  -d "client_id=YOUR_CLIENT_ID"
```

### Refresh Token Grant

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `refresh_token` |
| `refresh_token` | Yes | A valid refresh token |
| `client_id` | Yes | Your application's client ID |

```bash
curl -X POST https://your-authorizer.example/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=YOUR_REFRESH_TOKEN" \
  -d "client_id=YOUR_CLIENT_ID"
```

### Success Response (RFC 6749 Section 5.1)

```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "id_token": "eyJhbG...",
  "expires_in": 1800,
  "scope": "openid profile email",
  "refresh_token": "eyJhbG..."
}
```

### Error Response (RFC 6749 Section 5.2)

```json
{
  "error": "invalid_grant",
  "error_description": "The authorization code is invalid or has expired"
}
```

Standard error codes: `invalid_request`, `invalid_client`, `invalid_grant`, `unsupported_grant_type`, `invalid_scope`.

---

## UserInfo Endpoint

**Endpoint:** `GET /userinfo`

**Specs:** [OIDC Core Section 5.3](https://openid.net/specs/openid-connect-core-1_0.html#UserInfo) | [OIDC Core Section 5.4 (Requesting Claims using Scope Values)](https://openid.net/specs/openid-connect-core-1_0.html#ScopeClaims) | [RFC 6750 (Bearer Token)](https://www.rfc-editor.org/rfc/rfc6750)

Returns claims about the authenticated end-user, filtered by the scopes encoded in the access token.

```bash
curl -H "Authorization: Bearer ACCESS_TOKEN" \
  https://your-authorizer.example/userinfo
```

### Scope → claim mapping

Per OIDC Core §5.4, the response always includes `sub` plus only the claims permitted by the standard scope groups present on the access token. Clients must request the scopes they actually consume.

| Scope     | Claims returned in addition to `sub`                                                                                          |
|-----------|-------------------------------------------------------------------------------------------------------------------------------|
| `profile` | `name`, `family_name`, `given_name`, `middle_name`, `nickname`, `preferred_username`, `profile`, `picture`, `website`, `gender`, `birthdate`, `zoneinfo`, `locale`, `updated_at` |
| `email`   | `email`, `email_verified`                                                                                                     |
| `phone`   | `phone_number`, `phone_number_verified`                                                                                       |
| `address` | `address`                                                                                                                     |

Claim keys belonging to a granted scope group are always present in the response. If the underlying user has no value for a specific claim, the key is emitted with JSON `null` — explicitly permitted by OIDC Core §5.3.2 — so callers can rely on a stable response schema.

### Example responses

Requesting `openid email`:

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "email_verified": true
}
```

Requesting `openid profile email`:

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "email_verified": true,
  "given_name": "Jane",
  "family_name": "Doe",
  "nickname": null,
  "preferred_username": "user@example.com",
  "picture": "https://example.com/photo.jpg",
  "name": null,
  "middle_name": null,
  "profile": null,
  "website": null,
  "gender": null,
  "birthdate": null,
  "zoneinfo": null,
  "locale": null,
  "updated_at": 1712486400
}
```

Requesting only `openid`:

```json
{
  "sub": "user-uuid"
}
```

The `sub` claim is always returned per OIDC Core §5.3.2.

### Error Response (RFC 6750 Section 3)

When the token is missing or invalid, the response includes the `WWW-Authenticate` header:

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="authorizer", error="invalid_token", error_description="The access token is invalid or has expired"
```

---

## Token Revocation Endpoint

**Endpoint:** `POST /oauth/revoke`

**Spec:** [RFC 7009 (Token Revocation)](https://www.rfc-editor.org/rfc/rfc7009)

Revokes a refresh token. Per RFC 7009, this endpoint returns HTTP 200 even for invalid or already-revoked tokens (to prevent token scanning).

**Content-Type:** `application/x-www-form-urlencoded` (standard) or `application/json` (backward compatible)

| Parameter | Required | Description |
|-----------|----------|-------------|
| `token` | Yes | The refresh token to revoke |
| `client_id` | Yes | Your application's client ID |
| `token_type_hint` | No | `refresh_token` or `access_token` |

```bash
curl -X POST https://your-authorizer.example/oauth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=YOUR_REFRESH_TOKEN" \
  -d "client_id=YOUR_CLIENT_ID"
```

### Responses

- **200 OK** - Token was revoked (or was already invalid)
- **400 Bad Request** - Missing `client_id` or unsupported `token_type_hint`
- **401 Unauthorized** - Invalid `client_id`
- **503 Service Unavailable** - Server temporarily unable to process

---

## JSON Web Key Set Endpoint

**Endpoint:** `GET /.well-known/jwks.json`

**Spec:** [RFC 7517 (JWK)](https://www.rfc-editor.org/rfc/rfc7517)

Returns the public keys used to verify JWT signatures. Clients use this to validate access tokens and ID tokens.

```bash
curl https://your-authorizer.example/.well-known/jwks.json
```

### Response

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "your-client-id",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

Supports RSA (`RS256`, `RS384`, `RS512`), ECDSA (`ES256`, `ES384`, `ES512`), and HMAC (`HS256`, `HS384`, `HS512`) algorithms depending on configuration.

---

## Logout Endpoint

**Endpoint:** `GET /logout`

**Spec:** [OIDC RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html)

Ends the user's session and optionally redirects.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `redirect_uri` | No | URL to redirect to after logout |

```bash
# Redirect to logout, then back to your app
GET /logout?redirect_uri=https://yourapp.com
```

If no `redirect_uri` is provided, returns JSON: `{"message": "Logged out successfully"}`.

---

## PKCE (Proof Key for Code Exchange) Guide

PKCE (RFC 7636) is required for the authorization code flow. It prevents authorization code interception attacks.

### Step 1: Generate Code Verifier

A random string of 43-128 characters from `[A-Za-z0-9-._~]`:

```javascript
const codeVerifier = generateRandomString(43);
```

### Step 2: Create Code Challenge

```javascript
const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(codeVerifier));
const codeChallenge = btoa(String.fromCharCode(...new Uint8Array(hash)))
  .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
```

### Step 3: Start Authorization

```
GET /authorize?response_type=code&code_challenge=CODE_CHALLENGE&code_challenge_method=S256&...
```

### Step 4: Exchange Code

```
POST /oauth/token
grant_type=authorization_code&code=AUTH_CODE&code_verifier=CODE_VERIFIER&client_id=CLIENT_ID
```

---

## Standards Compliance Summary

| Standard | Status | Notes |
|----------|--------|-------|
| RFC 6749 (OAuth 2.0) | Implemented | Authorization Code + Refresh Token grants |
| RFC 7636 (PKCE) | Implemented | S256 (default) and plain methods; optional for confidential clients |
| RFC 7009 (Token Revocation) | Implemented | Returns 200 for invalid tokens |
| RFC 6750 (Bearer Token) | Implemented | WWW-Authenticate on 401 |
| OIDC Core 1.0 | Implemented | ID tokens, UserInfo, nonce |
| OIDC Discovery 1.0 | Implemented | All required + recommended fields |
| RFC 7517 (JWK) | Implemented | RSA, ECDSA, HMAC support |
