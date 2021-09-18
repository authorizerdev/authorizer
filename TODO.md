# Task List

# Feature roles

For the first version we will only support setting roles master list via env

- [x] Support following ENV
  - [x] `ROLES` -> comma separated list of role names
  - [x] `DEFAULT_ROLE` -> default role to assign to users
- [x] Add roles input for signup
- [ ] Add roles input for login
- [ ] Return roles to user
- [ ] Return roles in users list for super admin
- [ ] Add roles to the JWT token generation
- [ ] Validate token should also validate the role, if roles to validate again is present in request
