# Workload Identity Program — Master Spec

**Status:** Planning  
**Date:** 2026-06-18  
**Authors:** Principal Engineer, CTO, Security, Auth Expert

---

## 1. Executive Summary

Authorizer currently supports only human-facing OAuth2 flows (`authorization_code`, `refresh_token`). It has no way for services, AI agents, or machines to authenticate. This program adds machine/workload identity as a first-class primitive in 6 ordered phases.

**Dependency chain (must implement in order):**

```
Phase 1: Client Credentials + App Registration     ← FOUNDATION (unblocks everything)
Phase 2: RFC 8693 Token Exchange                   ← depends on Phase 1
Phase 3: Trusted External JWT Issuer Primitive     ← depends on Phase 1
Phase 4: Kubernetes SA Token Auth                  ← depends on Phase 3
Phase 5: SPIFFE JWT-SVID Auth (preview)            ← depends on Phase 3
Phase 6: X.509-SVID mTLS Auth                      ← depends on Phase 5
```

---

## 2. Standards Reference (Confirmed as of 2026-06-18)

| Spec | Status | Role in this program |
|---|---|---|
| RFC 6749 §4.4 — Client Credentials Grant | Finalized | Phase 1 grant type |
| RFC 7521 — Assertion Framework | Finalized | Basis for `client_assertion` |
| RFC 7523 — JWT as `client_assertion` | Finalized | Phases 3–5 client-auth mechanism |
| RFC 8693 — Token Exchange | Finalized | Phase 2 grant type |
| RFC 8707 — Resource Indicators | Finalized | Audience binding on all issued tokens |
| K8s SA OIDC/JWKS (upstream K8s docs) | Stable | Phase 4 key source |
| K8s TokenReview API | Stable | Phase 4 optional hardening |
| SPIFFE JWT-SVID (spiffe/spiffe standards) | Confirmed | Phase 5 identity format |
| SPIFFE X.509-SVID (spiffe/spiffe standards) | Confirmed | Phase 6 identity format |
| SPIFFE Trust Domain and Bundle | Confirmed | Phases 5–6 key management |
| `draft-schwenkschuster-oauth-spiffe-client-auth-00` | **EXPIRED 2026-01-02, individual draft** | Phase 5 reference only — **ship as preview** |
| RFC 9728 — Protected Resource Metadata | Finalized | **Does NOT apply here** (resource-server metadata only) |

---

## 3. New Database Schemas

All new entities follow the existing schema pattern from `internal/storage/schemas/webhook.go`. Every field uses multi-DB struct tags: `json`, `bson`, `cql`, `dynamo`, `gorm`.

### 3.1 App (`internal/storage/schemas/app.go`)

```go
package schemas

import (
    "encoding/json"
    "strings"

    "github.com/authorizerdev/authorizer/internal/graph/model"
    "github.com/authorizerdev/authorizer/internal/refs"
)

// App represents a registered machine/service application.
// Note: any change here must be reflected in the cassandradb provider
// (no model support for collection creation).
type App struct {
    Key         string  `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB key
    ID          string  `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
    Name        string  `json:"name" bson:"name" cql:"name" dynamo:"name"`
    Description *string `json:"description" bson:"description" cql:"description" dynamo:"description"`
    ClientID    string  `gorm:"uniqueIndex;type:char(36)" json:"client_id" bson:"client_id" cql:"client_id" dynamo:"client_id" index:"client_id,hash"`
    ClientSecret string  `json:"client_secret" bson:"client_secret" cql:"client_secret" dynamo:"client_secret"` // bcrypt-hashed; never returned in API responses
    // AppType is one of: "machine" (M2M, client_credentials only),
    // "web" (browser redirect flows), "native" (mobile/desktop).
    AppType        string  `json:"app_type" bson:"app_type" cql:"app_type" dynamo:"app_type"`
    // AllowedScopes is a comma-separated list of scopes this app may request.
    AllowedScopes  string  `json:"allowed_scopes" bson:"allowed_scopes" cql:"allowed_scopes" dynamo:"allowed_scopes"`
    IsActive       bool    `json:"is_active" bson:"is_active" cql:"is_active" dynamo:"is_active"`
    CreatedAt      int64   `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
    UpdatedAt      int64   `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIApp converts App to GraphQL model. Never exposes ClientSecret.
func (a *App) AsAPIApp() *model.App {
    id := a.ID
    if strings.Contains(id, Collections.App+"/") {
        id = strings.TrimPrefix(id, Collections.App+"/")
    }
    return &model.App{
        ID:            id,
        Name:          a.Name,
        Description:   a.Description,
        ClientID:      a.ClientID,
        AppType:       a.AppType,
        AllowedScopes: strings.Split(a.AllowedScopes, ","),
        IsActive:      a.IsActive,
        CreatedAt:     refs.NewInt64Ref(a.CreatedAt),
        UpdatedAt:     refs.NewInt64Ref(a.UpdatedAt),
    }
}
```

