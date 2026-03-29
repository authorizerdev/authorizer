# Authorizer v2 Roadmap

> Comprehensive roadmap based on competitive analysis of WorkOS, Clerk, Keycloak, and emerging standards (A2A, MCP, OAuth 2.1). Organized by priority phases.

---

## Current State Summary

| Capability | Status | Notes |
|---|---|---|
| Basic Auth (email/password, magic link, OTP) | Done | Solid foundation |
| Social OAuth (11 providers) | Done | Google, GitHub, Facebook, Apple, etc. |
| MFA/TOTP | Done | Google Authenticator compatible |
| JWT tokens (HS/RS/ES families) | Done | JWKS endpoint, custom claims |
| Webhooks (8 event types) | Done | No structured audit log |
| PKCE (RFC 7636) | Done | OAuth authorization code flow |
| Basic RBAC | Done | Comma-separated role strings, no permissions model |
| 13+ Database backends | Done | Unique differentiator |
| Rate Limiting / Brute Force | Missing | No protection at all |
| M2M / Client Credentials | Missing | No service accounts or API keys |
| Fine-Grained Permissions | Missing | Roles only, no resource-level control |
| SAML | Missing | Zero support |
| SCIM / Directory Sync | Missing | No provisioning |
| Audit Logs | Missing | Webhooks only, no queryable audit trail |
| Bot Detection | Missing | No CAPTCHA, no fingerprinting |
| Monitoring / Metrics | Missing | Health check only, no Prometheus |
| MCP Auth | Missing | No OAuth 2.1 AS capabilities |
| A2A / Agent Auth | Missing | No agent identity support |
| Passkeys / WebAuthn | Missing | Not implemented |
| Authorizer as OIDC Provider | Missing | Can consume OIDC, cannot provide |

---

## Phase 1: Security Hardening & Enterprise Foundation (Q2-Q3 2026)

These are table-stakes features that every competitor has. Without them, Authorizer cannot be recommended for production enterprise use.

### 1.1 Rate Limiting & Brute Force Protection

**Why**: Every competitor (WorkOS Radar, Clerk, Keycloak) has this. Currently zero protection against credential stuffing or brute force.

- [ ] **Configurable rate limiter middleware** (per-IP and per-user)
  - Token bucket or sliding window algorithm
  - Configurable via CLI flags: `--rate-limit-requests=100 --rate-limit-window=60s`
  - Storage: Redis (preferred) or in-memory with DB fallback
- [ ] **Account lockout after N failed attempts**
  - Configurable threshold (default: 10 attempts)
  - Configurable lockout duration (default: 15 minutes) with exponential backoff
  - Admin unlock via GraphQL mutation `_unlock_user`
  - Track `failed_login_count` and `locked_until` on User schema
- [ ] **IP-based blocking/allowlisting**
  - GraphQL mutations: `_add_ip_block`, `_remove_ip_block`, `_add_ip_allow`, `_remove_ip_allow`
  - Middleware checks before auth processing
- [ ] **Leaked password detection**
  - Integration with Have I Been Pwned k-Anonymity API (SHA-1 prefix lookup, no full password sent)
  - Check on signup and password change
  - Configurable: `--check-leaked-passwords=true`

### 1.2 Bot Detection & CAPTCHA

**Why**: WorkOS Radar and Clerk both have sophisticated bot protection. This is a top enterprise requirement.

- [ ] **Pluggable CAPTCHA integration**
  - Support Cloudflare Turnstile (free, privacy-friendly) and Google reCAPTCHA v3
  - Config flags: `--captcha-provider=turnstile --captcha-site-key=... --captcha-secret-key=...`
  - Server-side token verification on login/signup endpoints
  - GraphQL and REST endpoints accept `captcha_token` parameter
- [ ] **Honeypot fields** in default login/signup UI
- [ ] **Configurable challenge triggers**
  - Always, never, or risk-based (after N failed attempts from same IP)

### 1.3 Structured Audit Log System

**Why**: WorkOS, Keycloak, and Clerk all have audit logs. Required for SOC 2, HIPAA, GDPR compliance.

