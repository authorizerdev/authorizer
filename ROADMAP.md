# Roadmap for authorizer 

This document contains detailed information about the project scope and future roadmap

## V 0.1.0 [To be released by 20-August-2021]

- [x] Create server
  - [x] Use golang as server side language
  - [x] Use [gorm](https://github.com/go-gorm/gorm) as ORM
  - [x] Configure https://github.com/99designs/gqlgen for generating graphql schemas
  - [x] Configure https://github.com/gin-gonic/gin for creating http server
  - [x] Define base schema for user
  - [x] Define the auth schemes and variables required for that
     - [x] Basic Auth (Username & Password based)
     - [x] Google Login
     - [x] Github Login
     - [ ] Twitter Login
     - [ ] Facebook Login
     - [ ] Login with magic link (Send magic link mail)
  - [x] Add [mailing server](https://github.com/emersion/go-smtp) to send the magic link
  - [x] Allow configuring the master password to access the console (If not set UI console can be accessed by anyone)
  - [x] Allow configuring mailing server
  - [x] Allow configuring HSA Keys for oauth
  - [ ] Allow configuring RSA keys for oauth
  - [x] Allow configuring the DB client
  - [x] Allow configuring the Secret
  - [x] Allow configuring callback urls
  - [x] Allow configuring redis, should be optional if not used use the memory to store session
- [x] Create Graphql mutations and query for following
   - [x] Login mutation
   - [x] Logout muttion
   - [x] Token query :- Authorize [Currently checks for valid token & if token is present in session]
   - [x] Should authorize using cookies
   - [x] Should authorize using Authorization header
   - [ ] Role based access [Checks for particular role in JWT]
   - [x] Signup
   - [x] Forgot password
   - [x] Update profile
- [ ] Create a UI console to configure the above parts [For now using graphql playground]
   - [ ] Create react app
   - [ ] Allow user to configure above mentioned envs
   - [ ] Allow user to add user
   - [ ] Allow user to view users
   - [ ] Allow user to define the JWT token field
- [x] A component library for react
    - [x] Create AuthorizerProvider -> gives token, user, loading, setters
    - [x] Create Authorizer component -> Complete Login/Signup & Forgot password solution
    - [x] Create AuthorizerResetPassword component -> Component that can be used to verify forgot password token and reset the password
- [ ] Create a sdks
  - [ ] NodeJS sdk which acts as a middleware and can be used to authenticate & authorize users
  - [ ] Golang sdk which acts as a middleware and can be used to authenticate & authorize users
- [x] Create docker image
- [x] Create docker-compose file to quickly start this
- [x] Create heroku button
- [ ] Create a website
