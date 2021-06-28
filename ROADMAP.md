# Roadmap for yauth.io

This document contains detailed information about the project scope and future roadmap

## V 0.1.0 [To be released by 20-August-2021]

- [ ] Create boilerplate for server
  - [ ] Use golang as server side language
  - [ ] Use [gorm](https://github.com/go-gorm/gorm) as ORM
  - [ ] Define base schema for user
  - [ ] Define the auth schemes and variables required for that
     - [ ] Basic Auth (Username & Password based)
     - [ ] Google Login
     - [ ] Github Login
     - [ ] Twitter Login
     - [ ] Facebook Login
     - [ ] Login with magic link (Send magic link mail)
  - [ ] Add [mailing server](https://github.com/emersion/go-smtp) to send the magic link
  - [ ] Allow configuring the master password to access the console (If not set UI console can be accessed by anyone)
  - [ ] Allow configuring mailing server
  - [ ] Allow configuring RSA/HSA Keys for oauth
  - [ ] Allow configuring the DB client
  - [ ] Allow configuring the Secret
  - [ ] Allow configuring callback urls
  - [ ] Allow configuring redis, should be optional if not used use the memory to store session
  - [ ] Use [gorilla sessions](https://github.com/gorilla/sessions) for session management
- [ ] Create REST API
   - [ ] Login
   - [ ] Logout
   - [ ] Authorize [Currently checks for valid token & if token is present in session]
   - [ ] Should authorize using cookies
   - [ ] Should authorize using Authorization header
   - [ ] Role based access [Checks for particular role in JWT]
   - [ ] Register
   - [ ] Authorize UI console
- [ ] Create a UI console to configure the above parts
   - [ ] Create next js app
   - [ ] Use [Chakra UI](https://chakra-ui.com/docs/getting-started) for quick component boostraping
   - [ ] Allow user to configure above mentioned envs
   - [ ] Allow user to add user
   - [ ] Allow user to view users
   - [ ] Allow user to define the JWT token field
- [ ] A component library for react
- [ ] Create a sdks
  - [ ] NodeJS sdk which acts as a middleware and can be used to authenticate & authorize users
  - [ ] Golang sdk which acts as a middleware and can be used to authenticate & authorize users
