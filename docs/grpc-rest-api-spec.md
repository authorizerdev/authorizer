# Specification: gRPC & REST API with gRPC-Gateway

This document specifies the architecture and mapping for exposing Authorizer's public APIs via gRPC and a generated RESTful layer using `grpc-gateway`.

## 1. Architecture Overview

Authorizer uses a single-source-of-truth Protobuf definition to generate:
1.  **gRPC Service**: For high-performance, typed backend-to-backend communication.
2.  **RESTful API**: Via `grpc-gateway`, maintaining backward compatibility and ease of use for web/mobile clients.
3.  **TypeScript Clients**: Via `ts-proto`, providing type-safe API consumption out of the box.

### Package & Versioning
- **Package**: `authorizer.v1`
- **Directory Structure**: `proto/authorizer/v1/`
- **API Versioning**: Hardcoded in HTTP paths as `/v1/...` and tracked via Protobuf package versioning.

### REST Naming Conventions (Stripe-aligned)

Authorizer's REST surface follows the **Stripe "gold standard" REST conventions**. Stripe is widely regarded as the benchmark for developer-facing REST API design, and aligning with it keeps the surface consistent and unsurprising:

1. **`snake_case` everywhere — paths, query parameters, and JSON bodies.**
   Multi-word path segments use underscores, e.g. `/v1/magic_link_login`,
   `/v1/verify_email`, `/v1/validate_jwt_token`. This mirrors Stripe's own
   paths (`/v1/payment_intents`, `/v1/setup_intents`) and, critically, keeps a
   **single** naming style across the entire product: the path segment, the
   gRPC method identifier, the GraphQL operation name, and every JSON field are
   all `snake_case`. (The gateway sets `UseProtoNames=true` so response bodies
   stay `snake_case` rather than protobuf-default `camelCase`.)

   > We deliberately do **not** use `kebab-case` paths. While some guides
   > (Microsoft, Google AIP) and Keycloak prefer hyphens, mixing hyphenated
   > paths with `snake_case` bodies/operations would introduce a second naming
   > convention. Internal consistency wins; Stripe and Auth0
   > (`/dbconnections/change_password`) set the precedent.

2. **HTTP method reflects effect.** `GET` is reserved for safe, side-effect-free
   reads (`meta`, `profile`, `permissions`). Anything that mutates server state
   — including `logout` (it clears the session and is audit-logged) — uses
   `POST`. A mutating `GET` would violate RFC 9110 §9.2.1 and expose the
   endpoint to CSRF.

3. **Path prefix `/v1`** (not `/api/v1`) — the version is the first segment,
   matching Stripe's `/v1/...`.

4. **Stable, snake_case error envelope** on every `/v1` endpoint:

   ```json
   { "code": "invalid_argument", "message": "email or phone number is required" }
   ```

   The HTTP status is derived from the gRPC status code
   (`invalid_argument`→400, `unauthenticated`→401, `permission_denied`→403,
   `not_found`→404, `failed_precondition`→400, `internal`→500). The service
   layer classifies each error with a transport-neutral `ErrorKind`
   (`internal/service/errors.go`); the gRPC `ErrorMap` interceptor turns that
   into a status code, and grpc-gateway maps the code to the HTTP status.

### Ecosystem Tooling
- **`buf`**: Managed build system for Protobuf.
- **`protoc-gen-grpc-gateway`**: Generates the reverse proxy from gRPC to REST.
- **`protoc-gen-openapiv2`**: Generates Swagger/OpenAPI documentation.
- **`protoc-gen-ts_proto`**: Generates the TypeScript client.
- **`protovalidate`**: High-performance validation rules using Common Expression Language (CEL).

---

## 2. API Mapping (Public Surface)

All public GraphQL queries and mutations are mapped to RPC methods. Terminologies are preserved from the GraphQL schema.

### 2.1. Authentication Service (`authorizer.v1.AuthorizerService`)

Paths are `snake_case` under `/v1` (see "REST Naming Conventions" above). The
gRPC method name is the `PascalCase` form of the same identifier.

