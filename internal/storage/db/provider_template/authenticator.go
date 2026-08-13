package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}

	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()
	return authenticators, nil
}

// UpdateAuthenticatorSecretAndVerifiedAt writes ONLY the secret, verified_at
// and updated_at columns.
//
// Do not implement this by loading the row, mutating a struct, and calling the
// whole-row write: that carries a stale recovery_codes blob back to the server
// and undoes a concurrent ConsumeAuthenticatorRecoveryCode. The two methods must
// keep touching disjoint columns. If the backend's UPDATE is an upsert, add an
// existence condition — a missing row is a no-op, never an insert.
func (p *provider) UpdateAuthenticatorSecretAndVerifiedAt(ctx context.Context, id, secret string, verifiedAt int64) error {
	return nil
}

// ConsumeAuthenticatorRecoveryCode swaps the recovery-code blob only while the
// row still holds oldCodes, and reports whether THIS call performed the write.
//
// Implement it with ONE conditional statement — a WHERE/filter on the expected
// blob combined with the write, or the backend's compare-and-swap. Never read
// the row, compare in Go, and then write unconditionally: that is the bug this
// method exists to remove, and it lets a single TOTP recovery code be redeemed
// by any number of concurrent requests. A refused write is (false, nil), not an
// error.
func (p *provider) ConsumeAuthenticatorRecoveryCode(ctx context.Context, id, oldCodes, newCodes string) (bool, error) {
	return false, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators *schemas.Authenticator
	return authenticators, nil
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// Used by admin MFA reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	return nil
}
