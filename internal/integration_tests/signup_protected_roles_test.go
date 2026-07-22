package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestSignup_RejectsProtectedRole is the regression test for the role-parity
// finding: signup must reject a client-requested protected role explicitly,
// not rely on Config.Roles/Config.ProtectedRoles staying disjoint by
// operator convention. Mirrors oauth_callback.go's explicit check.
func TestSignup_RejectsProtectedRole(t *testing.T) {
	cfg := getTestConfig()
	// Misconfiguration under test: a role listed in both Roles and
	// ProtectedRoles. OAuth already defends against this; signup must too.
	cfg.Roles = append(append([]string{}, cfg.Roles...), "admin")
	cfg.ProtectedRoles = append(append([]string{}, cfg.ProtectedRoles...), "admin")
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "signup_protected_role_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
		Roles:           []string{"admin"},
	})
	assert.Error(t, err, "signup must reject a protected role even when it is also present in Roles")
	assert.Nil(t, signupRes)
}
