# authorizer 

authorizer is a complete open source authentication and authorization solution for your applications.  Bring your database and have complete control over the authentication, authorization and user data. It is a microservice that can be deployed anywhere and connected any sql database.

This an [Auth0](https://auth0.com) opensource alternative.

Deploy authorizer Server with Postgres DB on Heroku and get a authorizer GraphQL endpoint in under 30 seconds 

[![Deploy to
Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/authorizerdev/authorizer-heroku)

## Features
### Flexible and easy to use
* Designed to work with any OAuth service, it supports OAuth 1.0, 1.0A and 2.0
* Built-in support for many popular sign-in services
* Supports email / passwordless authentication
* Supports stateless authentication with any backend (Active Directory, LDAP, etc)
* Supports both JSON Web Tokens and database sessions
* Easy to deploy with docker, heroku
* Phase 1: supports postgres database
* SDKs for popular languages
* Quick frontend page library for (react, vue, svelete, vanilla)

### Own your own data
* An open source solution that allows you to keep control of your data
* Supports Bring Your Own Database (BYOD) and can be used with any database
* Built-in support for Postgres

### Secure by default
* Promotes the use of passwordless sign in mechanisms
* Designed to be secure by default and encourage best practice for safeguarding user data
* Uses Cross Site Request Forgery Tokens on POST routes (sign in, sign out)
* Default cookie policy aims for the most restrictive policy appropriate for each cookie
* When JSON Web Tokens are enabled, they are signed by default (JWS) with HS512
* Use JWT encryption (JWE) by setting the option encryption: true (defaults to A256GCM)
* Auto-generates symmetric signing and encryption keys for developer convenience
* Attempts to implement the latest guidance published by Open Web Application Security Project
* Advanced options allow you to define your own routines to handle controlling what accounts are allowed to sign in, for encoding and decoding JSON Web Tokens and to set custom cookie security policies and session properties, so you can control who is able to sign in and how often sessions have to be re-validated.

# License
[MIT](https://github.com/authorizerdev/authorizer/blob/main/LICENSE)