- [ ] **New `AuditLog` schema**
  ```
  id, timestamp, actor_id, actor_type (user|admin|system|service_account),
  action, resource_type, resource_id, ip_address, user_agent,
  metadata (JSON), organization_id
  ```
- [ ] **Capture all auth events** (extends current 8 webhook events)
  - `user.login_success`, `user.login_failed`, `user.signup`, `user.logout`
  - `user.password_changed`, `user.password_reset`, `user.mfa_enabled`, `user.mfa_disabled`
  - `user.locked`, `user.unlocked`, `user.deleted`, `user.updated`
  - `admin.user_created`, `admin.user_updated`, `admin.user_deleted`
  - `admin.role_assigned`, `admin.role_removed`
  - `admin.config_changed`, `admin.webhook_created`
  - `token.issued`, `token.revoked`, `token.refreshed`
  - `session.created`, `session.terminated`
- [ ] **GraphQL query**: `_audit_logs(filter: AuditLogFilter, pagination: Pagination): AuditLogResponse`
  - Filter by actor, action, resource, date range, IP
- [ ] **Retention policy**: `--audit-log-retention-days=90` (auto-cleanup)
- [ ] **Immutable storage**: Audit logs cannot be modified or deleted via API (only via retention policy)

### 1.4 Prometheus Metrics & Health

**Why**: Keycloak has full Prometheus/Grafana support. Essential for production deployments.

- [ ] **`/metrics` endpoint** (OpenMetrics/Prometheus format)
  - `authorizer_login_total{method,status}` -- login attempts by method and success/failure
  - `authorizer_signup_total{method,status}` -- signup attempts
  - `authorizer_token_issued_total{type}` -- tokens issued by type
  - `authorizer_active_sessions` -- current active sessions gauge
  - `authorizer_request_duration_seconds{endpoint,method}` -- request latency histogram
  - `authorizer_db_query_duration_seconds` -- database query latency
  - `authorizer_failed_login_total` -- failed logins (for alerting)
  - `authorizer_account_lockouts_total` -- lockout events
  - Go runtime metrics (goroutines, memory, GC)
- [ ] **Enhanced `/health` endpoint** returning JSON with component status
  ```json
  {"status": "healthy", "db": "ok", "redis": "ok", "uptime": "72h"}
  ```
- [ ] **Readiness/liveness probes** (`/healthz`, `/readyz`) for Kubernetes

### 1.5 Session Security Enhancements

- [ ] **Configurable session limits** per user (`--max-sessions-per-user=5`)
- [ ] **Unrecognized device notifications** (email alert when login from new device/IP)
  - Track `device_id` cookie + user agent hash
- [ ] **Session listing & remote revocation** via GraphQL
  - `_user_sessions(user_id): [Session]` and `_revoke_session(session_id)`
- [ ] **Admin impersonation**
  - `_impersonate_user(user_id, reason)` -- returns short-lived token (60 min max)
  - Logged in audit trail with impersonator metadata
  - Configurable: `--enable-impersonation=false` (off by default)

---

## Phase 2: Authorization & M2M (Q3-Q4 2026)

### 2.1 Fine-Grained Permissions Model

**Why**: WorkOS FGA, Keycloak Authorization Services, and Clerk all go beyond simple roles. This is the most requested enterprise feature.

- [ ] **New data model: Permissions and Resources**
  ```
  Permission: { id, name, description, created_at, updated_at }
  RolePermission: { role_id, permission_id }  -- many-to-many
  Resource: { id, type, name, organization_id }
  ResourcePermission: { resource_id, permission_id, role_id }
  ```
- [ ] **Permission naming convention**: `resource:action` (e.g., `documents:read`, `invoices:create`, `users:delete`)
- [ ] **GraphQL API for permission management**
  - `_add_permission`, `_update_permission`, `_delete_permission`, `_list_permissions`
  - `_assign_permission_to_role`, `_remove_permission_from_role`
  - `_check_permission(user_id, permission, resource_id?): Boolean`
