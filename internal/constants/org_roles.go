package constants

// OrgRoleAdmin is the reserved, namespaced OrgMembership role that grants a user
// org-scoped admin rights (manage their own org's SSO/SCIM/domain config and
// members) without being a platform super-admin.
//
// It is intentionally the namespaced string "authorizer:org_admin", NOT the bare
// "admin": app-defined org roles are free-form and commonly include "admin" or
// "owner" for the app's own RBAC meaning, so reserving the bare string would
// silently hand SSO/SCIM control to every member a tenant already made an
// app-level admin. The ":" namespace cannot collide with a normal app role.
// Existing bare-"admin" memberships are NEVER auto-promoted — this role must be
// granted explicitly.
const OrgRoleAdmin = "authorizer:org_admin"