| RPC Method | GraphQL Equivalent | HTTP Path | Permissions |
| :--- | :--- | :--- | :--- |
| `Signup` | `signup` | `POST /v1/signup` | Public |
| `Login` | `login` | `POST /v1/login` | Public |
| `MagicLinkLogin` | `magic_link_login` | `POST /v1/magic_link_login` | Public |
| `Logout` | `logout` | `POST /v1/logout` | Authenticated |
| `VerifyEmail` | `verify_email` | `POST /v1/verify_email` | Public |
| `ResendVerifyEmail` | `resend_verify_email` | `POST /v1/resend_verify_email` | Public |
| `ForgotPassword` | `forgot_password` | `POST /v1/forgot_password` | Public |
| `ResetPassword` | `reset_password` | `POST /v1/reset_password` | Public |
| `VerifyOtp` | `verify_otp` | `POST /v1/verify_otp` | Public |
| `ResendOtp` | `resend_otp` | `POST /v1/resend_otp` | Public |
| `Revoke` | `revoke` | `POST /v1/revoke` | Authenticated |
| `UpdateProfile` | `update_profile` | `POST /v1/update_profile` | Authenticated |
| `DeactivateAccount` | `deactivate_account` | `POST /v1/deactivate_account` | Authenticated |
| `Meta` | `meta` | `GET /v1/meta` | Public |
| `Session` | `session` | `POST /v1/session` | Authenticated |
| `Profile` | `profile` | `GET /v1/profile` | Authenticated |
| `ValidateJwtToken` | `validate_jwt_token` | `POST /v1/validate_jwt_token` | Public/Service |
| `ValidateSession` | `validate_session` | `POST /v1/validate_session` | Public/Service |
| `Permissions` | `permissions` | `GET /v1/permissions` | Authenticated |

### 2.2. OIDC & OAuth2 REST Endpoints
The following endpoints remain as pure HTTP handlers to comply with strict OIDC/OAuth2 protocol requirements (redirects, form-encoding):
- `GET /.well-known/openid-configuration`
- `GET /.well-known/jwks.json`
- `GET /authorize`
- `GET /userinfo`
- `POST /oauth/token`
- `POST /oauth/revoke`
- `POST /oauth/introspect`
- `GET /oauth_login/:oauth_provider`

---

## 3. Documentation & Commenting Standards

To ensure the generated API documentation (Swagger/OpenAPI) and TypeScript clients are well-documented, the following standards MUST be followed in all `.proto` files:

### 3.1. General Principles
- Use `//` for all descriptions.
- Every RPC, Message, and Field must have a description.
- Start with a clear summary line, followed by details on constraints or behavior.

### 3.2. Field Metadata Labels
Include explicit labels in field comments to denote behavior:
- `// Required.` - For fields that must be provided.
- `// Optional.` - For fields that can be omitted.
- `// Read-only.` - For fields that are only populated by the server in responses.
- `// Output only.` - Similar to read-only, specifically for create/update requests where the field is ignored.

### 3.3. Permission Blocks
Every RPC method comment must include a standardized permission block:
```protobuf
// [Description of the RPC]
//
// Required permissions:
// - [scope_name] (or "Public" for open endpoints)
```

---

## 4. Protocol Buffer Definition Samples

### Permissions, Validations & Documentation
Using the Qdrant pattern for permissions, `protovalidate` for field rules, and the documentation standards defined above.

