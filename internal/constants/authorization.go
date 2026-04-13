package constants

const (
	// PolicyTypeRole is the policy type for role-based policies.
	// A role-based policy grants or denies access based on the principal's roles.
	PolicyTypeRole = "role"
	// PolicyTypeUser is the policy type for user-based policies.
	// A user-based policy grants or denies access to specific user IDs.
	PolicyTypeUser = "user"

	// PolicyLogicPositive grants access when the policy condition matches.
	PolicyLogicPositive = "positive"
	// PolicyLogicNegative denies access when the policy condition matches (blacklist).
	PolicyLogicNegative = "negative"

	// DecisionStrategyAffirmative grants if ANY policy/target grants (OR logic).
	DecisionStrategyAffirmative = "affirmative"
	// DecisionStrategyUnanimous grants only if ALL policies/targets agree (AND logic).
	DecisionStrategyUnanimous = "unanimous"

	// AuthorizationEnforcementDisabled turns off the authorization engine entirely.
	// All check_permission calls return true. Default for backward compatibility.
	AuthorizationEnforcementDisabled = "disabled"
	// AuthorizationEnforcementPermissive returns true when no matching permission
	// exists, but logs a warning. Useful for migration/development.
	AuthorizationEnforcementPermissive = "permissive"
	// AuthorizationEnforcementEnforcing is default-deny. Returns false when no
	// matching permission exists.
	AuthorizationEnforcementEnforcing = "enforcing"

	// PrincipalTypeUser identifies a human user principal (from authorization_code grant).
	PrincipalTypeUser = "user"
	// PrincipalTypeClient identifies a service/M2M principal (from client_credentials grant).
	PrincipalTypeClient = "client"
	// PrincipalTypeAgent identifies an AI agent principal (future use).
	PrincipalTypeAgent = "agent"

	// TargetTypeRole is a policy target that matches by role name.
	TargetTypeRole = "role"
	// TargetTypeUser is a policy target that matches by user ID.
	TargetTypeUser = "user"
)
