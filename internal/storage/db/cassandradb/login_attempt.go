package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddLoginAttempt records a login attempt in the database
func (p *provider) AddLoginAttempt(ctx context.Context, loginAttempt *schemas.LoginAttempt) error {
	if loginAttempt.ID == "" {
		loginAttempt.ID = uuid.New().String()
	}
	loginAttempt.Key = loginAttempt.ID
	loginAttempt.CreatedAt = time.Now().Unix()
	if loginAttempt.AttemptedAt == 0 {
		loginAttempt.AttemptedAt = loginAttempt.CreatedAt
	}
	insertQuery := fmt.Sprintf(
		"INSERT INTO %s (id, email, ip_address, successful, attempted_at, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.LoginAttempt,
	)
	return p.db.Query(insertQuery,
		loginAttempt.ID,
		loginAttempt.Email,
		loginAttempt.IPAddress,
		loginAttempt.Successful,
		loginAttempt.AttemptedAt,
		loginAttempt.CreatedAt,
	).Exec()
}

// CountFailedLoginAttemptsSince counts failed login attempts for an email since the given Unix timestamp
func (p *provider) CountFailedLoginAttemptsSince(ctx context.Context, email string, since int64) (int64, error) {
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM %s WHERE email = ? AND successful = false AND attempted_at >= ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.LoginAttempt,
	)
	var count int64
	err := p.db.Query(countQuery, email, since).Consistency(gocql.One).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteLoginAttemptsBefore removes all login attempts older than the given Unix timestamp
// Cassandra does not support DELETE with a range filter directly, so we first select the IDs and delete one by one.
func (p *provider) DeleteLoginAttemptsBefore(ctx context.Context, before int64) error {
	selectQuery := fmt.Sprintf(
		`SELECT id FROM %s WHERE attempted_at < ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.LoginAttempt,
	)
	scanner := p.db.Query(selectQuery, before).Iter().Scanner()
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
	deleteQuery := fmt.Sprintf(
		`DELETE FROM %s WHERE id = ?`,
		KeySpace+"."+schemas.Collections.LoginAttempt,
	)
	for _, id := range ids {
		if err := p.db.Query(deleteQuery, id).Exec(); err != nil {
			return err
		}
	}
	return nil
}