- [ ] **Permissions in JWT claims** (configurable)
  - Access tokens include `permissions: ["documents:read", "documents:write"]`
  - Or a `roles_permissions` map: `{"admin": ["*"], "editor": ["documents:read", "documents:write"]}`
- [ ] **SDK helper**: `hasPermission(token, "documents:write")` for downstream services
- [ ] **Organization-scoped permissions** -- permissions can be global or scoped to an org

### 2.2 Machine-to-Machine Authentication

**Why**: WorkOS M2M, Clerk M2M tokens, Keycloak client credentials. Core requirement for any auth platform.

- [ ] **OAuth 2.0 Client Credentials Grant** (`grant_type=client_credentials`)
  - New endpoint: `POST /oauth/token`
  - Client authenticates with `client_id` + `client_secret`
  - Returns short-lived JWT access token with configurable scopes
- [ ] **Service Account / Application schema**
  ```
  Application: { id, name, client_id, client_secret_hash, scopes,
                 organization_id, is_active, created_by, created_at }
  ```
- [ ] **GraphQL admin API**
  - `_create_application(name, scopes, organization_id): Application`
  - `_list_applications`, `_update_application`, `_delete_application`
  - `_rotate_application_secret(application_id): {client_id, client_secret}`
- [ ] **Scoped access tokens** -- applications request specific scopes, token contains only granted scopes
- [ ] **Rate limiting per application** -- separate limits for M2M clients
- [ ] **Audit logging** -- all M2M token issuance and usage logged

### 2.3 API Key Management

**Why**: WorkOS API Keys widget, Clerk API Keys. Enables end-users to create programmatic access.

- [ ] **API Key schema**
  ```
  APIKey: { id, name, key_hash, key_prefix (first 8 chars for identification),
            user_id, organization_id, permissions[], expires_at,
            last_used_at, is_active, created_at }
  ```
- [ ] **GraphQL API**
  - `create_api_key(name, permissions, expires_at): {key, key_id}` (key shown once)
  - `list_api_keys`: returns masked keys with metadata
  - `revoke_api_key(key_id)`
- [ ] **Authentication middleware** -- accept `Authorization: Bearer ak_...` header
  - Validate key hash, check permissions, check expiry
  - Populate request context with user/org identity from key
- [ ] **Key rotation support** -- create new key before revoking old one

### 2.4 Organization / Multi-Tenancy Enhancements

**Why**: WorkOS Organizations, Clerk Organizations, Keycloak Organizations/Realms. B2B requires this.

- [ ] **Organization schema** (if not already robust)
  ```
  Organization: { id, name, slug, domain, logo_url, metadata,
                  settings (JSON), created_at, updated_at }
  OrganizationMember: { org_id, user_id, role, joined_at }
  OrganizationInvitation: { org_id, email, role, token, expires_at, status }
  ```
- [ ] **Organization-scoped auth** -- tokens include `org_id` claim
- [ ] **Domain-based routing** -- users with `@company.com` auto-routed to org's SSO
- [ ] **Org-level settings** -- each org can configure its own MFA policy, password policy, allowed auth methods
- [ ] **GraphQL API**: `_create_organization`, `_invite_to_organization`, `_list_organization_members`, `_update_member_role`

---

## Phase 3: Enterprise SSO & Federation (Q4 2026 - Q1 2027)

### 3.1 SAML 2.0 Support

**Why**: WorkOS, Keycloak both support SAML. Required for enterprise customers using Okta, Azure AD, OneLogin.

- [ ] **SAML Service Provider (SP)** -- Authorizer acts as SP, enterprise IdPs (Okta, Azure AD) as IdP
  - SP metadata endpoint: `/.well-known/saml-metadata`
  - ACS (Assertion Consumer Service) endpoint
  - SP-initiated and IdP-initiated SSO flows
  - SAML assertion parsing, signature validation, attribute mapping
- [ ] **Per-organization SAML connections**
  - Each org can configure its own SAML IdP
  - Admin portal for IT admins to upload IdP metadata / configure manually
