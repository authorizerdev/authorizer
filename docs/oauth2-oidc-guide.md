# OAuth 2.0 & OpenID Connect Guide for Authorizer

## What is this about? (Plain English)

Imagine you're building a website and you want users to log in. You have two choices:

1. **Build your own login system** — store passwords, handle "forgot password", deal with security. A lot of work.
2. **Use someone else's login system** — redirect users to a trusted service (like Google, Auth0, or your own Authorizer instance), let them log in there, and get back a "proof" that the user is who they say they are.

OAuth 2.0 and OpenID Connect (OIDC) are the standards that make option 2 work. They define a common language so any app can talk to any login provider.

**Authorizer can play two roles:**

- **Identity Provider (IdP)** — Authorizer IS the login system. Your apps redirect users to Authorizer to log in.
- **Relying Party (RP)** — Authorizer USES another login system (like Google, Auth0, GitHub) to authenticate users, then issues its own tokens.

### Real-world analogy

Think of it like a passport system:

- **IdP (Identity Provider)** = The government that issues passports. It verifies who you are and gives you a document that proves it.
- **RP (Relying Party)** = The hotel front desk. It doesn't verify your identity directly — it trusts your passport (issued by the government) and lets you check in.
- **OAuth 2.0** = The rules for how the hotel can verify your passport without calling the government every time.
- **OIDC** = An upgrade to OAuth 2.0 that adds a standardized ID card (the `id_token`) alongside the access pass.

---

## Table of Contents