**ClientSecret handling:** store as bcrypt hash (cost 12). The raw secret is returned **once** at creation and **never again**. Rotation creates a new secret.

### 3.2 TrustedIssuer (`internal/storage/schemas/trusted_issuer.go`)

```go
package schemas

import (
    "strings"

    "github.com/authorizerdev/authorizer/internal/graph/model"
    "github.com/authorizerdev/authorizer/internal/refs"
)

// TrustedIssuer is an external JWT issuer whose tokens the system will accept
// as client credentials (via client_assertion / RFC 7523).
// Covers: Kubernetes SA tokens, SPIFFE JWT-SVIDs, cloud OIDC tokens.
//
// Note: any change here must be reflected in cassandradb provider.
type TrustedIssuer struct {
    Key  string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"`
    ID   string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
    // AppID links this issuer to an App. Tokens from this issuer authenticate AS that App.
    AppID        string  `json:"app_id" bson:"app_id" cql:"app_id" dynamo:"app_id" index:"app_id,hash"`
    Name         string  `json:"name" bson:"name" cql:"name" dynamo:"name"`
    // IssuerURL is the `iss` claim value in JWTs from this issuer,
    // OR the SPIFFE trust domain (e.g. "spiffe://example.org") for SPIFFE issuers.
    IssuerURL    string  `gorm:"uniqueIndex" json:"issuer_url" bson:"issuer_url" cql:"issuer_url" dynamo:"issuer_url" index:"issuer_url,hash"`
    // KeySourceType determines how JWKS are fetched:
    //   "oidc_discovery"       — fetch from <IssuerURL>/.well-known/openid-configuration
    //   "static_jwks_url"      — fetch directly from JWKSUrl (required for private clusters)
    //   "spiffe_bundle_endpoint" — fetch from SPIFFE bundle endpoint (Phase 5)
    KeySourceType string  `json:"key_source_type" bson:"key_source_type" cql:"key_source_type" dynamo:"key_source_type"`
    // JWKSUrl is required when KeySourceType is "static_jwks_url".
    JWKSUrl      *string `json:"jwks_url" bson:"jwks_url" cql:"jwks_url" dynamo:"jwks_url"`
    // ExpectedAud is the audience claim value the token MUST contain.
    // For K8s SA tokens this MUST be Authorizer's issuer URL (configured via --issuer-url flag).
    ExpectedAud  string  `json:"expected_aud" bson:"expected_aud" cql:"expected_aud" dynamo:"expected_aud"`
    // SubjectClaim is the JWT claim used to identify the workload (usually "sub").
    SubjectClaim string  `json:"subject_claim" bson:"subject_claim" cql:"subject_claim" dynamo:"subject_claim"`
    // IssuerType is one of: "kubernetes_sa" | "spiffe_jwt" | "oidc" | "cloud_oidc"
    IssuerType   string  `json:"issuer_type" bson:"issuer_type" cql:"issuer_type" dynamo:"issuer_type"`
    IsActive     bool    `json:"is_active" bson:"is_active" cql:"is_active" dynamo:"is_active"`
    // SpiffeRefreshHintSeconds controls JWKS refresh cadence for SPIFFE bundle endpoints.
    // Default 300 (5 min) if 0. Ignored for non-SPIFFE key sources.
    SpiffeRefreshHintSeconds int64  `json:"spiffe_refresh_hint_seconds" bson:"spiffe_refresh_hint_seconds" cql:"spiffe_refresh_hint_seconds" dynamo:"spiffe_refresh_hint_seconds"`
    CreatedAt    int64   `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
    UpdatedAt    int64   `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

