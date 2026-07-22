# Contributing

We're so excited you're interested in helping with Authorizer! We are happy to help you get started, even if you don't have any previous open-source experience :blush:

## New to Open Source?

1. Take a look at [How to Contribute to an Open Source Project on GitHub](https://egghead.io/courses/how-to-contribute-to-an-open-source-project-on-github)
2. Go through the [Authorizer Code of Conduct](./CODE_OF_CONDUCT.md)

## Where to ask questions?

1. Check our [Github Issues](https://github.com/authorizerdev/authorizer/issues) to see if someone has already answered your question.
2. Join our community on [Discord](https://discord.gg/Zv2D5h6kkK) and feel free to ask us your questions

As you gain experience with Authorizer, please help answer other people's questions! :pray:

## What to work on?

You can get started by taking a look at our [Github issues](https://github.com/authorizerdev/authorizer/issues)  
If you find one that looks interesting and no one else is already working on it, comment on that issue and start contributing 🙂.

Please ask as many questions as you need, either directly in the issue or on [Discord](https://discord.gg/Zv2D5h6kkK). We're happy to help!:raised_hands:

### Contributions that are ALWAYS welcome

1. More tests
2. Improved Docs
3. Improved error messages
4. Educational content like blogs, videos, courses

## Development Setup

### Prerequisites

- OS: Linux or macOS or Windows
- [Go](https://golang.org/dl/) >= 1.24 (see `go.mod`)
- [Node.js](https://nodejs.org/) >= 18 and npm (only if building web app or dashboard)

### Familiarize yourself with Authorizer

1. [Architecture of Authorizer](http://docs.authorizer.dev/)
2. [GraphQL APIs](https://docs.authorizer.dev/core/graphql-api/)
3. [Migration Guide (v1 → v2)](../MIGRATION.md) – v2 uses CLI-based configuration

### Project Setup for Authorizer core

1. Fork the [authorizer](https://github.com/authorizerdev/authorizer) repository (**Skip this step if you have access to repo**)
2. Clone repo: `git clone https://github.com/authorizerdev/authorizer.git` or use the forked url from step 1
3. Change directory: `cd authorizer`
4. Build the server: `make build` (or `go build -o build/authorizer .`)
5. (Optional) Build the web app and dashboard: `make build-app` and `make build-dashboard`
6. Run locally: `make dev` (uses SQLite and demo secrets for development)

> **v2:** The server does **not** read from `.env`. All configuration is passed via CLI arguments. See [MIGRATION.md](../MIGRATION.md).

### Updating GraphQL schema

- Modify `internal/graph/schema.graphqls` (or other files in `internal/graph/`)
- Run `make generate-graphql` to regenerate models and resolvers
- If a new mutation or query is added, implement the resolver in `internal/graph/` (resolver layout follows schema)

### Adding support for new database

1. Run `make generate-db-template dbname=NEW_DB_NAME`
   - e.g. `make generate-db-template dbname=dynamodb`

   This copies `internal/storage/db/provider_template/` to `internal/storage/db/NEW_DB_NAME/` and renames the package. The template already stubs every method of `storage.Provider` (`internal/storage/provider.go`) across all feature areas — users, sessions, webhooks, email templates, OTP, authenticators, memory-store (session/MFA/OAuth-state), audit logs, clients, trusted issuers, SAML (SP + IDP keys), SCIM (endpoints + groups), WebAuthn credentials, organizations, org memberships, org domains, and federated identities. Run `go test ./internal/storage/db/NEW_DB_NAME/...` any time to confirm it still satisfies `storage.Provider` in full — `interface_test.go` fails to compile the instant a method goes missing.

2. Change the `provider` struct and `NewProvider` in `NEW_DB_NAME/provider.go` to hold and construct your actual database client (the template ships with a placeholder `*gorm.DB` field — replace it).

3. Implement each stubbed method for real, one feature file at a time. Use an existing provider as a reference for the query patterns of a similar backend:
   - SQL-like/GORM backend → `internal/storage/db/sql/`
   - Document store → `internal/storage/db/mongodb/` or `internal/storage/db/arangodb/`
   - Wide-column store → `internal/storage/db/cassandradb/`
   - Key-value store → `internal/storage/db/dynamodb/` or `internal/storage/db/couchbase/`

4. Wire the new provider into `storage.New()` (`internal/storage/provider.go`) behind its config-selected database type.

5. Add the new provider to the storage test matrix (`TEST_DBS`) and a `make test-NEW_DB_NAME` / `test-cleanup-NEW_DB_NAME` Docker target in the `Makefile`, following the pattern of the existing `test-postgres` / `test-mongodb` targets.

> Note: `go test ./internal/storage/db/NEW_DB_NAME/...` will fail to compile with a `does not implement storage.Provider (missing method ...)` error until every method is implemented. This check lives in a `_test` file rather than in `provider.go` itself — `internal/storage` imports every concrete provider (including yours, once step 4 is done), so a same-package assertion would create an import cycle.

### Testing

Make sure you test before creating a PR.

The main `make test` target spins up Postgres, Redis, ScyllaDB, MongoDB, ArangoDB, DynamoDB, and Couchbase via Docker, runs the Go test suite, then tears down containers.

For local development without full DB matrix:

```sh
make dev   # run server for manual testing
go test -v ./...   # run tests (requires Docker for full suite)
```

If you are adding a new resolver:

1. Create a new test file in `internal/integration_tests/` (naming: `resolver_name_test.go`)
2. Follow the existing pattern using `getTestConfig()` and `initTestSetup()`

**Command to run full test suite:**

```sh
make test
```

**Manual Testing:**

For manually testing using graphql playground, you can paste following queries and mutations in your playground and test it

```gql
mutation Signup {
  signup(
    params: {
      email: "lakhan@yopmail.com"
      password: "test"
      confirm_password: "test"
      given_name: "lakhan"
    }
  ) {
    message
    user {
      id
      family_name
      given_name
      email
      email_verified
    }
  }
}

mutation ResendEamil {
  resend_verify_email(
    params: { email: "lakhan@yopmail.com", identifier: "basic_auth_signup" }
  ) {
    message
  }
}

query GetVerifyRequests {
  _verification_requests {
    id
    token
    expires
    identifier
  }
}

mutation VerifyEmail {
  verify_email(params: { token: "" }) {
    access_token
    expires_at
    user {
      id
      email
      given_name
      email_verified
    }
  }
}

mutation Login {
  login(params: { email: "lakhan@yopmail.com", password: "test" }) {
    access_token
    expires_at
    user {
      id
      family_name
      given_name
      email
    }
  }
}

query GetSession {
  session {
    access_token
    expires_at
    user {
      id
      given_name
      family_name
      email
      email_verified
      signup_methods
      created_at
      updated_at
    }
  }
}

mutation ForgotPassword {
  forgot_password(params: { email: "lakhan@yopmail.com" }) {
    message
  }
}

mutation ResetPassword {
  reset_password(
    params: { token: "", password: "test", confirm_password: "test" }
  ) {
    message
  }
}

mutation UpdateProfile {
  update_profile(params: { family_name: "samani" }) {
    message
  }
}

query GetUsers {
  _users {
    id
    email
    email_verified
    given_name
    family_name
    picture
    signup_methods
    phone_number
  }
}

mutation MagicLinkLogin {
  magic_link_login(params: { email: "test@yopmail.com" }) {
    message
  }
}

mutation Logout {
  logout {
    message
  }
}

mutation UpdateUser {
  _update_user(
    params: {
      id: "dafc9400-d603-4ade-997c-83fcd54bbd67"
      roles: ["user", "admin"]
    }
  ) {
    email
    roles
  }
}

mutation DeleteUser {
  _delete_user(params: { email: "signup.test134523@yopmail.com" }) {
    message
  }
}
```
