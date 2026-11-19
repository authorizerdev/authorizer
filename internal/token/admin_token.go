package token

import (
	"fmt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/gin-gonic/gin"
)

// TODO remove if not used
// // CreateAdminAuthToken creates the admin token based on secret key
// func CreateAdminAuthToken(tokenType string, c *gin.Context) (string, error) {
// 	adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
// 	if err != nil {
// 		return "", err
// 	}
// 	return crypto.EncryptPassword(adminSecret)
// }

// GetAdminAuthToken helps in getting the admin token from the request cookie.
//
// The cookie carries an opaque server-side session handle, validated by store
// lookup. It used to carry bcrypt(AdminSecret) and be validated by comparing
// against the secret — which made it a stateless bearer credential with no
// expiry and no revocation path: a captured cookie stayed valid forever, and
// logout could only ask the browser to forget its own copy. See
// admin_session.go.
func (p *provider) GetAdminAuthToken(gc *gin.Context) (string, error) {
	sessionID, err := cookie.GetAdminCookie(gc)
	if err != nil || sessionID == "" {
		return "", fmt.Errorf("unauthorized")
	}
	if err := p.ValidateAdminSession(sessionID); err != nil {
		return "", fmt.Errorf(`unauthorized`)
	}
	return sessionID, nil
}

// AdminAuthMode reports how the caller authenticated as super admin, or "" if
// they did not. This is the single place that decision is made; IsSuperAdmin
// is a thin predicate over it.
func (p *provider) AdminAuthMode(gc *gin.Context) string {
	token, err := p.GetAdminAuthToken(gc)
	if err == nil && token != "" {
		return constants.AuditAuthModeAdminSession
	}
	if p.config.DisableAdminHeaderAuth {
		return ""
	}
	// Reject header auth if no AdminSecret is configured — an unconfigured
	// secret must never grant super-admin access.
	if p.config.AdminSecret == "" {
		return ""
	}
	secret := gc.Request.Header.Get("x-authorizer-admin-secret")
	if secret == "" {
		return ""
	}
	// Throttled: this header is an unauthenticated guess at the single
	// highest-privilege credential in the system, and the only limiter in
	// front of it used to be the shared 30rps budget ordinary traffic gets.
	if valid, _ := p.VerifyAdminSecret(utils.GetIP(gc.Request), secret); valid {
		return constants.AuditAuthModeSharedSecret
	}
	return ""
}

// IsSuperAdmin checks if user is super admin
func (p *provider) IsSuperAdmin(gc *gin.Context) bool {
	return p.AdminAuthMode(gc) != ""
}