func (t *TrustedIssuer) AsAPITrustedIssuer() *model.TrustedIssuer {
    id := t.ID
    if strings.Contains(id, Collections.TrustedIssuer+"/") {
        id = strings.TrimPrefix(id, Collections.TrustedIssuer+"/")
    }
    return &model.TrustedIssuer{
        ID:                       id,
        AppID:                    t.AppID,
        Name:                     t.Name,
        IssuerURL:                t.IssuerURL,
        KeySourceType:            t.KeySourceType,
        JWKSUrl:                  t.JWKSUrl,
        ExpectedAud:              t.ExpectedAud,
        SubjectClaim:             t.SubjectClaim,
        IssuerType:               t.IssuerType,
        IsActive:                 t.IsActive,
        SpiffeRefreshHintSeconds: refs.NewInt64Ref(t.SpiffeRefreshHintSeconds),
        CreatedAt:                refs.NewInt64Ref(t.CreatedAt),
        UpdatedAt:                refs.NewInt64Ref(t.UpdatedAt),
    }
}
```

### 3.3 Collections Registry addition (`internal/storage/schemas/model.go`)

Add to the existing `Collections` struct:
```go
App           = "authorizer_apps"
TrustedIssuer = "authorizer_trusted_issuers"
```

---

## 4. Storage Interface Additions (`internal/storage/provider.go`)

Add these method signatures to the `Provider` interface. Each DB provider must implement all of them.

```go
// ===== App methods =====

// AddApp creates a new application record.
AddApp(ctx context.Context, app *schemas.App) (*schemas.App, error)

// UpdateApp updates an existing application record.
UpdateApp(ctx context.Context, app *schemas.App) (*schemas.App, error)

// DeleteApp removes an application record.
DeleteApp(ctx context.Context, app *schemas.App) error

// GetAppByID fetches an app by its primary key.
GetAppByID(ctx context.Context, id string) (*schemas.App, error)

// GetAppByClientID fetches an app by its OAuth2 client_id.
GetAppByClientID(ctx context.Context, clientID string) (*schemas.App, error)

// ListApps returns a paginated list of all apps.
ListApps(ctx context.Context, pagination *model.Pagination) ([]*schemas.App, *model.Pagination, error)

// ===== TrustedIssuer methods =====

// AddTrustedIssuer creates a new trusted issuer record.
AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error)

// UpdateTrustedIssuer updates an existing trusted issuer.
UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error)

// DeleteTrustedIssuer removes a trusted issuer.
DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error)

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its issuer URL / trust domain.
GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error)

// ListTrustedIssuers returns all trusted issuers, optionally filtered by app_id.
ListTrustedIssuers(ctx context.Context, appID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error)
```

---

## 5. Database Provider Implementation Checklist

Every new entity must be implemented in **all 6 providers**. Follow the pattern of the corresponding `webhook.go` file in each provider directory.

| Provider | Directory | Pattern reference |
|---|---|---|
| SQL (postgres/sqlite/mysql/etc.) | `internal/storage/db/sql/` | `sql/webhook.go` |
| MongoDB | `internal/storage/db/mongodb/` | `mongodb/webhook.go` |
| ArangoDB | `internal/storage/db/arangodb/` | `arangodb/webhook.go` |
| CassandraDB/Scylla | `internal/storage/db/cassandradb/` | `cassandradb/webhook.go` |
| DynamoDB | `internal/storage/db/dynamodb/` | `dynamodb/webhook.go` |
| Couchbase | `internal/storage/db/couchbase/` | `couchbase/webhook.go` |

**SQL auto-migration:** Add `&schemas.App{}` and `&schemas.TrustedIssuer{}` to the `AutoMigrate` call in `internal/storage/db/sql/provider.go`.

**CassandraDB note:** CassandraDB does not use GORM AutoMigrate. Table DDL must be added manually in `cassandradb/provider.go`. See existing collection-creation code there for the pattern.

**DynamoDB note:** DynamoDB uses on-demand table creation. Add table-creation calls in `dynamodb/provider.go` following the existing pattern.

---

## 6. GraphQL Schema Additions (`internal/graph/schema.graphqls`)

Add these types, inputs, queries, and mutations. After editing, run `make generate-graphql`.

### New Types

```graphql
type App {
  id: ID!
  name: String!
  description: String
  client_id: String!
  # client_secret is NEVER returned after initial creation
  # It is returned once in CreateAppResponse only
  app_type: String!
  allowed_scopes: [String!]!
  is_active: Boolean!
  created_at: Int64
  updated_at: Int64
}