- [ ] **Attribute mapping** -- map SAML attributes to Authorizer user fields (email, name, roles)
- [ ] **Library**: Use `crewjam/saml` (Go) for SAML implementation

### 3.2 SCIM 2.0 / Directory Sync

**Why**: WorkOS Directory Sync, Keycloak LDAP federation. Enables automated user provisioning from enterprise directories.

- [ ] **SCIM 2.0 server endpoints**
  - `GET /scim/v2/Users` -- list/filter users
  - `POST /scim/v2/Users` -- create user
  - `GET /scim/v2/Users/:id` -- get user
  - `PUT /scim/v2/Users/:id` -- replace user
  - `PATCH /scim/v2/Users/:id` -- update user attributes
  - `DELETE /scim/v2/Users/:id` -- deactivate user
  - `GET /scim/v2/Groups` -- list groups (map to roles)
  - `POST /scim/v2/Groups` -- create group
  - `PATCH /scim/v2/Groups/:id` -- update group membership
- [ ] **Bearer token auth** for SCIM endpoint (per-organization SCIM token)
- [ ] **Attribute mapping** -- SCIM attributes to Authorizer user schema
- [ ] **Webhook events** for provisioning: `user.provisioned`, `user.deprovisioned`, `group.updated`

### 3.3 Authorizer as OIDC Provider

**Why**: WorkOS Connect, Clerk as IdP, Keycloak as IdP. Enables downstream services to use Authorizer for SSO.

- [ ] **Full OIDC Provider implementation**
  - Authorization endpoint: `/oauth/authorize`
  - Token endpoint: `/oauth/token`
  - UserInfo endpoint: `/oauth/userinfo`
  - Discovery: `/.well-known/openid-configuration` (enhance existing)
  - JWKS: `/.well-known/jwks.json` (already exists)
- [ ] **Client registration** -- register third-party applications that authenticate against Authorizer
  - Confidential clients (server-side) and public clients (SPAs, mobile)
  - Redirect URI validation, scope management
- [ ] **Consent screen** -- user approves what data the third-party app can access
- [ ] **Standard scopes**: `openid`, `profile`, `email`, `roles`, `permissions`, `org`
- [ ] **Grant types**: authorization_code (with PKCE), client_credentials, refresh_token

### 3.4 Admin Portal (Self-Service)

**Why**: WorkOS Admin Portal is a key differentiator. Lets customer IT admins configure SSO without engineering support.

- [ ] **Embeddable/hosted admin portal** for organization IT admins
  - Configure SAML/OIDC connections
  - Set up SCIM directory sync
  - View audit logs for their organization
  - Manage domain verification (DNS TXT record)
  - Configure MFA policy for their organization
- [ ] **Portal access** via time-limited link generated by admin API
- [ ] **White-label theming** -- custom logo, colors

---

## Phase 4: AI-Era Auth - MCP, A2A, Agent Identity (Q1-Q2 2027)

### 4.1 OAuth 2.1 Authorization Server for MCP

**Why**: WorkOS, Keycloak 26.4, and Clerk all support MCP auth. This is the fastest-growing auth use case in 2026.

- [ ] **OAuth 2.1 compliance**
  - Mandatory PKCE (S256) on all authorization code flows
  - Remove implicit grant support
  - Refresh token rotation
- [ ] **Authorization Server Metadata (RFC 8414)**
  - `/.well-known/oauth-authorization-server` endpoint
  - Publishes supported grant types, scopes, response types, token endpoint auth methods
- [ ] **Dynamic Client Registration (RFC 7591)**
  - `POST /oauth/register` -- MCP clients register programmatically
  - Returns `client_id` (and optionally `client_secret` for confidential clients)
  - Registration access token for subsequent client management
- [ ] **Resource Indicators (RFC 8707)**
  - `resource` parameter in authorization and token requests
  - Token audience (`aud`) set to the target MCP server URL
  - Prevents token reuse across different resource servers
- [ ] **Protected Resource Metadata**
  - MCP servers can publish `/.well-known/oauth-protected-resource` pointing to Authorizer as the AS