- [Key Concepts](#key-concepts)
- [Authorizer as Identity Provider (IdP)](#authorizer-as-identity-provider-idp)
- [Authorizer as Relying Party (RP)](#authorizer-as-relying-party-rp)
- [Grant Types Explained](#grant-types-explained)
- [Response Types Explained](#response-types-explained)
- [PKCE Explained](#pkce-explained)
- [Tokens Explained](#tokens-explained)
- [Scopes and Claims](#scopes-and-claims)
- [Endpoints Reference](#endpoints-reference)
- [Integration Examples](#integration-examples)
- [Security Considerations](#security-considerations)
- [FAQ](#faq)

---

## Key Concepts

### The Players

| Term | Who/What | Example |
|------|----------|---------|
| **Resource Owner** | The human user | Jane who wants to log in |
| **Client** | The application requesting access | Your React app, mobile app, or backend |
| **Authorization Server (AS)** | The server that authenticates users and issues tokens | Authorizer |
| **Resource Server (RS)** | The API that holds protected data | Your backend API |
| **Identity Provider (IdP)** | A server trusted to verify identity and issue ID tokens | Authorizer, Google, Auth0 |
| **Relying Party (RP)** | An application that trusts an IdP to handle authentication | Your app trusting Authorizer, or Authorizer trusting Google |

### The Tokens

| Token | What it is | Analogy |
|-------|-----------|---------|
| **Authorization Code** | A short-lived, one-time ticket exchanged for real tokens | A baggage claim ticket — useless on its own, exchanged for something valuable |
| **Access Token** | Proves you're allowed to access a resource | A hotel key card — lets you open doors |
| **ID Token** | Proves who you are (a JWT with user info) | A passport — contains your name, photo, and identity |
| **Refresh Token** | Used to get new access tokens without re-logging in | A season pass — lets you get new day passes without buying again |

### OAuth 2.0 vs OpenID Connect

| | OAuth 2.0 | OpenID Connect (OIDC) |
|---|---|---|
| **Purpose** | Authorization — "what can you do?" | Authentication — "who are you?" |
| **Token** | Access token | ID token (+ access token) |
| **User info** | No standard way | Standard `/userinfo` endpoint and `id_token` claims |
| **Discovery** | No standard | `/.well-known/openid-configuration` auto-discovery |

OIDC is built on top of OAuth 2.0. Every OIDC flow is also an OAuth 2.0 flow, but with identity information added.

---

## Authorizer as Identity Provider (IdP)

**When to use:** You want Authorizer to BE the login system for your applications.

In this mode, your application redirects users to Authorizer's `/authorize` endpoint. Users log in (email/password, social login, MFA), and Authorizer redirects them back to your app with tokens.

### Architecture

```
┌──────────────┐     1. Redirect to /authorize      ┌──────────────────┐
│              │ ──────────────────────────────────►  │                  │
│  Your App    │                                      │   Authorizer     │
│  (Client)    │  4. Redirect back with code/tokens   │   (IdP)          │
│              │ ◄──────────────────────────────────  │                  │
└──────┬───────┘                                      └────────┬─────────┘
       │                                                       │
       │  5. Call API with access_token                        │ 2. Show login UI
       │                                                       │ 3. User authenticates
       ▼                                                       ▼
┌──────────────┐                                      ┌──────────────────┐
│  Your API    │                                      │   User's Browser │
│  (Resource   │                                      │                  │
│   Server)    │                                      └──────────────────┘
└──────────────┘
```

### When to use Authorizer as IdP

- You want **full control** over user data (self-hosted)
- You need a **central login system** for multiple apps
- You're building an **enterprise SSO** solution
- You want to offer Authorizer as an OpenID Connect provider to **third-party applications**
- You want to connect Authorizer to **Auth0, Okta, or another service** as an enterprise connection

### Setup Checklist (IdP mode)

1. Deploy Authorizer and note your instance URL (e.g., `https://auth.yourcompany.com`)
2. Note your `client_id` and `client_secret` from Authorizer config
3. Set `--allowed-origins` to your app's domain(s)
4. In your app, redirect users to `https://auth.yourcompany.com/authorize?...`
5. Handle the callback at your `redirect_uri`
6. Exchange the code for tokens at `/oauth/token`

### Connecting Authorizer as IdP to Auth0

When using Authorizer as an enterprise OpenID Connect connection in Auth0:

1. In Auth0 Dashboard, go to **Authentication > Enterprise > OpenID Connect**
2. Set:
   - **Issuer URL**: `https://auth.yourcompany.com`  
   - **Client ID**: Your Authorizer `client_id`
   - **Client Secret**: Your Authorizer `client_secret`
3. Auth0 will auto-discover endpoints via `/.well-known/openid-configuration`
4. Choose **Front Channel** (implicit) or **Back Channel** (authorization code):

| Channel | How it works | When to use |
|---------|-------------|-------------|
| **Front Channel** | Auth0 gets tokens directly via browser redirect (`response_type=id_token`) | Simple setup, no server-to-server connectivity needed |
| **Back Channel** | Auth0's server exchanges the code at Authorizer's `/oauth/token` | More secure, requires Authorizer to be reachable from Auth0's servers |

**Important for Back Channel:** Authorizer must be publicly accessible from the internet (Auth0's servers need to reach `/oauth/token` and `/.well-known/jwks.json`).

---

## Authorizer as Relying Party (RP)

**When to use:** You want Authorizer to delegate login to external providers (Google, GitHub, Facebook, etc.) while still managing users and sessions itself.

In this mode, Authorizer redirects users to the external provider, receives their identity, creates/links a local user, and issues its own tokens.

### Architecture

```
┌──────────────┐    1. Click "Login with Google"     ┌──────────────────┐
│              │ ──────────────────────────────────►  │                  │
│  Your App    │                                      │   Authorizer     │
│  (Client)    │  5. Redirect back with tokens        │   (RP to Google) │
│              │ ◄──────────────────────────────────  │                  │
└──────────────┘                                      └────────┬─────────┘
                                                               │
                                            2. Redirect to     │  4. Exchange code,
                                               Google          │     get user info,
                                                               │     create/link user
                                                               ▼
                                                      ┌──────────────────┐
                                                      │   Google (IdP)   │
                                                      │   GitHub, etc.   │
                                                      └──────────────────┘
                                                               ▲
                                                               │ 3. User logs in
                                                               │    at Google
                                                      ┌──────────────────┐
                                                      │   User's Browser │
                                                      └──────────────────┘
```

### When to use Authorizer as RP

- You want **social logins** (Google, GitHub, Facebook, Apple, etc.)
- You want to **federate identity** from external providers
- You want users to have **one account** regardless of how they sign in

### Supported OAuth Providers (RP mode)

| Provider | Config flags |
|----------|-------------|
| Google | `--google-client-id`, `--google-client-secret` |
| GitHub | `--github-client-id`, `--github-client-secret` |
| Facebook | `--facebook-client-id`, `--facebook-client-secret` |
| LinkedIn | `--linkedin-client-id`, `--linkedin-client-secret` |
| Apple | `--apple-client-id`, `--apple-client-secret` |
| Discord | `--discord-client-id`, `--discord-client-secret` |
| Twitter/X | `--twitter-client-id`, `--twitter-client-secret` |
| Microsoft | `--microsoft-client-id`, `--microsoft-client-secret`, `--microsoft-tenant-id` |
| Twitch | `--twitch-client-id`, `--twitch-client-secret` |
| Roblox | `--roblox-client-id`, `--roblox-client-secret` |

### Social Login Flow

```
GET /oauth_login/google?redirect_uri=https://yourapp.com/callback&scope=openid profile email
```

This redirects the user to Google. After login, Google redirects back to Authorizer's `/oauth_callback/google`, which processes the user and redirects to your `redirect_uri` with tokens.

---

## Grant Types Explained

A "grant type" is the method by which your application obtains tokens. Think of it as different ways to prove you deserve access.

### Authorization Code Grant (`authorization_code`)

**The recommended flow for almost everything.**

```
Your App                    Authorizer                    User
   │                            │                          │
   │  1. Redirect to /authorize │                          │
   │ ──────────────────────────►│                          │
   │                            │  2. Show login page      │
   │                            │ ────────────────────────►│
   │                            │                          │
   │                            │  3. User logs in         │
   │                            │ ◄────────────────────────│
   │                            │                          │
   │  4. Redirect with code     │                          │
   │ ◄──────────────────────────│                          │
   │                            │                          │
   │  5. POST /oauth/token      │                          │
   │     (code + verifier)      │                          │
   │ ──────────────────────────►│                          │
   │                            │                          │
   │  6. Return tokens          │                          │
   │ ◄──────────────────────────│                          │
```

**When to use:**
- Web apps with a backend (server-side rendering)
- Single Page Apps (SPAs) — with PKCE
- Mobile apps — with PKCE
- Any new application

**Security:** The authorization code is exchanged server-to-server (or with PKCE), so tokens never appear in the browser URL bar.

**Example:**

```bash
# Step 1: Redirect user to authorize
GET /authorize?
  client_id=YOUR_CLIENT_ID
  &response_type=code
  &state=random_csrf_token
  &redirect_uri=https://yourapp.com/callback
  &scope=openid profile email
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256

# Step 2: User logs in, gets redirected to:
# https://yourapp.com/callback?code=AUTH_CODE&state=random_csrf_token

# Step 3: Exchange code for tokens (server-side)
curl -X POST https://auth.example.com/oauth/token \
  -d "grant_type=authorization_code" \
  -d "code=AUTH_CODE" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk" \
  -d "redirect_uri=https://yourapp.com/callback"
```

### Implicit Grant (`implicit`)

**Tokens returned directly in the URL. Simpler but less secure.**

```
Your App                    Authorizer                    User
   │                            │                          │
   │  1. Redirect to /authorize │                          │
   │ ──────────────────────────►│                          │
   │                            │  2. User logs in         │
   │                            │ ◄───────────────────────►│
   │                            │                          │
   │  3. Redirect with tokens   │                          │
   │     (in URL fragment #)    │                          │
   │ ◄──────────────────────────│                          │
```

**When to use:**
- Legacy SPAs that can't use PKCE
- Simple integrations where security requirements are lower
- Front-channel connections (e.g., Auth0 front channel)

**Not recommended for new apps** — use Authorization Code + PKCE instead.

**Example:**

```bash
# response_type=token → returns access_token
GET /authorize?
  client_id=YOUR_CLIENT_ID
  &response_type=token
  &state=random_csrf_token
  &nonce=random_nonce
  &redirect_uri=https://yourapp.com/callback

# User logs in, gets redirected to:
# https://yourapp.com/callback#access_token=...&token_type=Bearer&state=...

# response_type=id_token → returns only id_token
GET /authorize?
  client_id=YOUR_CLIENT_ID
  &response_type=id_token
  &state=random_csrf_token
  &nonce=random_nonce
  &redirect_uri=https://yourapp.com/callback
```

### Refresh Token Grant (`refresh_token`)

**Get new tokens without making the user log in again.**

```bash
curl -X POST https://auth.example.com/oauth/token \
  -d "grant_type=refresh_token" \
  -d "refresh_token=YOUR_REFRESH_TOKEN" \
  -d "client_id=YOUR_CLIENT_ID"
```

**When to use:**
- Your access token expired but you don't want to redirect the user
- Background API calls that need fresh tokens
- Mobile apps maintaining long sessions

**Requirements:** The original authorization must include `offline_access` scope to receive a refresh token.

### Hybrid Flow (OIDC)

**Combines authorization code and implicit — returns some tokens immediately, others via code exchange.**

| `response_type` | Returns at redirect | Returns at `/oauth/token` |
|---|---|---|
| `code id_token` | code + id_token | access_token + refresh_token |
| `code token` | code + access_token | id_token + refresh_token |
| `code id_token token` | code + id_token + access_token | refresh_token |
| `id_token token` | id_token + access_token | (no code exchange) |

**When to use:**
- You need the ID token immediately (for client-side validation) but want to exchange the code securely for the access token
- Advanced OIDC integrations

---

## Response Types Explained

The `response_type` parameter tells Authorizer what to return after the user logs in.

| `response_type` | Flow | Returns | Token delivery |
|---|---|---|---|
| `code` | Authorization Code | Authorization code | Via redirect (query or fragment) |
| `token` | Implicit | Access token + ID token | Via redirect (fragment only) |
| `id_token` | Implicit | ID token only | Via redirect (fragment only) |
| `code id_token` | Hybrid | Code + ID token | Via redirect (fragment only) |
| `code token` | Hybrid | Code + Access token | Via redirect (fragment only) |
| `code id_token token` | Hybrid | Code + ID token + Access token | Via redirect (fragment only) |
| `id_token token` | Hybrid | ID token + Access token | Via redirect (fragment only) |

### Response Modes

The `response_mode` parameter controls HOW the response is delivered:

| `response_mode` | Description | Use case |
|---|---|---|
| `query` | Parameters in URL query string (`?code=...`) | Authorization code flow (default) |
| `fragment` | Parameters in URL fragment (`#token=...`) | Implicit/hybrid flows (default) |
| `form_post` | Parameters POSTed as a form to redirect_uri | Server-side apps, avoids URL length limits |
| `web_message` | HTML5 `postMessage` to parent window | Silent authentication in iframes |

**Security rule:** `response_mode=query` is only allowed with `response_type=code`. Tokens in query strings get logged in server access logs and browser history — a credential leak.

---

## PKCE Explained

**PKCE (Proof Key for Code Exchange)** prevents attackers from intercepting the authorization code and exchanging it for tokens.

### The Problem Without PKCE

```
Your App ──► /authorize ──► User logs in ──► Redirect with code
                                                    │
                                              Attacker intercepts
                                              the code and exchanges
                                              it for tokens!
```

### How PKCE Solves It

```
1. Your app generates a random secret: code_verifier
2. Your app computes: code_challenge = BASE64URL(SHA256(code_verifier))
3. Your app sends code_challenge to /authorize (public)
4. User logs in, code is returned
5. Even if attacker gets the code, they don't have the code_verifier
6. Your app sends code_verifier to /oauth/token
7. Server verifies: SHA256(code_verifier) == stored code_challenge
```

### Methods

| Method | How it works | Security |
|---|---|---|
| `S256` | `code_challenge = BASE64URL(SHA256(code_verifier))` | Recommended. Even if challenge is intercepted, verifier cannot be derived. |
| `plain` | `code_challenge = code_verifier` | Only use when S256 is impossible. Offers no protection if challenge is intercepted. |

### Code Examples

**JavaScript (Browser/Node.js):**

```javascript
// Generate code_verifier (43-128 characters)
function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Compute code_challenge (S256)
async function generateCodeChallenge(verifier) {
  const hash = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(verifier)
  );
  return btoa(String.fromCharCode(...new Uint8Array(hash)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Usage
const codeVerifier = generateCodeVerifier();
const codeChallenge = await generateCodeChallenge(codeVerifier);

// Send code_challenge to /authorize
// Store code_verifier in sessionStorage
// Send code_verifier to /oauth/token when exchanging the code
```

**Go:**

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
)

// Generate code_verifier
verifierBytes := make([]byte, 32)
rand.Read(verifierBytes)
codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

// Compute code_challenge (S256)
hash := sha256.Sum256([]byte(codeVerifier))
codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
```

**Python:**

```python
import secrets, hashlib, base64

# Generate code_verifier
code_verifier = secrets.token_urlsafe(32)

# Compute code_challenge (S256)
digest = hashlib.sha256(code_verifier.encode()).digest()
code_challenge = base64.urlsafe_b64encode(digest).rstrip(b'=').decode()
```

### When is PKCE required?

| Client type | PKCE | client_secret |
|---|---|---|
| **Public client** (SPA, mobile) | Required | Not available |
| **Confidential client** (server-side) | Optional (recommended) | Required if no PKCE |
| **Confidential client with PKCE** | Verified | Also verified |

---

## Tokens Explained

### Access Token

A JWT that grants access to protected resources. Short-lived (default: 30 minutes).

**Claims:**
```json
{
  "sub": "user-uuid",
  "iss": "https://auth.example.com",
  "aud": "client-id",
  "exp": 1712488200,
  "iat": 1712486400,
  "scope": ["openid", "profile", "email"],
  "roles": ["user"],
  "token_type": "access_token"
}
```

**Usage:** Send in the `Authorization` header:
```
Authorization: Bearer eyJhbG...
```

### ID Token

A JWT that contains identity information about the user. This is the OIDC layer on top of OAuth 2.0.

**Claims:**
```json
{
  "sub": "user-uuid",
  "iss": "https://auth.example.com",
  "aud": "client-id",
  "exp": 1712488200,
  "iat": 1712486400,
  "nonce": "random-nonce-from-request",
  "at_hash": "half-hash-of-access-token",
  "email": "user@example.com",
  "email_verified": true,
  "given_name": "Jane",
  "family_name": "Doe",
  "token_type": "id_token"
}
```

**Validation:** Clients MUST verify:
1. `iss` matches the expected issuer
2. `aud` contains their `client_id`
3. `exp` is in the future
4. `nonce` matches what they sent (if they sent one)
5. Signature verifies against the JWKS keys

### Refresh Token

An opaque token used to get new access/ID tokens. Long-lived (default: configurable). Rotated on each use for security.

---

## Scopes and Claims

Scopes determine what information is included in tokens and the `/userinfo` response.

| Scope | What it grants |
|---|---|
| `openid` | Required for OIDC. Returns `sub` claim. |
| `profile` | Name, nickname, picture, gender, birthdate, etc. |
| `email` | Email address and verification status |
| `phone` | Phone number and verification status |
| `offline_access` | Issues a refresh token |

### Example

```
scope=openid profile email
```

This returns the user's identity (`sub`), profile information (name, picture), and email in both the ID token and `/userinfo` response.

---

## Endpoints Reference

| Endpoint | Method | Purpose | Spec |
|---|---|---|---|
| `/.well-known/openid-configuration` | GET | Auto-discovery of all endpoints | OIDC Discovery 1.0 |
| `/.well-known/jwks.json` | GET | Public keys for token verification | RFC 7517 |
| `/authorize` | GET | Start OAuth/OIDC flow | RFC 6749, OIDC Core |
| `/oauth/token` | POST | Exchange code/refresh token for tokens | RFC 6749 |
| `/userinfo` | GET | Get user claims (with access token) | OIDC Core §5.3 |
| `/oauth/revoke` | POST | Revoke a refresh token | RFC 7009 |
| `/oauth/introspect` | POST | Check if a token is active | RFC 7662 |
| `/logout` | GET | End session, optional redirect | OIDC RP-Initiated Logout |
| `/oauth_login/:provider` | GET | Start social login | Authorizer-specific |
| `/oauth_callback/:provider` | GET/POST | Social login callback | Authorizer-specific |

See [oauth2-oidc-endpoints.md](oauth2-oidc-endpoints.md) for detailed parameter reference.

---

## Integration Examples

### Example 1: React SPA with Authorizer as IdP

```javascript
// 1. Redirect to login
const codeVerifier = generateCodeVerifier();
sessionStorage.setItem('code_verifier', codeVerifier);
const codeChallenge = await generateCodeChallenge(codeVerifier);

window.location.href = `https://auth.example.com/authorize?` +
  `client_id=YOUR_CLIENT_ID` +
  `&response_type=code` +
  `&redirect_uri=${encodeURIComponent('https://yourapp.com/callback')}` +
  `&scope=openid profile email` +
  `&state=${generateRandomState()}` +
  `&code_challenge=${codeChallenge}` +
  `&code_challenge_method=S256`;

// 2. Handle callback (at /callback route)
const params = new URLSearchParams(window.location.search);
const code = params.get('code');
const codeVerifier = sessionStorage.getItem('code_verifier');

const response = await fetch('https://auth.example.com/oauth/token', {
  method: 'POST',
  headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({
    grant_type: 'authorization_code',
    code: code,
    client_id: 'YOUR_CLIENT_ID',
    code_verifier: codeVerifier,
    redirect_uri: 'https://yourapp.com/callback',
  }),
});

const tokens = await response.json();
// tokens.access_token, tokens.id_token, tokens.refresh_token
```

### Example 2: Server-side app (Node.js/Express)

```javascript
// 1. Login route — redirect to Authorizer
app.get('/login', (req, res) => {
  const state = crypto.randomBytes(16).toString('hex');
  req.session.oauthState = state;

  res.redirect(`https://auth.example.com/authorize?` +
    `client_id=YOUR_CLIENT_ID` +
    `&response_type=code` +
    `&redirect_uri=${encodeURIComponent('https://yourapp.com/callback')}` +
    `&scope=openid profile email` +
    `&state=${state}`);
});

// 2. Callback route — exchange code
app.get('/callback', async (req, res) => {
  // Verify state
  if (req.query.state !== req.session.oauthState) {
    return res.status(400).send('Invalid state');
  }

  // Exchange code for tokens (server-to-server, uses client_secret)
  const response = await fetch('https://auth.example.com/oauth/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      code: req.query.code,
      client_id: 'YOUR_CLIENT_ID',
      client_secret: 'YOUR_CLIENT_SECRET',
      redirect_uri: 'https://yourapp.com/callback',
    }),
  });

  const tokens = await response.json();
  req.session.accessToken = tokens.access_token;
  res.redirect('/dashboard');
});
```

### Example 3: Authorizer as Auth0 Enterprise Connection

```
Auth0 Dashboard:
  Authentication > Enterprise > OpenID Connect > Create Connection

  Issuer URL:     https://auth.yourcompany.com
  Client ID:      kbyuFDidLLm280LIwVFiazOqjO3ty8KH
  Client Secret:  60Op4HFM0I8ajz0WdiStAbziZ-VFQttXuxixHHs2R7r7-CW8GR79l-mmLqMhc-Sa

  Type: Back Channel (recommended) or Front Channel

  Auth0 auto-discovers:
    /.well-known/openid-configuration → all endpoints
    /.well-known/jwks.json → public keys for token verification
```

---

## Security Considerations

### Always Use PKCE for Public Clients

SPAs and mobile apps cannot securely store a `client_secret`. PKCE replaces it.

### Validate the `state` Parameter

Always verify the `state` you receive matches the one you sent. This prevents CSRF attacks.

### Validate `nonce` in ID Tokens

When using implicit or hybrid flows, always verify the `nonce` in the ID token matches what you sent to prevent token replay.

### Use HTTPS

All OAuth 2.0/OIDC flows MUST use HTTPS in production. Tokens in HTTP are trivially interceptable.

### Token Storage

| Token | Where to store | Why |
|---|---|---|
| Access token | Memory (JS variable) | Short-lived, no persistence needed |
| Refresh token | Secure HttpOnly cookie | Prevents XSS access |
| ID token | Memory or sessionStorage | Only needed for identity verification |

Never store tokens in `localStorage` — it's accessible to any JavaScript on the page (XSS vulnerability).

### Redirect URI Validation

Authorizer validates redirect URIs against an exact-match allowlist configured via `--allowed-origins`. No wildcards, no prefix matching.

---

## FAQ

### When should I use Authorization Code vs Implicit?

**Always use Authorization Code + PKCE for new apps.** Implicit is supported for backward compatibility and legacy integrations (like Auth0 front channel), but it's less secure because tokens are exposed in the URL.

### What's the difference between IdP and RP?

- **IdP**: Authorizer manages users and issues tokens. Your apps trust Authorizer.
- **RP**: Authorizer trusts an external provider (Google, GitHub) to verify identity, then issues its own tokens.

Authorizer can be both simultaneously — it acts as an RP to Google (for social login) and as an IdP to your apps (issuing tokens they trust).

### Do I need both `client_id` and `client_secret`?

- **Public clients** (SPAs, mobile): Only `client_id` + PKCE
- **Confidential clients** (server-side): `client_id` + `client_secret` (PKCE optional but recommended)
- **With PKCE**: Both `code_verifier` AND `client_secret` are validated if `client_secret` is provided

### What's `offline_access` scope?

It tells Authorizer to issue a refresh token. Without it, you only get access and ID tokens, and users must log in again when the access token expires.

### How do I refresh tokens?

```bash
POST /oauth/token
grant_type=refresh_token
&refresh_token=YOUR_REFRESH_TOKEN
&client_id=YOUR_CLIENT_ID
```

The old refresh token is invalidated and a new one is returned (rotation for security).

### What's the difference between Front Channel and Back Channel?

| | Front Channel | Back Channel |
|---|---|---|
| **How** | Tokens flow through the browser (redirects) | Code flows through browser, tokens exchanged server-to-server |
| **Security** | Tokens visible in URL | Tokens never in URL |
| **Network** | Only needs browser connectivity | Server must be reachable from the RP's backend |
| **Example** | `response_type=id_token` | `response_type=code` + `/oauth/token` exchange |