```protobuf
syntax = "proto3";

package authorizer.v1;

import "google/api/annotations.proto";
import "buf/validate/validate.proto";

// Custom option for granular permission checks
extend google.protobuf.MethodOptions {
  string permissions = 50001;
}

service AuthorizerService {
  // Signup registers a new user in the system.
  //
  // Required permissions:
  // - Public
  rpc Signup(SignUpRequest) returns (AuthResponse) {
    option (google.api.http) = {
      post: "/api/v1/signup"
      body: "*"
    };
    option (authorizer.v1.permissions) = "";
  }

  // GetProfile returns the profile of the currently authenticated user.
  //
  // Required permissions:
  // - read:profile
  rpc GetProfile(GetProfileRequest) returns (User) {
    option (google.api.http) = {
      get: "/api/v1/profile"
    };
    option (authorizer.v1.permissions) = "read:profile";
  }
}

// SignUpRequest defines the parameters for user registration.
// It supports both email-based and phone-based signup.
message SignUpRequest {
  // The unique email address for the user.
  // Optional. If provided, must be a valid email format.
  optional string email = 1 [(buf.validate.field).string.email = true];
  
  // The password for the account. Must be at least 8 characters.
  // Required.
  string password = 2 [(buf.validate.field).string.min_len = 8];
  
  // Confirmation of the password. Must match the 'password' field.
  // Required.
  string confirm_password = 3 [(buf.validate.field).string.min_len = 8];
  
  // The user's phone number in E.164 format.
  // Optional. Example: +1234567890.
  optional string phone_number = 4 [(buf.validate.field).string.pattern = "^\\+[1-9]\\d{1,14}$"];

  // The first name of the user.
  // Optional.
  optional string given_name = 5;

  // The last name of the user.
  // Optional.
  optional string family_name = 6;

  // List of roles to be assigned to the user.
  // Optional. Defaults to project's default roles if empty.
  repeated string roles = 7;

  // Arbitrary JSON data associated with the user.
  // Optional.
  map<string, string> app_data = 8;
}
```

---

## 5. Migration Strategy: Interface Pattern

To avoid duplicating business logic between GraphQL and gRPC, all logic is moved to a unified `service.Provider` interface.

### 1. Define the Interface
```go
// internal/service/provider.go
package service

type Provider interface {
    // Authentication & Profile
    Signup(ctx context.Context, params *SignUpRequest) (*AuthResponse, error)
    Login(ctx context.Context, params *LoginRequest) (*AuthResponse, error)
    MagicLinkLogin(ctx context.Context, params *MagicLinkLoginRequest) (*Response, error)
    Logout(ctx context.Context) (*Response, error)
    UpdateProfile(ctx context.Context, params *UpdateProfileRequest) (*Response, error)
    VerifyEmail(ctx context.Context, params *VerifyEmailRequest) (*AuthResponse, error)
    ResendVerifyEmail(ctx context.Context, params *ResendVerifyEmailRequest) (*Response, error)
    ForgotPassword(ctx context.Context, params *ForgotPasswordRequest) (*ForgotPasswordResponse, error)
    ResetPassword(ctx context.Context, params *ResetPasswordRequest) (*Response, error)
    Revoke(ctx context.Context, params *OAuthRevokeRequest) (*Response, error)
    VerifyOtp(ctx context.Context, params *VerifyOTPRequest) (*AuthResponse, error)
    ResendOtp(ctx context.Context, params *ResendOTPRequest) (*Response, error)
    DeactivateAccount(ctx context.Context) (*Response, error)
    
    // Metadata & Validation
    GetMeta(ctx context.Context) (*Meta, error)
    GetSession(ctx context.Context, params *SessionQueryRequest) (*AuthResponse, error)
    GetProfile(ctx context.Context) (*User, error)
    ValidateJwtToken(ctx context.Context, params *ValidateJWTTokenRequest) (*ValidateJWTTokenResponse, error)
    ValidateSession(ctx context.Context, params *ValidateSessionRequest) (*ValidateSessionResponse, error)
    GetPermissions(ctx context.Context) ([]*Permission, error)
}
```

### 2. Refactor Resolvers
Existing GraphQL resolvers will be thin wrappers around this service, using encapsulated mapping methods:
```go
// internal/graph/schema.resolvers.go
func (r *mutationResolver) Signup(ctx context.Context, params model.SignUpRequest) (*model.AuthResponse, error) {
    // Convert GraphQL model to Service request
    req := service.SignUpRequestFromGQL(params)
    
    // Call business logic
    res, err := r.ServiceProvider.Signup(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Convert Service response back to GraphQL model
    return res.AsGQL(), nil
}
```