- [ ] **Tool-level permission scopes**
  - Define MCP tool permissions as OAuth scopes (e.g., `mcp:tool:read_file`, `mcp:tool:execute_query`)
  - Consent screen shows which tools the agent is requesting access to

### 4.2 Agent-to-Agent (A2A) Protocol Support

**Why**: Google A2A protocol backed by 50+ partners. Authorizer can be the identity backbone for agent ecosystems.

- [ ] **Agent identity via service accounts**
  - Each AI agent registered as a service account with `agent_type` metadata
  - Agent Card generation: `/.well-known/agent.json` for agents using Authorizer
- [ ] **Agent-specific OAuth scopes**
  - `agent:invoke`, `agent:delegate`, `agent:read_context`
  - Scope-limited tokens prevent agents from exceeding their authority
- [ ] **Short-lived agent tokens**
  - Default 5-minute expiry for agent-to-agent tokens
  - Automatic refresh via client credentials
- [ ] **Token exchange (RFC 8693)** for agent delegation
  - Agent A can exchange its token for a token valid for Agent B's scope
  - Delegation chain tracking in audit logs
- [ ] **Agent audit trail**
  - All agent actions logged with: agent_id, delegating_user_id, tool_invoked, scope_used

### 4.3 User-Delegated Agent Access

**Why**: WorkOS and Clerk both support this pattern. Users authorize AI agents to act on their behalf with limited scope.

- [ ] **Delegated authorization flow**
  - User authenticates via standard OAuth flow
  - User consents to specific scopes for the agent
  - Agent receives a scoped token that can only access what user approved
- [ ] **Scope downscoping** -- agent token permissions are intersection of user's permissions and requested scopes
- [ ] **Revocation** -- user can revoke agent access at any time via dashboard
- [ ] **Active agent sessions** -- users see which agents have active tokens in their account settings

---

## Phase 5: Advanced Security & Enterprise (Q2-Q3 2027)

### 5.1 Passkeys / WebAuthn

**Why**: Keycloak 26.4, Clerk both support passkeys. Industry moving toward passwordless.

- [ ] **WebAuthn registration and authentication** (FIDO2 / Passkeys)
  - Registration: `POST /webauthn/register/begin` + `POST /webauthn/register/finish`
  - Authentication: `POST /webauthn/login/begin` + `POST /webauthn/login/finish`
  - Library: `go-webauthn/webauthn` (Go)
- [ ] **Passkey schema**
  ```
  WebAuthnCredential: { id, user_id, credential_id, public_key, attestation_type,
                        transport[], sign_count, aaguid, created_at, last_used_at, name }
  ```
- [ ] **Multiple passkeys per user** (up to 10)
- [ ] **Passkey as primary auth** (skip password) or as MFA second factor
- [ ] **Conditional UI** -- auto-suggest passkey login when available

### 5.2 DPoP (Proof-of-Possession Tokens)

**Why**: Keycloak 26.4 has full DPoP, FAPI 2.0 requires it. Prevents stolen token replay.

- [ ] **DPoP token binding (RFC 9449)**
  - Client sends `DPoP` header (signed JWT proving key possession) with token requests
  - Access tokens include `cnf.jkt` (JWK thumbprint) claim
  - Resource servers validate DPoP proof matches token binding
- [ ] **Configurable enforcement** -- `--require-dpop=false` (opt-in per client)

### 5.3 Advanced Bot Protection (Radar-style)

**Why**: WorkOS Radar is a major differentiator. Goes beyond CAPTCHA.

- [ ] **Device fingerprinting**
  - Client-side JS collects browser/OS/hardware signals
  - Server-side fingerprint storage and comparison
  - `DeviceFingerprint` schema: `{ id, user_id, fingerprint_hash, first_seen, last_seen, trust_level }`
- [ ] **Risk scoring engine**
  - Score based on: IP reputation, device fingerprint match, geo-location anomaly, login velocity, time-of-day patterns
  - Configurable thresholds for: allow, challenge (MFA/CAPTCHA), block
- [ ] **Progressive rate limiting** tied to fingerprints (not just IP)
  - Prevents attackers from bypassing limits by rotating IPs
