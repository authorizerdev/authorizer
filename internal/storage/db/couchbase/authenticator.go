package couchbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	// ponytail: check-then-insert has no atomic uniqueness guard on (user_id, method).
	// Collection.Insert only enforces uniqueness on the document key (authenticator ID), so
	// two concurrent calls can both pass this pre-check and insert duplicate authenticators.
	// Accepted for now (shared by the other NoSQL providers); a real fix needs a Couchbase
	// unique secondary index on (user_id, method) plus a conditional/index-backed insert.
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(authenticators)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.Authenticators).Insert(authenticators.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(authenticators)
	if err != nil {
		return nil, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	authenticator := map[string]interface{}{}
	err = decoder.Decode(&authenticator)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(authenticator)
	params["_id"] = authenticators.ID
	query := fmt.Sprintf("UPDATE %s.%s SET %s WHERE _id = $_id", p.scopeName, schemas.Collections.Authenticators, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

// ConsumeAuthenticatorRecoveryCode swaps the recovery-code blob only while the
// document still holds oldCodes.
//
// This goes through the KV API rather than N1QL on purpose. A N1QL `UPDATE ...
// WHERE recovery_codes = $old` gives no dependable way to tell "matched nothing"
// from "the statement was retried", whereas Get→Replace carries the document's
// CAS: the Replace is rejected outright if anything mutated the document between
// the two calls, which is precisely the race being closed. Both a stale blob and
// a CAS mismatch mean the same thing — another caller got there first — and both
// return (false, nil).
func (p *provider) ConsumeAuthenticatorRecoveryCode(ctx context.Context, id, oldCodes, newCodes string) (bool, error) {
	collection := p.db.Collection(schemas.Collections.Authenticators)
	res, err := collection.Get(id, &gocb.GetOptions{Context: ctx})
	if err != nil {
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			return false, nil
		}
		return false, err
	}
	var authenticator schemas.Authenticator
	if err := res.Content(&authenticator); err != nil {
		return false, err
	}
	// Decode into the schema rather than a bare map so the int64 timestamps keep
	// their type on the way back out — a map round trip turns them into float64.
	if authenticator.RecoveryCodes == nil || *authenticator.RecoveryCodes != oldCodes {
		return false, nil
	}
	authenticator.RecoveryCodes = &newCodes
	authenticator.UpdatedAt = time.Now().Unix()
	doc, err := structToDocument(&authenticator)
	if err != nil {
		return false, err
	}
	_, err = collection.Replace(id, doc, &gocb.ReplaceOptions{Context: ctx, Cas: res.Cas()})
	if err != nil {
		if errors.Is(err, gocb.ErrCasMismatch) || errors.Is(err, gocb.ErrDocumentNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators *schemas.Authenticator
	query := fmt.Sprintf("SELECT _id, user_id, method, secret, recovery_codes, verified_at, created_at, updated_at FROM %s.%s WHERE user_id = $1 AND method = $2 LIMIT 1", p.scopeName, schemas.Collections.Authenticators)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{userId, authenticatorType},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&authenticators)
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// Used by admin MFA reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	query := fmt.Sprintf("DELETE FROM %s.%s WHERE user_id = $1", p.scopeName, schemas.Collections.Authenticators)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{userID},
	})
	if err != nil {
		return err
	}
	return nil
}
