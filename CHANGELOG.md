# Changelog

All notable changes to Authorizer will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Targets the 2.4.0 release. Significant additions include enterprise SSO (SAML IdP + verified domains + home realm discovery), WebAuthn/passkey login, SCIM 2.0 groups, redesigned MFA behavior, and OAuth 2.1/MCP hardening.

### Added

- **Unified OAuth Client registry (machine & agent identity foundation)**: All clients (human, machine, agent) are registered in a single `authorizer_clients` table with a `kind` discriminator (`interactive` | `service_account`). Service accounts can use the `client_credentials` grant for machine-to-machine authentication, while agents can participate in delegation chains. Admin GraphQL/gRPC operations manage clients with secret generation (32-byte crypto/rand, bcrypt-12 at rest), scope-subset enforcement, and one-time secret reveal ([#648](https://github.com/authorizerdev/authorizer/pull/648)).
- **Machine-to-machine (service-to-service) authentication**: Service account clients use the RFC 6749 §4.4 `client_credentials` grant at `/oauth/token` to mint access tokens for autonomous workloads. Tokens carry `login_method: service_account` and resolve to `service_account:<client_id>` FGA subjects instead of users. Scope-subset enforcement and timing-safe authentication prevent privilege escalation ([#641](https://github.com/authorizerdev/authorizer/pull/641), [#642](https://github.com/authorizerdev/authorizer/pull/642), [#644](https://github.com/authorizerdev/authorizer/pull/644), [#645](https://github.com/authorizerdev/authorizer/pull/645), [#647](https://github.com/authorizerdev/authorizer/pull/647)).
- **Secretless workload identity (RFC 7523 + SPIFFE JWT-SVID + Kubernetes TokenReview)**: Service accounts can authenticate via `client_assertion` (JWT-bearer) with `private_key_jwt` or `jwt-spiffe` assertion types. Trusted issuers validate assertion signatures, pin subject claims, and prevent replay via single-use `jti` in a bounded-TTL cache. When enabled, Kubernetes TokenReview API validates projected ServiceAccount tokens before issuance. All authentication paths share constant-time comparison and SSRF-hardened external fetch for JWKS ([#654](https://github.com/authorizerdev/authorizer/pull/654), [#659](https://github.com/authorizerdev/authorizer/pull/659)).
- **Registry-authoritative client authentication**: All client-auth sites (`/oauth/introspect`, `/oauth/revoke`, `/graphql` client-check middleware, OIDC discovery) now route through the unified client registry via a shared resolver instead of static config comparisons. Introspection no longer leaks cross-client information; revocation uses token-ownership guards; `/graphql` client-id validation is now abort-safe. Discovery endpoint (`/.well-known/oauth-authorization-server` alias for MCP compliance) now advertises `client_credentials` and `private_key_jwt` as supported grant/auth methods ([#655](https://github.com/authorizerdev/authorizer/pull/655)).
- **Interactive client registry columns & reserved-client seed**: Client registry schema extended with `interactive_kind` (for future SSO/delegation profiles), and a reserved boot-seeded client with immutable `ClientID` matching `--client-id` flag. Prevents accidental duplication and enables deterministic identity ([#652](https://github.com/authorizerdev/authorizer/pull/652)).
- **Shared client-auth resolver & grant-matrix hardening**: The token endpoint and OAuth client-check middleware now call a single `clientauth` resolver to authenticate via Basic auth, form body, or `client_assertion`. Grant-matrix validation ensures `client_credentials` cannot be issued to `interactive` clients and vice versa ([#651](https://github.com/authorizerdev/authorizer/pull/651)).
- **Agent-to-agent (A2A) delegation (RFC 8693 token-exchange)**: Authenticated service account agents can exchange a user's access token + their own actor token for a resource-bound, attenuated access token carrying a nested `act` actor chain. `sub` remains the user; `act` encodes "agent acting on behalf of user" (multi-hop via recursion). Attenuation is monotonic — effective scope is the intersection of subject token scope, agent's allowed scopes, and requested scope (deny-all if empty). Single resource per request, hard depth cap on nesting, and reserved claims prevent forging ([#658](https://github.com/authorizerdev/authorizer/pull/658)).
- **Organizations & user-org membership**: Foundational entities for multi-tenant isolation. Organizations are created/managed via admin API; users are members of orgs with audit trails. Org membership and admin roles gate access to org-scoped resources (OIDC/SAML/SCIM connections, group bindings) ([#653](https://github.com/authorizerdev/authorizer/pull/653)).
- **Per-organization OIDC SSO federation**: Organizations can configure upstream OIDC IdP connections (Okta, Entra, Google). When enabled, `/oauth/sso/:org_slug/login` initiates PKCE+nonce to the upstream IdP; `/callback` exchanges the code, validates the ID token, and JIT-provisions users. User resolution is federated-identity-namespaced `(org_id, issuer, subject)`, preventing account-takeover via email collision. Discovery of upstream `sso_oidc` rows rejects `client_assertion` auth via CR1 kind discriminator ([#657](https://github.com/authorizerdev/authorizer/pull/657)).
- **Per-organization SAML 2.0 SSO (Service Provider)**: Organizations can configure upstream SAML IdP connections. Authorizer acts as a SAML Service Provider, handling signed assertions with XML-DSIG validation, per-org audience/recipient/destination binding, NotBefore/NotOnOrAfter skew, and single-use AssertionID replay detection. JIT-provisioning namespaces users by `(org_id, IdP entity-id, NameID)`. Admin CRUD is super-admin gated ([#660](https://github.com/authorizerdev/authorizer/pull/660)).
- **Per-organization SCIM 2.0 user provisioning**: Organizations create scoped SCIM bearer tokens; IdP (Okta, Entra) auto-provisions users via `/scim/v2/Users` CRUD endpoints. `active:false` PATCH/DELETE deactivates users and revokes sessions/refresh tokens. Org isolation is enforced by bearer token (never URL/body). ExternalID dedup is org-scoped. Three pre-existing storage bugs fixed: Couchbase silent deprovision no-op, Redis session revocation glob mismatch, and Cassandra async-index race on membership inserts ([#656](https://github.com/authorizerdev/authorizer/pull/656)).
- **Fine-grained authorization (FGA) — embedded OpenFGA ReBAC engine**: Replaces the non-released bespoke FGA with an OpenFGA-backed relationship-based access control (ReBAC) engine. Admin GraphQL `_fga_model`, `_fga_tuples`, `_fga_write_tuples`, `_fga_read_tuples`, `_fga_delete_tuples` manage the authorization model and tuples. Public API: `CheckPermissions` and `ListPermissions` (both transport-agnostic, subject pinned to caller, fail-closed). Session/validate endpoints use `required_relations` (FGA) instead of `required_permissions`. `--authorization-engine=fga`, `--fga-mode` (embedded|external), `--fga-store` (memory|sqlite|postgres|mysql) configure deployment. SQLite driver unified on `modernc.org/sqlite` for both app and embedded OpenFGA datastore ([#625](https://github.com/authorizerdev/authorizer/pull/625)).
- **Multi-protocol public API surface (GraphQL + gRPC + REST + MCP)**: Public GraphQL operations are now available on all four transports. gRPC (`port 9091`, `authorizer.v1.AuthorizerService`, 20 RPCs); REST (`/v1/{method}`, gin router, same middleware); MCP (`authorizer mcp` CLI subcommand for `claude mcp add`). Single proto source of truth with buf STANDARD lint enforced. Single service layer (`RequestMetadata` / `ResponseSideEffects`) backing all transports. gRPC reflection enabled (flag-gated). REST uses `UseProtoNames=true` for snake_case parity with GraphQL. MCP schema derived from proto descriptors with cycle guard. Admin ops remain GraphQL-only ([#620](https://github.com/authorizerdev/authorizer/pull/620)).
- **Dashboard admin pages for client management and trusted issuers**: New `/identity/clients` page lists, creates, edits, and rotates service account clients; `client_secret` plaintext shown exactly once (create/rotate) in a copy dialog. `/identity/trusted-issuers` provides full CRUD of trusted issuers with field constraints per update request shape. Overview card relabeled **Default Client ID** with a link to the new Clients page ([#662](https://github.com/authorizerdev/authorizer/pull/662)).
- **Dashboard admin pages for organization management**: New `/identity/organizations` page lists, creates, edits, deletes organizations. Organization detail view includes Members (add/remove), per-org SSO connections (OIDC/SAML create/edit/delete), and SCIM endpoint management (create/delete/rotate token) ([#663](https://github.com/authorizerdev/authorizer/pull/663)).
- **gRPC transport (port 9091)**: all 20 public auth operations and 32 admin operations are now served over native gRPC alongside GraphQL and REST. The listener binds to `--grpc-port` (default `9091`), separate from the HTTP port. All three transports share the same service layer and return identical flat response shapes ([#634](https://github.com/authorizerdev/authorizer/pull/634), [#635](https://github.com/authorizerdev/authorizer/pull/635)).
- **`AuthorizerAdminService` gRPC + REST**: 32 admin operations — user management (`Users`, `User`, `UpdateUser`, `DeleteUser`, `InviteMembers`), verification requests, tokens, webhooks, email templates, audit logs, FGA model/tuples (`_fga_model`, `_fga_tuples`, `_add_fga_tuple`, `_delete_fga_tuple`), and admin session/meta — are now reachable over all three transports. Previously admin ops were GraphQL-only ([#631](https://github.com/authorizerdev/authorizer/pull/631)).
- **gRPC auth interceptor**: bearer-token / session-cookie authentication is applied uniformly by a gRPC server interceptor. The verified identity is attached as `authctx.Principal` so all handlers share a single auth path with no per-handler duplication ([#636](https://github.com/authorizerdev/authorizer/pull/636)).
- **Client metadata helpers** (`transport.MetaFromGRPC`): extract client ID, session token, and access token from gRPC incoming metadata using the same keys as the HTTP handlers, enabling consistent context propagation across transports ([#636](https://github.com/authorizerdev/authorizer/pull/636)).
- **WebAuthn / passkey registration and login**: users can register and authenticate with FIDO2 security keys or platform authenticators (Windows Hello, Touch ID, Face ID, etc.). Supports usernameless discoverable login, passkey as a second MFA factor, and multi-passkey management per user ([#671](https://github.com/authorizerdev/authorizer/pull/671)).
- **Enterprise SSO with SAML 2.0 Identity Provider**: Authorizer can now act as a SAML IdP so downstream SaaS applications (Zendesk, Notion, Tableau, etc.) can federate against Authorizer. Includes signing key rotation with overlap windows, attribute mapping, and strict ACS/entity-ID binding. Service Provider role (consuming upstream Okta/Entra) continues to work in parallel ([#691](https://github.com/authorizerdev/authorizer/pull/691)).
- **Multi-tenant SSO with verified email domains and home realm discovery**: Three-phase addition enabling tenant isolation and automatic IdP routing. Phase 1: org-scoped admin role (`authorizer:org_admin`) lets tenant admins manage their org's SAML/OIDC/SCIM and members without platform super-admin access. Phase 2: verified email domains per organization via DNS TXT challenge or super-admin assertion, with first-writer-wins atomicity across all 13 database providers. Phase 3: `/api/v1/org-discovery` endpoint routes login traffic to the appropriate tenant's IdP based on email domain, integrated into the `/app` login page with `--enable-org-discovery` (off by default) ([#672](https://github.com/authorizerdev/authorizer/pull/672), [#674](https://github.com/authorizerdev/authorizer/pull/674), [#675](https://github.com/authorizerdev/authorizer/pull/675)).
- **SCIM 2.0 group provisioning with OpenFGA role binding and SAML group assertions**: SCIM `/Groups` endpoint (full CRUD + PATCH) with RFC 7644 §3.5.2 patch semantics and real-world Entra/Okta deviations handled. Membership flows through OpenFGA tuples (`group:<org>/<id>#member@user:<uid>`), and group→role bindings via the existing userset pattern. SAML IdP automatically asserts group membership as multi-valued attributes in issued assertions, with cross-tenant containment gates to prevent group-name leakage across organizations ([#694](https://github.com/authorizerdev/authorizer/pull/694)).
- **Service accounts as first-class FGA subjects**: registered `kind=service_account` clients in `client_credentials` token flow now resolve to `service_account:<client_id>` in authorization checks instead of `user:<sub>`, enabling autonomous machine authorization. Opt-in via modeling (engine denies if the model lacks the `service_account` type), and scopes remain the issuance ceiling ([#665](https://github.com/authorizerdev/authorizer/pull/665)).
- **Trusted-issuer token review config in admin API**: new `_trusted_issuer_request_token_review` and `_revoke_trusted_issuer_token_review` mutations allow admins to request and revoke token review certificates per trusted issuer without restarting the server ([#667](https://github.com/authorizerdev/authorizer/pull/667)).
- **Server-side user search and org membership in dashboard**: `_users` query now accepts a search parameter supporting case-insensitive matching across email, name, and ID fields on all 13 database providers (native SQL/Mongo/Arango; O(n) scan with documented upgrade paths for DynamoDB/Cassandra). `OrgMember` type now includes email and name fields, so admin UIs can display human-readable member lists ([#678](https://github.com/authorizerdev/authorizer/pull/678), [#680](https://github.com/authorizerdev/authorizer/pull/680)).
- **Per-method MFA availability signals in meta**: new `is_totp_mfa_enabled`, `is_email_otp_mfa_enabled`, `is_sms_otp_mfa_enabled`, and `is_webauthn_enabled` fields on the public `meta` query allow login UIs to show only available MFA methods ([#681](https://github.com/authorizerdev/authorizer/pull/681)).
- **`authorizer_required_permissions_checks_total{endpoint, outcome}`**: per-endpoint Prometheus counter for FGA adoption + enforcement signal. Outcomes are `granted`, `denied`, `not_requested`, `error`. Endpoints are `session`, `validate_session`, `validate_jwt_token`. Alert on `outcome="error"` rising; it indicates a storage/validation failure preventing checks from completing ([#527](https://github.com/authorizerdev/authorizer/pull/527)).
- **`--rate-limit-fail-closed`**: when the rate-limit backend returns an error, respond with `503` instead of allowing the request (default remains fail-open).
- **`--metrics-host`**: bind address for the dedicated `/metrics` listener (default `127.0.0.1`). Use `0.0.0.0` when a scraper on another host/pod must reach the metrics port over the network; keep the metrics port off public ingress.
- **OIDC Discovery — `grant_types_supported` includes `implicit`**: honestly reflects that `/authorize` accepts `response_type=token` and `response_type=id_token`.
- **OIDC Discovery caching**: discovery document and JWKS are now cached server-side with strict expiry, reducing external provider request load during token validation for social logins like Twitter ([#668](https://github.com/authorizerdev/authorizer/pull/668)).
- **Graceful shutdown for background work**: detached goroutines that fire request side effects (email/SMS sends, webhook events, audit log writes) are now tracked and drained on shutdown instead of being silently killed mid-flight, and a panic inside one is recovered and logged instead of crashing the whole process ([#696](https://github.com/authorizerdev/authorizer/pull/696)).

### Changed

- **Admin dashboard UI migration from Chakra UI to shadcn/ui + Tailwind CSS**: Dashboard (`web/dashboard/`) completely modernized. Replaced Chakra UI v2 with shadcn/ui (Radix primitives) + Tailwind CSS v4. All TypeScript `any` types and `@ts-ignore` directives eliminated; full type safety on GraphQL responses, component props, and data models. Dead dependencies removed (react-draft-wysiwyg, @emotion, framer-motion, react-icons, focus-visible). 17 shadcn/ui-style components built on Radix; Authorizer branding (logo + blue-500) applied throughout. Cleaner tables, Sheet panels for forms, sonner toast notifications, skeleton loading states ([#605](https://github.com/authorizerdev/authorizer/pull/605)).
- **BREAKING — MFA behavior completely redesigned: on by default, optional per user, withheld token until setup complete.** MFA methods (TOTP, Email OTP, SMS OTP, WebAuthn) are now enabled by default and opted out via new `--disable-totp-login`, `--disable-email-otp`, `--disable-sms-otp`, and `--disable-webauthn-mfa` flags; the old `--enable-totp-login`, `--enable-mfa`, `--enable-email-otp`, and `--enable-sms-otp` flags are removed. Email and SMS OTP only take effect when their provider (SMTP / Twilio) is configured. Whether MFA is available is now derived from the enabled methods rather than a standalone flag, which fixes the case where MFA appeared "enabled" while every method was unavailable. **New token-withholding behavior:** when MFA is optional (`--enforce-mfa` default `false`), first-time users who haven't set up MFA no longer receive an immediate token followed by a setup offer — the token is withheld until the user completes enrollment or explicitly skips (remembered as `has_skipped_mfa_setup_at`). This withheld-token model now applies uniformly to password login, passkey login, signup, and social login. When `--enforce-mfa` is set, MFA is mandatory and un-skippable. **Email/SMS OTP now require explicit enrollment** (new `email_otp_mfa_setup`/`sms_otp_mfa_setup` mutations) before they can be used for MFA verification, fixing the previous behavior where they fired automatically for any user with a phone/email on file. **Admin recovery:** new `reset_mfa` operation on `_update_user` clears all MFA state and enrolled factors across all storage backends. **User-initiated lockout:** new `lock_mfa` mutation prevents future MFA enrollment (admin-recoverable); lockout is refused if a verified Email/SMS OTP factor exists as a fallback. **`--disable-mfa` one-way kill switch** disables MFA entirely regardless of per-method flags (does not affect WebAuthn, which is a separate login recipe) ([#682](https://github.com/authorizerdev/authorizer/pull/682), [#684](https://github.com/authorizerdev/authorizer/pull/684), [#685](https://github.com/authorizerdev/authorizer/pull/685), [#686](https://github.com/authorizerdev/authorizer/pull/686)).
- **License: relicensed from MIT to Apache License 2.0.** Per the CNCF IP Policy ([Charter §11(b)(iii)](https://github.com/cncf/foundation/blob/main/charter.md#11-ip-policy)), Authorizer's outbound code is now distributed under the Apache License 2.0. Existing copies distributed under the MIT License remain valid under their original grant; this change applies to the project's outbound license going forward. See [NOTICE](NOTICE) for attribution.
- **Fine-grained authorization is always enforcing.** The previously-proposed `--authorization-enforcement` flag and its dual `permissive`/`enforcing` modes were removed before shipping. `required_permissions` checks against an unmatched or denied `(resource, scope)` pair return `unauthorized`. There is no permissive "log but allow" mode.
- **Authz Prometheus shape**: `authorizer_authz_checks_total` has only a `result` label (`allowed|denied|unmatched|error`); `authorizer_authz_unmatched_total` has no labels.
- **Prometheus `/metrics`**: always served on a **dedicated** HTTP listener (`--metrics-host`:`--metrics-port`, default `127.0.0.1:8081`). **`--http-port` and `--metrics-port` must differ**; `/metrics` is not registered on the main Gin server.
- **HTTP metrics**: unmatched Gin routes use the fixed path label `unmatched` instead of the raw request URL (prevents cardinality attacks).
- **GraphQL metrics**: the `operation` label is now `anonymous` or `op_<sha256-prefix>` so client-supplied operation names cannot explode time-series cardinality.
- **Health/readiness JSON**: failure responses return a generic `error` string; details remain in server logs.
- **OAuth callback JSON**: generic OAuth-style error body on provider processing failure; details remain in logs.
- **`/playground`** is subject to the same per-IP rate limits as other routes (health and OIDC discovery paths stay exempt). **`/metrics`** is not on the main HTTP router.
- **BREAKING: `/userinfo` now strictly filters claims by scope per OIDC Core §5.4.** The endpoint returns only `sub` plus the claims permitted by the standard scope groups (`profile`, `email`, `phone`, `address`) encoded in the access token. Previously, `/userinfo` returned the full user object regardless of scopes. Clients that request only the `openid` scope but read profile/email claims from `/userinfo` **must** now request those scopes explicitly. See https://docs.authorizer.dev/core/oauth2-oidc for the full scope→claim mapping.
- **OAuth 2.1 standards compliance**: refresh-token reuse detection revokes the user's entire session family on replay (RFC 8707 compliance); `resource` parameter binding on authorization code flow (binds access token `aud` claim); new `--oauth21-strict` flag (default off) gates implicit-grant and PKCE-plain removal behind opt-in. New `GET /.well-known/oauth-authorization-server` thin alias of OIDC discovery for MCP compliance ([#693](https://github.com/authorizerdev/authorizer/pull/693)).

### Security

- **OIDC/OAuth2 specification compliance for Enterprise IdP integration**: `/authorize` now returns RFC 6749 error codes (invalid_request, unauthorized_client, unsupported_response_type) instead of freeform strings; errors after `redirect_uri` validation redirect to the RP per spec instead of returning JSON. ID tokens now include `auth_time` claim on all issuance paths (OIDC Core §2 requirement for `max_age`). Discovery endpoint advertises `"none"` in `token_endpoint_auth_methods_supported` for PKCE-only public clients. `token_type` normalized to `"Bearer"` (capitalized). In-memory state store enforced with 10-minute TTL; DB state store enforces 600-second read-time TTL. `Cache-Control` caching added to discovery endpoint ([#604](https://github.com/authorizerdev/authorizer/pull/604)).
- **RFC-compliant PKCE and redirect_uri security hardening**: S256 `code_challenge` now tolerates base64url padding (Auth0 compatibility). `client_secret` validation enforced whenever provided, even when PKCE is used (prevents secret bypass). `code_verifier` rejected when no `code_challenge` was registered (prevents PKCE bypass). `redirect_uri` URL-encoded in state to prevent `@@`-delimiter injection. `/oauth/token` now validates `redirect_uri` matches the `/authorize` registration (RFC 6749 §4.1.3). Authorize state removal is synchronous (prevents code reuse). Constant-time `redirect_uri` comparison ([#603](https://github.com/authorizerdev/authorizer/pull/603)).
- **Introspection authentication & backchannel SSRF hardening**: `/oauth/introspect` now requires `client_secret` when configured (previously omitting secret bypassed auth entirely). Timing-safe `crypto/subtle.ConstantTimeCompare` used for all secret validation. Backchannel logout SSRF fixed by routing through `SafeHTTPClient` (upfront DNS, IP pinning, rejects private/loopback). Session rollover goroutine errors now logged instead of silently discarded ([#606](https://github.com/authorizerdev/authorizer/pull/606)).
- **Session revocation on password reset**: when a user resets their password via email verification link or the recovery flow, all of their active sessions are immediately revoked, preventing unauthorized account access after a compromised password ([#669](https://github.com/authorizerdev/authorizer/pull/669), [#673](https://github.com/authorizerdev/authorizer/pull/673)).
- **Per-user TOTP brute-force lockout**: failed TOTP verification attempts are now tracked per user with temporary lockout after 5 failures (matching the existing per-user email/SMS OTP lockout), and recovery codes are hashed at rest using bcrypt (never stored plaintext) ([#670](https://github.com/authorizerdev/authorizer/pull/670)).
- **SAML ACS CSRF-origin exemption**: the SAML Assertion Consumer Service endpoint is now correctly exempted from strict CSRF Origin checking, since browser-based SAML POSTs from a different origin (the IdP) are expected and legitimate ([#666](https://github.com/authorizerdev/authorizer/pull/666)).
- **Trusted base URL + email/SMS OTP lockout**: new `--url` flag (`config.AuthorizerURL`) sets the single trusted source for the server's own URL used in email verification links, JWT `iss` claim, and OIDC discovery, preventing header-spoofing attacks that could redirect users to attacker-controlled sites while carrying single-use tokens. Email/SMS OTP verification now gets the same per-user brute-force lockout that TOTP already had ([#698](https://github.com/authorizerdev/authorizer/pull/698)).
- **Type-safe error handling in gRPC admin service**: admin service methods now return properly-typed errors (400 for validation, 409 for conflicts, etc.) instead of generic Internal errors (500), and public-method bypass is tightly scoped to only the public service and `AdminLogin` ([#700](https://github.com/authorizerdev/authorizer/pull/700)).
- **Atomic storage operations with transaction guards**: `UpdateUsers` empty-ids filter is now enforced across all 13 database providers (preventing silent full-table updates on Mongo/Arango/Cassandra/Couchbase/DynamoDB); cascade deletes (`DeleteOrganization`, `DeleteClient`, `DeleteWebhook`, `DeleteUser`) are now wrapped in transactions, rolling back on partial failure ([#699](https://github.com/authorizerdev/authorizer/pull/699)).

### Fixed

- **Public client_id now exposed in Client API type**: `Client` GraphQL type and proto now include `client_id` (distinct from surrogate `id`). Dashboard Clients page displays correct "Client ID" (the configured client_id, not the internal id). Seeded interactive client has immutable `client_id` from `--client-id` flag ([#664](https://github.com/authorizerdev/authorizer/pull/664)).
- **Nil-pointer panics in claim/header type assertions**: two unguarded type assertions on untrusted map values (email-verify redirect-uri and webhook-event headers) could panic and crash the process; now guarded with safe type coercion ([#701](https://github.com/authorizerdev/authorizer/pull/701)).
- **Dashboard and login UI crashes**: CSV file import error handling in dashboard, non-null assertion guards in InputField component, logout button event handling, and WCAG label association for home realm discovery email input ([#702](https://github.com/authorizerdev/authorizer/pull/702)).
- **OIDC ID token `at_hash`**: now correctly set to `base64url(sha256(access_token)[:16])` for all flows. Previously the implicit/token branch incorrectly set `at_hash` to the nonce value (OIDC Core §3.2.2.10).
- **OIDC ID token `nonce`**: now echoed in the ID token whenever it was supplied in the auth request, regardless of the flow used (OIDC Core §2).
- **Admin service error mapping and InviteMembers**: gRPC error responses now use proper status codes (not all `codes.Internal`); the gRPC `public` bypass is scoped correctly; `InviteMembers` no longer has redundant re-fetches, missing `continue` statements, or unbounded batch sizes ([#700](https://github.com/authorizerdev/authorizer/pull/700)).

### Removed

- **`authorizer_client_id_not_found_total`**: replaced by **`authorizer_client_id_header_missing_total`**, which matches the actual behavior (header omitted, request still allowed). Update dashboards and alerts accordingly.
- **OIDC Discovery — `registration_endpoint`**: previously pointed to the signup UI rather than an RFC 7591 dynamic client registration endpoint. It will return when RFC 7591 is implemented.

## [2.2.1-rc.0] - 2026-04-06

Pre-release. See [2.2.1-rc.0](https://github.com/authorizerdev/authorizer/releases/tag/2.2.1-rc.0) on GitHub.

### Added

- **CSRF protection** (middleware).
- **Per-IP rate limiting** with Redis and in-memory backends.
- **GraphQL query complexity limit**.
- **5-second execution timeout** for custom access token scripts.

### Security

- **Crypto**: AES-GCM with HKDF key derivation (replaces AES-CFB); RSA 4096, improved `DecryptRSA` error handling and base64-related naming; `crypto/rand` for HMAC key generation.
- **JWT / tokens**: Verify JWT algorithm in parse keyfunc; safe type assertions for claims; bearer extraction case-sensitivity fix; shorter session and refresh token lifetimes; reserved claim blocklist for custom token scripts.
- **Cookies**: `HttpOnly` on all cookies; reduced cookie max-age; `SameSite` on admin cookie (with broader security-header and CORS credential fixes).
- **OAuth / redirects**: Apple ID token signature verified via OIDC; `redirect_uri` validation hardened against open redirects and wildcard abuse.
- **GraphQL**: SSRF protection for `_test_endpoint`; constant-time admin secret comparison; user enumeration mitigated via generic error messages.
- **HTTP / parsers**: Host header validation to reduce injection risk.
- **Storage / DB**: Parameterized AQL in ArangoDB `UpdateUsers`; Cassandra client TLS verification enabled; GORM `AllowGlobalUpdate` disabled; `DeleteSession` implemented for SQL and ArangoDB.
- **Email / templates**: Explicit TLS `ServerName` for SMTP; `html/template` for email rendering (SSTI mitigation); `template.JS` XSS-related fix.
- **Webhooks**: SSRF protection, HMAC signatures, and response size limits.
- **Data exposure**: Password hash excluded from JSON serialization; JWKS no longer leaks HMAC keys.
- **Operational**: Sanitized errors, panics replaced with errors where appropriate; Dockerfiles hardened (defaults, signals, healthcheck); client ID audit logging and CSRF origin validation tightened.

### Fixed

- GitHub OAuth display name handling and **POST logout** behavior.
- MongoDB driver update and related compilation issues.
- Tests: custom script timeout coverage, client-ID metric behavior, and ArangoDB-related test hardening.

**Full changelog**: [2.2.0...2.2.1-rc.0](https://github.com/authorizerdev/authorizer/compare/2.2.0...2.2.1-rc.0)

## [2.2.0] - 2026-04-03

See [2.2.0](https://github.com/authorizerdev/authorizer/releases/tag/2.2.0) on GitHub.

### Added

- **Prometheus metrics**, **health** checks, and **readiness** HTTP endpoints ([#528](https://github.com/authorizerdev/authorizer/pull/528)).

**Full changelog**: [2.1.0...2.2.0](https://github.com/authorizerdev/authorizer/compare/2.1.0...2.2.0)

## [2.1.0] - 2026-04-03

See [2.1.0](https://github.com/authorizerdev/authorizer/releases/tag/2.1.0) on GitHub.

### Added

- **Structured audit logging** system.

### Changed

- **Audit logging** consolidated behind an `internal/audit` provider.

### Security

- **Open redirect**: stricter validation for `redirect_uri`.

**Full changelog**: [2.0.1...2.1.0](https://github.com/authorizerdev/authorizer/compare/2.0.1...2.1.0)

## [2.0.0] - 2025-02-28

### Added

- **CLI-based configuration**: All configuration is now passed at server start via CLI root arguments. No env store in cache or database.
- **New security flags**:
  - `--disable-admin-header-auth`: When `true`, server does not accept `X-Authorizer-Admin-Secret` header; only secure admin cookie is honored. Recommended for production.
  - `--enable-graphql-introspection`: Controls GraphQL introspection on `/graphql` (default `true`; set `false` for hardened production).
- **Metrics endpoint**: Metrics server on port 8081 (configurable via `--metrics-port`).
- **Restructured project layout**:
  - Root-level `main.go` and `cmd/` for CLI
  - `internal/` for core packages (config, graph, storage, etc.)
  - `web/app` and `web/dashboard` for embedded UIs
  - `web/templates` for HTML templates
- **Build outputs**: Binary named `authorizer`; output to `build/<os>/<arch>/authorizer`.
- **Docker improvements**:
  - Multi-arch builds (linux/amd64, linux/arm64)
  - `ENTRYPOINT [ "./authorizer" ]` for passing CLI args at runtime
  - Alpine 3.23 base images
- **Makefile targets**: `make dev`, `make bootstrap`, `make build-local-image`, `make build-push-image`.

### Changed

- **BREAKING**: Configuration is no longer read from `.env` or OS environment variables. Pass config via CLI flags.
- **BREAKING**: `--client-id` and `--client-secret` are **required**; server exits if missing.
- **BREAKING**: Deprecated mutations `_admin_signup`, `_update_env`, `_generate_jwt_keys` now return errors directing users to configure via CLI.
- **BREAKING**: Dashboard cannot update server configuration. Admin secret, JWT keys, and all env must be set at startup.
- **BREAKING**: Flag names use kebab-case (e.g. `--database-url` instead of `database_url`).
- **BREAKING**: Some inverted boolean flags (e.g. `DISABLE_LOGIN_PAGE` → `--enable-login-page` with `false` to disable).
- **BREAKING**: Go version requirement: >= 1.24 (see `go.mod`).
- **BREAKING**: Node.js >= 18 for web app and dashboard builds.
- Database provider template path: `internal/storage/db/provider_template` (was `server/db/providers/provider_template`).
- GraphQL schema and resolvers moved to `internal/graph/`.
- Tests moved to `internal/integration_tests/`; run with `go test -v ./...` from repo root.

### Deprecated

- `database_url`, `database_type`, `log_level`, `redis_url` flags (use kebab-case `--database-url`, etc.).
- `env_file` flag (no longer supported).

### Fixed

- Corrected Makefile `generate-db-template` and DB-specific test targets to use current project structure.
- Docker build and release workflow updated for v2 layout and binary name.

### Migration

See [MIGRATION.md](MIGRATION.md) for a detailed guide from v1 to v2.

---

## [1.x] - Legacy

Authorizer v1 used environment-based configuration stored in cache/DB and configurable via dashboard or `_update_env` mutation. For v1 documentation, see [docs.authorizer.dev](https://docs.authorizer.dev/) and the [v1 release branch](https://github.com/authorizerdev/authorizer).
