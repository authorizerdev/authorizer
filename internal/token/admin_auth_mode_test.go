package token

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/constants"
)

const adminSecretHeader = "x-authorizer-admin-secret"

func TestAdminAuthMode_SharedSecretWhenHeaderMatches(t *testing.T) {
	p := newProvider("correct-secret", false)
	assert.Equal(t, constants.AuditAuthModeSharedSecret,
		p.AdminAuthMode(newGinCtx(adminSecretHeader, "correct-secret")))
}

func TestAdminAuthMode_EmptyWhenNotSuperAdmin(t *testing.T) {
	p := newProvider("correct-secret", false)
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "wrong-secret")))
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "")))
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx("", "")))
}

// IsSuperAdmin is re-expressed in terms of AdminAuthMode, so they must agree
// on every input. A divergence would mean a caller is admitted as super-admin
// while the audit record says they are not an admin at all.
func TestIsSuperAdmin_AgreesWithAdminAuthMode(t *testing.T) {
	p := newProvider("correct-secret", false)
	for _, secret := range []string{"correct-secret", "wrong-secret", ""} {
		gc := newGinCtx(adminSecretHeader, secret)
		assert.Equal(t, p.AdminAuthMode(gc) != "", p.IsSuperAdmin(gc),
			"disagreement for secret %q", secret)
	}
}

// An unconfigured secret must never authenticate, and must never be recorded
// as though it had.
func TestAdminAuthMode_EmptyAdminSecretNeverReportsSharedSecret(t *testing.T) {
	p := newProvider("", false)
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "")))
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "anything")))
}

// With header auth disabled, even the correct secret is not super admin.
func TestAdminAuthMode_RespectsDisableAdminHeaderAuth(t *testing.T) {
	p := newProvider("correct-secret", true)
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "correct-secret")))
}