type CreateAppResponse {
  app: App!
  # client_secret returned ONCE at creation; store securely
  client_secret: String!
}

type Apps {
  pagination: Pagination!
  apps: [App!]!
}

type TrustedIssuer {
  id: ID!
  app_id: String!
  name: String!
  issuer_url: String!
  key_source_type: String!
  jwks_url: String
  expected_aud: String!
  subject_claim: String!
  issuer_type: String!
  is_active: Boolean!
  spiffe_refresh_hint_seconds: Int64
  created_at: Int64
  updated_at: Int64
}

type TrustedIssuers {
  pagination: Pagination!
  trusted_issuers: [TrustedIssuer!]!
}
```

### New Input Types

```graphql
input CreateAppRequest {
  name: String!
  description: String
  # app_type: "machine" | "web" | "native"
  app_type: String!
  allowed_scopes: [String!]!
}

input UpdateAppRequest {
  id: ID!
  name: String
  description: String
  allowed_scopes: [String!]
  is_active: Boolean
}

input AppRequest {
  id: ID!
}

input ListAppsRequest {
  pagination: PaginatedRequest
}

input AddTrustedIssuerRequest {
  app_id: String!
  name: String!
  issuer_url: String!
  # key_source_type: "oidc_discovery" | "static_jwks_url" | "spiffe_bundle_endpoint"
  key_source_type: String!
  jwks_url: String
  expected_aud: String!
  # subject_claim defaults to "sub" if omitted
  subject_claim: String
  # issuer_type: "kubernetes_sa" | "spiffe_jwt" | "oidc" | "cloud_oidc"
  issuer_type: String!
  spiffe_refresh_hint_seconds: Int64
}

input UpdateTrustedIssuerRequest {
  id: ID!
  name: String
  jwks_url: String
  expected_aud: String
  is_active: Boolean
  spiffe_refresh_hint_seconds: Int64
}

input TrustedIssuerRequest {
  id: ID!
}

input ListTrustedIssuersRequest {
  app_id: String
  pagination: PaginatedRequest
}

# Phase 2 — Token Exchange
input TokenExchangeRequest {
  subject_token: String!
  subject_token_type: String!
  requested_token_type: String
  resource: String
  audience: String
  scope: String
  actor_token: String
  actor_token_type: String
}
```

### New Admin Mutations (all `_` prefixed — admin only)

```graphql
# Phase 1
_create_app(params: CreateAppRequest!): CreateAppResponse!
_update_app(params: UpdateAppRequest!): App!
_delete_app(params: AppRequest!): Response!
_rotate_app_secret(params: AppRequest!): CreateAppResponse!

# Phases 3–5
_add_trusted_issuer(params: AddTrustedIssuerRequest!): TrustedIssuer!
_update_trusted_issuer(params: UpdateTrustedIssuerRequest!): TrustedIssuer!
_delete_trusted_issuer(params: TrustedIssuerRequest!): Response!
```

### New Admin Queries (all `_` prefixed)

```graphql
_app(params: AppRequest!): App!
_apps(params: ListAppsRequest): Apps!
_trusted_issuer(params: TrustedIssuerRequest!): TrustedIssuer!
_trusted_issuers(params: ListTrustedIssuersRequest): TrustedIssuers!
```

---

## 7. gRPC Proto Additions

### `proto/authorizer/v1/admin.proto` additions

```protobuf
// ===== App management =====
rpc CreateApp(CreateAppRequest) returns (CreateAppResponse);
rpc UpdateApp(UpdateAppRequest) returns (App);
rpc DeleteApp(AppRequest) returns (Response);
rpc RotateAppSecret(AppRequest) returns (CreateAppResponse);
rpc GetApp(AppRequest) returns (App);
rpc ListApps(ListAppsRequest) returns (Apps);

