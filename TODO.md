# Task List

## Feature Multiple sessions

- Multiple sessions for users to login use hMset from redis for this
  user_id access_token1 long_live_token1
  user_id access_token2 long_live_token2

# Feature roles

For the first version we will only support setting roles master list via env

- [x] Support following ENV
  - [x] `ROLES` -> comma separated list of role names
  - [x] `DEFAULT_ROLE` -> default role to assign to users
- [x] Add roles input for signup
- [x] Add roles to update profile mutation
- [x] Add roles input for login
- [x] Return roles to user
- [x] Return roles in users list for super admin
- [x] Add roles to the JWT token generation
- [x] Validate token should also validate the role, if roles to validate again is present in request

# Misc

- [x] Fix email template
- [x] Add support for organization name in .env
- [x] Add support for organization logo in .env