- [ ] **Credential stuffing detection**
  - Alert when high volume of failed logins from single IP across multiple accounts
- [ ] **New device alerts** -- email notification with device info, location, sign-in method

### 5.4 Log Streaming & SIEM Integration

**Why**: WorkOS Log Streams. Enterprise customers need to feed auth events into their existing security tooling.

- [ ] **Log stream configuration**
  - Stream audit events to external destinations in real-time
  - Supported destinations: HTTP webhook, AWS S3, Datadog, Splunk HEC
  - GraphQL admin API: `_create_log_stream`, `_list_log_streams`, `_delete_log_stream`
- [ ] **Structured event format** (JSON)
  ```json
  {
    "id": "evt_...", "timestamp": "2026-...", "type": "user.login_success",
    "actor": {"id": "usr_...", "type": "user", "ip": "1.2.3.4"},
    "target": {"type": "session", "id": "sess_..."},
    "metadata": {"method": "password", "mfa": true, "device_id": "dev_..."},
    "organization_id": "org_..."
  }
  ```
- [ ] **Batching and retry** -- buffer events, batch delivery, retry on failure with exponential backoff

### 5.5 Token Exchange (RFC 8693)

**Why**: Keycloak 26.2 added this. Enables complex microservice and agent delegation patterns.

- [ ] **Standard token exchange endpoint** at `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
- [ ] **Supported exchange patterns**
  - Delegation: User token -> service-scoped token
  - Impersonation: Admin token -> user token (with audit)
  - Cross-organization: Token valid for Org A -> token valid for Org B (if user is member of both)

---

## Phase 6: Developer Experience & Polish (Ongoing)

### 6.1 SDKs & Libraries

- [ ] **Go SDK** -- `authorizer-go` with typed client for all APIs
- [ ] **Node.js/TypeScript SDK** -- `authorizer-js` (enhance existing)
- [ ] **Python SDK** -- `authorizer-python`
- [ ] **React components** -- pre-built login, signup, org switcher, API key manager, session manager
- [ ] **Permission check middleware** for popular frameworks (Gin, Express, FastAPI, Next.js)

### 6.2 Dashboard Enhancements

- [ ] **Audit log viewer** -- search, filter, export in dashboard
- [ ] **Metrics dashboard** -- login activity charts, signup trends, active sessions
- [ ] **Application management UI** -- create/manage M2M apps, API keys
- [ ] **Organization management UI** -- create orgs, manage members, configure SSO per org
- [ ] **Permission designer** -- visual role-permission matrix editor

### 6.3 Compliance & Certification

- [ ] **SOC 2 Type 2** -- audit log + access controls + encryption enable compliance
- [ ] **GDPR** -- data export endpoint (`_export_user_data`), right to deletion, consent tracking
- [ ] **HIPAA** -- audit logging, session management, encryption at rest documentation
- [ ] **OpenID Certification** -- get official certification as OIDC Provider

---

## Priority Matrix

| Feature | Business Impact | Engineering Effort | Priority |
|---|---|---|---|
| Rate Limiting & Brute Force | Critical | Medium | P0 |
| Audit Logs | Critical | Medium | P0 |
| Prometheus Metrics | High | Low | P0 |
| M2M / Client Credentials | Critical | Medium | P0 |
| Fine-Grained Permissions | Critical | High | P1 |
| CAPTCHA / Bot Detection | High | Low | P1 |
| API Key Management | High | Medium | P1 |
| Organizations Enhancement | High | Medium | P1 |
| SAML 2.0 | High | High | P1 |
| OAuth 2.1 / MCP Auth | High | High | P2 |
| SCIM / Directory Sync | High | High | P2 |
| OIDC Provider | High | High | P2 |
| Passkeys / WebAuthn | Medium | Medium | P2 |
| A2A Agent Auth | Medium | Medium | P2 |
| DPoP | Medium | Medium | P3 |
| Risk Scoring / Device Fingerprinting | Medium | High | P3 |
| Log Streaming / SIEM | Medium | Medium | P3 |
| Token Exchange (RFC 8693) | Medium | Medium | P3 |
| Admin Portal (Self-Service) | Medium | High | P3 |

---

## Technical Dependencies

```
Phase 1 (no deps - can start immediately)
├── Rate Limiting
├── Audit Logs
├── Metrics
├── Bot Detection
└── Session Enhancements

