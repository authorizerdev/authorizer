# Task List

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
