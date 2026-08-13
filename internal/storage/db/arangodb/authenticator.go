package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
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

	authenticators.Key = authenticators.ID
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()

	authenticatorsCollection, _ := p.db.Collection(ctx, schemas.Collections.Authenticators)
	meta, err := authenticatorsCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), authenticators)
	if err != nil {
		return nil, err
	}
	authenticators.Key = meta.Key
	authenticators.ID = meta.ID.String()

	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()

	collection, _ := p.db.Collection(ctx, schemas.Collections.Authenticators)
	meta, err := collection.UpdateDocument(ctx, authenticators.Key, authenticators)
	if err != nil {
		return nil, err
	}

	authenticators.Key = meta.Key
	authenticators.ID = meta.ID.String()
	return authenticators, nil
}

// ConsumeAuthenticatorRecoveryCode swaps the recovery-code blob only while the
// document still holds oldCodes. FILTER and UPDATE are one AQL statement over a
// single document, which ArangoDB applies atomically, so RETURN NEW yields a
// document for the caller that wrote and nothing for the rest. The document is
// matched on `_id` because that is the identifier the struct carries back out
// of GetAuthenticatorDetailsByUserId — ArangoDB overwrites the field with its
// own "collection/key" value, so the ID a caller holds is never the bare UUID.
func (p *provider) ConsumeAuthenticatorRecoveryCode(ctx context.Context, id, oldCodes, newCodes string) (bool, error) {
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id AND d.recovery_codes == @old_codes UPDATE d WITH {recovery_codes: @new_codes, updated_at: @updated_at} IN %s RETURN NEW", schemas.Collections.Authenticators, schemas.Collections.Authenticators)
	bindVars := map[string]interface{}{
		"id":         id,
		"old_codes":  oldCodes,
		"new_codes":  newCodes,
		"updated_at": time.Now().Unix(),
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		// A concurrent writer that got there first surfaces as a write-write
		// conflict, which is a lost race and not a fault: the caller re-reads
		// and finds the code already consumed.
		if arangoDriver.IsConflict(err) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = cursor.Close() }()
	return cursor.HasMore(), nil
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// Used by admin MFA reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.user_id == @user_id REMOVE d IN %s", schemas.Collections.Authenticators, schemas.Collections.Authenticators)
	bindVars := map[string]interface{}{
		"user_id": userID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators *schemas.Authenticator
	query := fmt.Sprintf("FOR d in %s FILTER d.user_id == @user_id AND d.method == @method LIMIT 1 RETURN d", schemas.Collections.Authenticators)
	bindVars := map[string]interface{}{
		"user_id": userId,
		"method":  authenticatorType,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if authenticators == nil {
				return authenticators, fmt.Errorf("authenticator not found: %w", ErrNotFound)
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &authenticators)
		if err != nil {
			return nil, err
		}
	}
	return authenticators, nil
}
