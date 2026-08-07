package token

import (
	"fmt"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

const (
	// adminSessionCachePrefix namespaces server-side admin sessions in the
	// memory store.
	adminSessionCachePrefix = "admin_session:"
	// AdminSessionTTLSeconds is the absolute lifetime of an admin session. The
	// cookie's own Max-Age is a browser-side hint only — a captured cookie
	// ignores it entirely — so this is the value that actually bounds exposure.
	// Deliberately short: the admin session is the highest-privilege credential
	// in the system.
	AdminSessionTTLSeconds = int64(8 * 3600)
	// adminSessionIDBytes is the entropy of the opaque session handle. 256 bits
	// of crypto/rand means the handle needs no constant-time comparison — it is
	// not guessable and is never derived from a secret.
	adminSessionIDBytes = 32
)

// NewAdminSession mints a server-side admin session and returns the opaque
// handle to put in the cookie.
//
// The cookie used to carry bcrypt(AdminSecret), which made it a re-derivable
// stateless bearer credential: no `exp` inside it, no server-side record, and
// therefore no way to expire or revoke one. Any capture — a proxy log, an XSS
// exfil on the dashboard, a shared machine — granted admin access indefinitely
// until the operator rotated AdminSecret, which in turn killed every admin
// session at once. Logout could only ask the browser to drop its copy; it could
// not invalidate a copy someone else already held.
//
// A random handle backed by a store entry fixes all three: it expires (absolute
// TTL), it revokes (delete the entry), and revoking one session leaves the
// others alone.
func (p *provider) NewAdminSession() (string, error) {
	sessionID, err := crypto.NewRandomString(adminSessionIDBytes)
	if err != nil {
		return "", err
	}
	if err := p.dependencies.MemoryStoreProvider.SetCache(adminSessionCachePrefix+sessionID, "1", AdminSessionTTLSeconds); err != nil {
		return "", err
	}
	return sessionID, nil
}

// ValidateAdminSession reports whether an opaque admin session handle is still
// live. A handle that was never issued, has expired, or has been revoked by
// logout all fail identically.
func (p *provider) ValidateAdminSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("unauthorized")
	}
	val, err := p.dependencies.MemoryStoreProvider.GetCache(adminSessionCachePrefix + sessionID)
	if err != nil || val == "" {
		return fmt.Errorf("unauthorized")
	}
	return nil
}

// RefreshAdminSession extends a live session's absolute TTL. Used by the
// AdminSession refresh operation, which already required a valid session to
// reach.
func (p *provider) RefreshAdminSession(sessionID string) error {
	if err := p.ValidateAdminSession(sessionID); err != nil {
		return err
	}
	return p.dependencies.MemoryStoreProvider.SetCache(adminSessionCachePrefix+sessionID, "1", AdminSessionTTLSeconds)
}

// RevokeAdminSession deletes the session server-side, so a cookie copy an
// attacker already holds stops working. This is the part a stateless cookie
// could never do.
func (p *provider) RevokeAdminSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return p.dependencies.MemoryStoreProvider.DeleteCacheByPrefix(adminSessionCachePrefix + sessionID)
}
