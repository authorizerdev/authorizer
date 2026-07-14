package cassandradb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
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

	// Column names are sourced from the `cql` struct tag (not json.Marshal, which
	// drops json:"-" fields — see buildCQLColumnMap). Secret (raw TOTP seed) and
	// RecoveryCodes are exactly the sensitive fields a maintainer might later tag
	// json:"-" for API safety; the cql tag keeps them persisted regardless.
	authenticatorsMap := buildCQLColumnMap(authenticators)

	fields := "("
	placeholders := "("
	var insertValues []interface{}
	for key, value := range authenticatorsMap {
		if value != nil {
			fields += key + ","
			placeholders += "?,"
			insertValues = append(insertValues, value)
		}
	}

	fields = fields[:len(fields)-1] + ")"
	placeholders = placeholders[:len(placeholders)-1] + ")"

	// IF NOT EXISTS only guards the partition key (id) — a freshly generated UUID that
	// never collides — so it is NOT a uniqueness guard on user_id+method. That is
	// enforced by the GetAuthenticatorDetailsByUserId check-then-insert above, which
	// carries the same inherent TOCTOU race as any non-partition-key guard in Cassandra.
	query := fmt.Sprintf("INSERT INTO %s %s VALUES %s IF NOT EXISTS", KeySpace+"."+schemas.Collections.Authenticators, fields, placeholders)
	err := p.db.Query(query, insertValues...).Exec()
	if err != nil {
		return nil, err
	}

	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	authenticators.UpdatedAt = time.Now().Unix()

	// Column names are sourced from the `cql` struct tag (not json.Marshal, which
	// drops json:"-" fields such as a future secret — see buildCQLColumnMap).
	authenticatorsMap := buildCQLColumnMap(authenticators)

	updateFields := ""
	var updateValues []interface{}
	for key, value := range authenticatorsMap {
		if key == "id" {
			continue
		}

		if key == "_key" {
			continue
		}

		if value == nil {
			updateFields += fmt.Sprintf("%s = null, ", key)
			continue
		}

		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	updateValues = append(updateValues, authenticators.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Authenticators, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}

	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	var authenticators schemas.Authenticator
	query := fmt.Sprintf("SELECT id, user_id, method, secret, recovery_codes, verified_at, created_at, updated_at FROM %s WHERE user_id = ? AND method = ? LIMIT 1 ALLOW FILTERING", KeySpace+"."+schemas.Collections.Authenticators)
	err := p.db.Query(query, userId, authenticatorType).Consistency(gocql.One).Scan(&authenticators.ID, &authenticators.UserID, &authenticators.Method, &authenticators.Secret, &authenticators.RecoveryCodes, &authenticators.VerifiedAt, &authenticators.CreatedAt, &authenticators.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &authenticators, nil
}

// DeleteAuthenticatorsByUserID removes every authenticator row for a user.
// user_id is not the partition key (id is), so DELETE cannot filter on it
// directly — mirrors DeleteUser's session cleanup: look up matching ids via
// ALLOW FILTERING, then delete each by its partition key. Used by admin MFA
// reset.
func (p *provider) DeleteAuthenticatorsByUserID(ctx context.Context, userID string) error {
	getIDsQuery := fmt.Sprintf("SELECT id FROM %s WHERE user_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.Authenticators)
	scanner := p.db.Query(getIDsQuery, userID).Iter().Scanner()
	var ids []string
	for scanner.Next() {
		var id string
		if err := scanner.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Authenticators)
		if err := p.db.Query(deleteQuery, id).Exec(); err != nil {
			return err
		}
	}
	return nil
}