// ===== TrustedIssuer management =====
rpc AddTrustedIssuer(AddTrustedIssuerRequest) returns (TrustedIssuer);
rpc UpdateTrustedIssuer(UpdateTrustedIssuerRequest) returns (TrustedIssuer);
rpc DeleteTrustedIssuer(TrustedIssuerRequest) returns (Response);
rpc GetTrustedIssuer(TrustedIssuerRequest) returns (TrustedIssuer);
rpc ListTrustedIssuers(ListTrustedIssuersRequest) returns (TrustedIssuers);
```

After editing proto files, run `make proto-gen` and commit the `gen/` changes.

---

## 8. Token Endpoint Changes (`internal/http_handlers/token.go`)

### Phase 1 — client_credentials

Extend `RequestBody`:
```go
type RequestBody struct {
    // existing fields ...
    CodeVerifier string `form:"code_verifier" json:"code_verifier"`
    Code         string `form:"code" json:"code"`
    ClientID     string `form:"client_id" json:"client_id"`
    ClientSecret string `form:"client_secret" json:"client_secret"`
    GrantType    string `form:"grant_type" json:"grant_type"`
    RefreshToken string `form:"refresh_token" json:"refresh_token"`
    RedirectURI  string `form:"redirect_uri" json:"redirect_uri"`
    // Phase 1 additions:
    Scope        string `form:"scope" json:"scope"`
    // Phase 2 additions:
    SubjectToken     string `form:"subject_token" json:"subject_token"`
    SubjectTokenType string `form:"subject_token_type" json:"subject_token_type"`
    ActorToken       string `form:"actor_token" json:"actor_token"`
    ActorTokenType   string `form:"actor_token_type" json:"actor_token_type"`
    RequestedTokenType string `form:"requested_token_type" json:"requested_token_type"`
    Resource         string `form:"resource" json:"resource"`
    Audience         string `form:"audience" json:"audience"`
    // Phases 3–5 additions:
    ClientAssertion     string `form:"client_assertion" json:"client_assertion"`
    ClientAssertionType string `form:"client_assertion_type" json:"client_assertion_type"`
}
```

Extend the `grantType` switch to add:
```go
const (
    GrantTypeAuthorizationCode = "authorization_code"
    GrantTypeRefreshToken      = "refresh_token"
    GrantTypeClientCredentials = "client_credentials"                                        // Phase 1
    GrantTypeTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"           // Phase 2
)

