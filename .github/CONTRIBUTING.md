# Contributing

We're so excited you're interested in helping with Authorizer! We are happy to help you get started, even if you don't have any previous open-source experience :blush:

## New to Open Source?

1. Take a look at [How to Contribute to an Open Source Project on GitHub](https://egghead.io/courses/how-to-contribute-to-an-open-source-project-on-github)
2. Go through the [Authorizer Code of Conduct](https://github.com/authorizerdev/authorizer/blob/main/.github/CODE_OF_CONDUCT.md)

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

- Run `make generate-db-template dbname=NEW_DB_NAME`
  - e.g. `make generate-db-template dbname=dynamodb`

This generates a folder in `internal/storage/db/` with the specified name. Implement the methods in that folder.

> Note: Database connection and schema changes are in `internal/storage/db/DB_NAME/provider.go`; `NewProvider` is called for the configured database type.

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
