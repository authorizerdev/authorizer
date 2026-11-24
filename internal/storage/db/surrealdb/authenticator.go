package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddAuthenticator saves an authenticator. If (user_id, method) already has
// one — including when a concurrent enrollment got there first — that row is
// updated in place, as the SQL provider's upsert does, so a user never ends up
// with two authenticators of one method.
func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.Key = authenticators.ID
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	err := p.create(ctx, schemas.Collections.Authenticators, authenticators.ID, authenticators)
	if isDuplicate(err) {
		err = p.exec(ctx, "UPDATE type::table($tb) MERGE $doc WHERE user_id = $u AND method = $m RETURN NONE;",
			map[string]any{"tb": schemas.Collections.Authenticators, "doc": doc(authenticators), "u": authenticators.UserID, "m": authenticators.Method})
	}
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

// UpdateAuthenticator updates an authenticator record. Callers MUST load the
// existing record first: every field is written.
func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	if authenticators.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateAuthenticator: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	authenticators.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.Authenticators, authenticators.ID, authenticators); err != nil {
		return nil, err
	}
	return authenticators, nil
}

// UpdateAuthenticatorSecretAndVerifiedAt writes only secret and verified_at,
// leaving recovery_codes to ConsumeAuthenticatorRecoveryCode. Updating a
// missing record matches nothing, so that is a no-op.
func (p *provider) UpdateAuthenticatorSecretAndVerifiedAt(ctx context.Context, id, secret string, verifiedAt int64) error {
	return p.exec(ctx, "UPDATE type::record($tb, $id) SET secret = $s, verified_at = $v, updated_at = $t RETURN NONE;",
		map[string]any{"tb": schemas.Collections.Authenticators, "id": id, "s": secret, "v": verifiedAt, "t": time.Now().Unix()})
}

// ConsumeAuthenticatorRecoveryCode swaps the recovery-code blob only while the
// record still holds oldCodes. The expected blob is part of the statement's
// condition and a single statement is atomic, so under concurrent redemption
// exactly one caller gets a row back.
func (p *provider) ConsumeAuthenticatorRecoveryCode(ctx context.Context, id, oldCodes, newCodes string) (bool, error) {
	type updated struct {
		ID string `json:"_id"`
	}
	rows, err := queryAll[updated](ctx, p,
		"UPDATE type::record($tb, $id) SET recovery_codes = $new, updated_at = $t WHERE recovery_codes = $old RETURN record::id(id) AS _id;",
		map[string]any{"tb": schemas.Collections.Authenticators, "id": id, "old": oldCodes, "new": newCodes, "t": time.Now().Unix()})
	if err != nil {
		return false, err
	}
	return len(rows) == 1, nil
}

// GetAuthenticatorDetailsByUserId returns the user's authenticator of a method.
func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	return getWhere[schemas.Authenticator](ctx, p, schemas.Collections.Authenticators, "user_id = $u AND method = $m", "authenticator", map[string]any{"u": userId, "m": authenticatorType})
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// Used by admin MFA reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE user_id = $u RETURN NONE;",
		map[string]any{"tb": schemas.Collections.Authenticators, "u": userID})
}