const (
    ClientAssertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" // Phases 3–4
    ClientAssertionTypeJWTSPIFFE = "urn:ietf:params:oauth:client-assertion-type:jwt-spiffe" // Phase 5 (preview)
)
```

---

## 9. New Service Files

| File | Phase | Purpose |
|---|---|---|
| `internal/service/app.go` | 1 | App CRUD + secret management |
| `internal/service/token_exchange.go` | 2 | RFC 8693 token exchange logic |
| `internal/service/jwks_cache.go` | 3 | JWKS fetching, caching, refresh |
| `internal/service/workload_auth.go` | 3 | JWT client-assertion validation |
| `internal/service/kubernetes_sa_auth.go` | 4 | K8s SA token validation |
| `internal/service/spiffe_auth.go` | 5 | SPIFFE JWT-SVID validation |
| `internal/service/mtls_auth.go` | 6 | X.509-SVID mTLS extraction |

---

## 10. New Token Claims

### Phase 1 — client_credentials access token

```go
// AuthTokenConfig additions for machine tokens
type AuthTokenConfig struct {
    // existing fields ...
    // Phase 1: machine identity
    AppID      string   // set for client_credentials tokens
    AppScopes  []string // granted scopes for this token
    // Phase 2: delegation
    ActorSub   string   // subject of the actor token (act.sub claim)
    ActorISS   string   // issuer of the actor token (act.iss claim)
}
```

JWT `act` claim structure (RFC 8693 §4.1):
```json
{
  "sub": "service-account-a",
  "act": {
    "sub": "agent-xyz",
    "iss": "https://authorizer.example.com"
  }
}
```

The `act` claim must be added to the `reservedClaims` map in `internal/token/auth_token.go`.

---

## 11. JWKS Cache Design (`internal/service/jwks_cache.go`)

```go
// JWKSCache caches JWKS key sets per issuer URL.
// Thread-safe. Refreshes on TTL or on validation failure (key rotation).
type JWKSCache interface {
    // GetKeySet returns the current KeySet for the given issuer.
    // Fetches from remote if not cached or TTL expired.
    GetKeySet(ctx context.Context, issuer *schemas.TrustedIssuer) (jwk.Set, error)
    // InvalidateAndRefresh forces an immediate refresh (call on validation failure).
    InvalidateAndRefresh(ctx context.Context, issuerURL string) (jwk.Set, error)
}
```

**Library:** `github.com/lestrrat-go/jwx/v2` — provides `jwk.AutoRefresh` with configurable TTL. Already used widely in the Go ecosystem; zero-SPIFFE-dependency for the base JWKS cache. For SPIFFE bundle endpoints, use `github.com/spiffe/go-spiffe/v2` bundle packages.

**Refresh cadence:**
- OIDC discovery / static JWKS: refresh every 60 minutes or on key-not-found
- SPIFFE bundle endpoint: refresh every `spiffe_refresh_hint_seconds` (default 300s). This is a **hard runtime requirement** — failure to refresh will reject valid SVIDs after trust-bundle rotation.

---

## 12. Security Invariants (ALL phases must uphold)

These are non-negotiable. Every implementation must satisfy all of them.

### S1 — Audience binding (mandatory)
Every external JWT presented as a `client_assertion` MUST have an `aud` claim that exactly matches Authorizer's configured issuer URL (`--issuer-url`). Reject with `invalid_client` if absent or mismatched. Without this, any SA token minted for another service (e.g. Vault) can be replayed against Authorizer.

### S2 — Algorithm allow-list (mandatory)
Only accept: `RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`, `PS256`, `PS384`, `PS512`.
Reject `HS*` (symmetric — issuer and verifier share the secret, which breaks third-party issuer model), `none`, and any unrecognised algorithm. Return `invalid_client`.

### S3 — `exp` required (mandatory)
Every presented JWT MUST have an `exp` claim. Reject if absent. Reject if expired.

### S4 — SPIFFE alg constraint (mandatory for Phase 5)
SPIFFE JWT-SVIDs additionally restrict to RFC 7518 §3.3–3.5 asymmetric algorithms only. `alg:none` is explicitly forbidden by the SPIFFE JWT-SVID standard.

### S5 — X.509-SVID leaf constraints (mandatory for Phase 6)
When validating an X.509-SVID:
- `cA` MUST be `false`
- `digitalSignature` key usage MUST be set
- `keyCertSign` and `cRLSign` MUST NOT be set
- SPIFFE ID MUST be in URI SAN (exactly one SPIFFE URI SAN per leaf cert)
- SPIFFE ID path MUST be non-empty (reject root-path SVIDs)

### S6 — No client_secret in response after creation (mandatory)
`client_secret` MUST NOT appear in any API response except the initial `CreateApp` / `RotateAppSecret` response. Store only the bcrypt hash (cost 12).

### S7 — Offline-validation staleness gap (must document)
OIDC/JWKS offline validation does NOT verify the bound K8s object (Pod/SA) still exists. Document this explicitly in: the API response for TrustedIssuer creation, the operator guide, and any error messages. For Phase 4, offer an optional `enable_token_review` flag on the TrustedIssuer record.

### S8 — RFC 8707 audience on issued tokens (mandatory for Phase 2+)
Every Authorizer access token issued as a result of a workload-identity flow MUST carry the `resource` / `aud` claim. Confused-deputy mitigation is doubly critical for machine tokens that call many services.

### S9 — Scope attenuation (mandatory for Phase 2)
In token exchange, the issued token's scopes MUST be a subset of the subject token's scopes. Never grant more than what the subject token authorised.

### S10 — Static JWKS URL option (mandatory for Phase 3+)
Never require OIDC discovery as the only key-source option. Private K8s clusters cannot expose `/.well-known/openid-configuration` publicly. `key_source_type: "static_jwks_url"` with a `jwks_url` field is required from day one.

---

## 13. Token Endpoint Response Formats

### Phase 1 — `client_credentials` success (RFC 6749 §5.1)
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```
No `refresh_token` for machine tokens (machines re-authenticate on expiry).