Phase 2 (depends on Phase 1 audit logs)
├── Permissions Model (depends on: audit logs)
├── M2M Auth (depends on: rate limiting, audit logs)
├── API Keys (depends on: permissions model)
└── Organizations (depends on: permissions model)

Phase 3 (depends on Phase 2 OIDC provider + orgs)
├── SAML (depends on: organizations)
├── SCIM (depends on: organizations)
├── OIDC Provider (depends on: M2M auth, permissions)
└── Admin Portal (depends on: SAML, SCIM, organizations)

Phase 4 (depends on Phase 3 OIDC provider)
├── MCP Auth (depends on: OIDC provider, M2M auth)
├── A2A Auth (depends on: M2M auth, token exchange)
└── Delegated Agent Access (depends on: MCP auth, permissions)

Phase 5 (can partially parallelize with Phase 3-4)
├── Passkeys (no hard deps)
├── DPoP (depends on: OIDC provider)
├── Advanced Bot Protection (depends on: audit logs)
├── Log Streaming (depends on: audit logs)
└── Token Exchange (depends on: OIDC provider)
```

---

## Competitive Positioning

### vs. WorkOS
WorkOS is SaaS-only, expensive at scale ($125/SSO connection/month). Authorizer's advantage is **self-hosted, open-source, database-agnostic**. Closing the feature gap in M2M, FGA, SAML, MCP auth, and bot protection makes Authorizer a credible self-hosted WorkOS alternative.

### vs. Clerk
Clerk is developer-experience-first but SaaS-only with vendor lock-in. Authorizer can match their DX with better SDKs and pre-built UI components while offering self-hosting and data sovereignty.

### vs. Keycloak
Keycloak is the closest competitor (open-source, self-hosted). Authorizer's advantages: **simpler deployment** (single binary vs. Java app server), **13+ database backends** (vs. 5 SQL-only), **GraphQL API** (vs. REST-only), **lighter resource footprint**. The gap is in enterprise features (SAML, SCIM, Authorization Services, passkeys, DPoP) which this roadmap addresses.

### Unique Differentiators to Maintain
1. **Database-agnostic** -- 13+ backends including NoSQL (MongoDB, DynamoDB, Cassandra, Couchbase, ArangoDB)
2. **Single binary deployment** -- no JVM, no app server, no dependencies
3. **GraphQL-first API** -- modern API design, introspectable schema
4. **CLI-driven configuration** -- no .env files, 12-factor app principles
5. **Lightweight** -- suitable for edge deployment, IoT, resource-constrained environments

---

## Key Standards & RFCs to Implement

| Standard | Phase | Purpose |
|---|---|---|
| OAuth 2.1 (draft) | 4 | Modern OAuth baseline (PKCE mandatory, no implicit) |
| RFC 7636 (PKCE) | Done | Proof Key for Code Exchange |
| RFC 7591 (DCR) | 4 | Dynamic Client Registration for MCP |
| RFC 8414 (AS Metadata) | 4 | Authorization Server discovery |
| RFC 8693 (Token Exchange) | 5 | Agent delegation, cross-service tokens |
| RFC 8707 (Resource Indicators) | 4 | Audience-restricted tokens for MCP |
| RFC 9449 (DPoP) | 5 | Proof-of-possession tokens |
| SAML 2.0 | 3 | Enterprise SSO |
| SCIM 2.0 (RFC 7644) | 3 | Directory provisioning |
| FIDO2 / WebAuthn | 5 | Passkeys |
| A2A Protocol | 4 | Google's agent interoperability |
| MCP Auth Spec | 4 | AI tool server authorization |

---

*Last updated: 2026-03-27*
*Based on analysis of: WorkOS, Clerk, Keycloak, and emerging A2A/MCP standards*