### 3. Implement gRPC Handler
The gRPC server will similarly use the encapsulated mapping logic:
```go
// internal/grpc/handler.go
func (s *Server) Signup(ctx context.Context, req *pb.SignUpRequest) (*pb.AuthResponse, error) {
    // Convert gRPC proto to Service request
    serviceReq := service.SignUpRequestFromPb(req)
    
    // Call business logic
    res, err := s.ServiceProvider.Signup(ctx, serviceReq)
    if err != nil {
        return nil, err
    }
    
    // Convert Service response back to gRPC proto
    return res.AsPb(), nil
}
```

---

## 6. Detailed Mapping Table

> **Note:** The table in §2.1 is the authoritative, as-implemented mapping
> (`snake_case` paths under `/v1`). The table below is the original design
> sketch retained for historical context; where it differs (hyphenated paths,
> `/api/v1` prefix, `Get*`/`DELETE`/`PUT` shapes), §2.1 wins.

| gRPC Method | GraphQL Field | REST Gateway Path | Perms | Logic Method |
| :--- | :--- | :--- | :--- | :--- |
| `Signup` | `signup` | `POST /api/v1/signup` | - | `Signup` |
| `Login` | `login` | `POST /api/v1/login` | - | `Login` |
| `MagicLinkLogin` | `magic_link_login` | `POST /api/v1/magic-link` | - | `MagicLinkLogin` |
| `Logout` | `logout` | `POST /api/v1/logout` | `auth` | `Logout` |
| `UpdateProfile` | `update_profile` | `PUT /api/v1/profile` | `auth` | `UpdateProfile` |
| `VerifyEmail` | `verify_email` | `POST /api/v1/verify-email` | - | `VerifyEmail` |
| `ResendVerifyEmail` | `resend_verify_email` | `POST /api/v1/resend-verify` | - | `ResendVerifyEmail` |
| `ForgotPassword` | `forgot_password` | `POST /api/v1/forgot-password` | - | `ForgotPassword` |
| `ResetPassword` | `reset_password` | `POST /api/v1/reset-password` | - | `ResetPassword` |
| `Revoke` | `revoke` | `POST /api/v1/revoke` | `auth` | `Revoke` |
| `VerifyOtp` | `verify_otp` | `POST /api/v1/verify-otp` | - | `VerifyOtp` |
| `ResendOtp` | `resend_otp` | `POST /api/v1/resend-otp` | - | `ResendOtp` |
| `DeactivateAccount`| `deactivate_account` | `DELETE /api/v1/account` | `auth` | `DeactivateAccount`|
| `GetMeta` | `meta` | `GET /api/v1/meta` | - | `GetMeta` |
| `GetSession` | `session` | `POST /api/v1/session` | `auth` | `GetSession` |
| `GetProfile` | `profile` | `GET /api/v1/profile` | `auth` | `GetProfile` |
| `ValidateJwtToken` | `validate_jwt_token` | `POST /api/v1/validate-jwt` | - | `ValidateJwtToken` |
| `ValidateSession` | `validate_session` | `POST /api/v1/validate-session` | - | `ValidateSession` |
| `GetPermissions` | `permissions` | `GET /api/v1/permissions` | `auth` | `GetPermissions` |

---

## 7. Testing Strategy

### 1. Service Logic Tests
Unit tests for the `internal/service` implementation using mock storage and memory providers. These tests ensure business logic correctness regardless of the transport layer.

### 2. Integration Tests (End-to-End)
- **gRPC Integration**: Using `buf` generated client to call a test gRPC server.
- **REST Gateway Integration**: Using standard HTTP clients to call the `/api/v1/...` endpoints.
- **GraphQL Regression**: Ensuring existing GraphQL tests still pass after the refactor.

### 3. Validation Tests
Assert that `protovalidate` rules (e.g., email format) correctly reject invalid requests before they reach the business logic.

---

## 8. Development Workflow

1.  **Modify Proto**: Edit files in `proto/authorizer/v1/`.
2.  **Generate Code**:
    ```bash
    buf generate
    ```
3.  **Implement Service Logic**: Update `internal/service` if new fields or logic are added.
4.  **Update Handlers**: Wire the new proto RPC to the service method.