### Phase 2 — token exchange success (RFC 8693 §2.2.1)
```json
{
  "access_token": "eyJ...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read"
}
```

### Phase 1–5 — error responses (RFC 6749 §5.2)
```json
{
  "error": "invalid_client",
  "error_description": "client_assertion signature verification failed"
}
```

Common error codes for this program:
- `invalid_client` — bad client_id/secret, bad assertion, expired token, audience mismatch
- `invalid_grant` — token exchange subject_token invalid or expired
- `unsupported_grant_type` — grant_type not supported
- `invalid_scope` — requested scope exceeds allowed_scopes for app
- `invalid_request` — missing required parameter

---

## 14. Audit Log Events

Add these to `internal/constants/` audit event constants:

```go
const (
    // Phase 1
    AuditEventAppCreated       = "app.created"
    AuditEventAppUpdated       = "app.updated"
    AuditEventAppDeleted       = "app.deleted"
    AuditEventAppSecretRotated = "app.secret_rotated"
    AuditEventClientCredentials = "token.client_credentials"

    // Phase 2
    AuditEventTokenExchange    = "token.exchange"

    // Phases 3–5
    AuditEventWorkloadAuth     = "token.workload_auth"
    AuditEventTrustedIssuerAdded   = "trusted_issuer.added"
    AuditEventTrustedIssuerUpdated = "trusted_issuer.updated"
    AuditEventTrustedIssuerDeleted = "trusted_issuer.deleted"
)
```

---

## 15. Go Dependencies (new)

| Package | Version | Purpose | Phase |
|---|---|---|---|
| `github.com/lestrrat-go/jwx/v2` | latest stable | JWKS fetch + JWT validation for external tokens | 3 |
| `github.com/spiffe/go-spiffe/v2` | latest stable | SPIFFE bundle endpoint fetching + JWT-SVID validation | 5 |
| `golang.org/x/crypto/bcrypt` | stdlib | Client secret hashing | 1 |

Note: `golang.org/x/crypto` is already a transitive dependency — no new import needed. Check `go.mod` before adding anything.

---

## 16. Implementation Agent Assignments

| Phase | Plan file | Recommended agent |
|---|---|---|
| 1 — Client Credentials + App Registration | `docs/superpowers/plans/2026-06-18-phase1-client-credentials-app-registration.md` | `principal-engineer` |
| 2 — RFC 8693 Token Exchange | `docs/superpowers/plans/2026-06-18-phase2-token-exchange.md` | `delegation-engineer` |
| 3 — Trusted JWT Issuer Primitive | `docs/superpowers/plans/2026-06-18-phase3-trusted-jwt-issuer.md` | `principal-engineer` |
| 4 — K8s SA Token Auth | `docs/superpowers/plans/2026-06-18-phase4-kubernetes-sa-auth.md` | `principal-engineer` |
| 5 — SPIFFE JWT-SVID Auth (preview) | `docs/superpowers/plans/2026-06-18-phase5-spiffe-jwt-svid.md` | `security-engineer` |
| 6 — X.509-SVID mTLS | `docs/superpowers/plans/2026-06-18-phase6-x509-mtls.md` | `security-engineer` |

Each plan file is self-contained and references this master spec. An agent implementing any phase MUST read this master spec first.

---

## 17. Branch Naming and PR Convention

- Phase 1: `feat/workload-identity-phase1-client-credentials`
- Phase 2: `feat/workload-identity-phase2-token-exchange`
- Phase 3: `feat/workload-identity-phase3-trusted-issuer`
- Phase 4: `feat/workload-identity-phase4-k8s-sa-auth`
- Phase 5: `feat/workload-identity-phase5-spiffe-jwt-svid`
- Phase 6: `feat/workload-identity-phase6-x509-mtls`

Each PR must pass: `make lint`, `make test-sqlite`, and CI green before merge. Phase 5 PR additionally requires `security-engineer` review.

---

## 18. What Does NOT Change

- Human auth flows (`authorization_code`, `refresh_token`, social logins, MFA, magic links) — untouched
- Existing `User` model — no changes
- OpenFGA integration — machine identities become new principals in FGA tuples; no schema change to FGA
- The existing single `ClientID`/`ClientSecret` global config — retained for backward compatibility; new per-App registration is additive
