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

- OS: Linux or macOS or windows
- Go: (Golang)(https://golang.org/dl/) >= v1.15

### Familiarize yourself with Authorizer

1. [Architecture of Authorizer](http://docs.authorizer.dev/)
2. [GraphQL APIs](https://docs.authorizer.dev/core/graphql-api/)

### Project Setup for Authorizer core

1. Fork the [authorizer](https://github.com/authorizerdev/authorizer) repository (**Skip this step if you have access to repo**)
2. `git clone https://github.com/authorizerdev/authorizer.git`
3. `cd authorizer`
4. `cp .env.sample .env`. Check all the supported env [here](https://docs.authorizer.dev/core/env/)
5. Build the code `make clean && make`
   > Note: if you don't have [`make`](https://www.ibm.com/docs/en/aix/7.2?topic=concepts-make-command), you can `cd` into `server` dir and build using the `go build` command
6. Run binary `./build/server`

### Testing

Make sure you test before creating PR.

If you want to test for all the databases that authorizer supports you will have to run `mongodb` & `arangodb` instances locally.

Setup mongodb & arangodb using Docker

```
docker run --name mongodb -d -p 27017:27017 mongo
docker run --name arangodb -d -p 8529:8529 -e ARANGO_ROOT_PASSWORD=root arangodb/arangodb:3.8.4
```
> Note: If you are not making any changes in db schema / db operations, you can disable those db tests [here](https://github.com/authorizerdev/authorizer/blob/main/server/__test__/resolvers_test.go#L14)

If you are adding new resolver, 
1. create new resolver test file [here](https://github.com/authorizerdev/authorizer/tree/main/server/__test__)
Naming convention filename: `resolver_name_test.go` function name: `resolverNameTest(s TestSetup, t *testing.T)`
2. Add your tests [here](https://github.com/authorizerdev/authorizer/blob/main/server/__test__/resolvers_test.go#L38)

__Command to run tests:__

```sh
make test
```

__Manual Testing:__

For manually testing using graphql playground, you can paste following queries and mutations in your playground and test it

```gql
mutation Signup {
  signup(params: {
    email: "lakhan@yopmail.com",
    password: "test",
    confirm_password: "test",
    given_name: "lakhan"
  }) {
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
  resend_verify_email(params: {
    email: "lakhan@yopmail.com"
    identifier: "basic_auth_signup"
  }) {
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
  verify_email(params: {
    token: ""
  }) {
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
  login(params: {
    email: "lakhan@yopmail.com",
    password: "test"
  }) {
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
  forgot_password(params: {
    email: "lakhan@yopmail.com"
  }) {
    message
  }
}

mutation ResetPassword {
  reset_password(params: {
    token: ""
    password: "test"
    confirm_password: "test"
  }) {
    message
  }
}

mutation UpdateProfile {
  update_profile(params: {
    family_name: "samani"
  }) {
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
  magic_link_login(params: {
    email: "test@yopmail.com"
  }) {
    message
  }
}

mutation Logout {
  logout {
    message
  }
}

mutation UpdateUser{
  _update_user(params: {
    id: "dafc9400-d603-4ade-997c-83fcd54bbd67",
    roles: ["user", "admin"]
  }) {
    email
    roles
  }
}

mutation DeleteUser {
  _delete_user(params: {
    email: "signup.test134523@yopmail.com"
  }) {
    message
  }
}
```



