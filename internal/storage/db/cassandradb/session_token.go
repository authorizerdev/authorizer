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

// AddSessionToken adds a session token to the database
func (p *provider) AddSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	if token.CreatedAt == 0 {
		token.CreatedAt = time.Now().Unix()
	}
	if token.UpdatedAt == 0 {
		token.UpdatedAt = time.Now().Unix()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, key_name, token_value, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		KeySpace+"."+schemas.Collections.SessionToken)
	// DefaultTimestamp(false): the coordinator stamps this write, not gocql.
	//
	// DeleteSessionTokenByUserIDAndKey consumes a token with DELETE ... IF
	// EXISTS and treats the applied flag as the exactly-once arbiter for
	// refresh-token rotation. That LWT's timestamp comes from the Paxos ballot,
	// i.e. the server clock, while a client-stamped insert comes from this
	// process's clock. When the client clock runs ahead, the tombstone lands
	// BELOW the insert: the row survives, yet Paxos saw it and returned
	// applied=true — so a second claim of the same single-use token also
	// returns applied=true. Measured on scylladb/scylla with the client clock
	// 10ms ahead: 20 of 20 tokens claimed twice. Server-stamping puts the
	// insert in the LWT's own domain and the double-claim goes to 0 of 20,
	// without adding a Paxos round-trip to the login path.
	return p.db.Query(query, token.ID, token.UserID, token.KeyName, token.Token, token.ExpiresAt, token.CreatedAt, token.UpdatedAt).
		DefaultTimestamp(false).Exec()
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	query := fmt.Sprintf(`SELECT id, user_id, key_name, token_value, expires_at, created_at, updated_at FROM %s WHERE user_id = ? AND key_name = ? LIMIT 1 ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken)
	err := p.db.Query(query, userId, key).Consistency(gocql.One).Scan(&token.ID, &token.UserID, &token.KeyName, &token.Token, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.SessionToken)
	// Server-stamped, same timestamp domain as the LWT delete. See AddSessionToken.
	return p.db.Query(query, id).DefaultTimestamp(false).Exec()
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (bool, error) {
	// Cassandra doesn't support delete with non-primary key filter directly, so scan first
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id = ? AND key_name = ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken)
	iter := p.db.Query(query, userId, key).WithContext(ctx).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return false, err
	}
	// The SELECT is NOT the race arbiter — two callers can read the same id.
	// `IF EXISTS` makes the DELETE a lightweight transaction (Paxos), so the
	// cluster applies exactly one of the concurrent deletes and reports
	// applied=false to the rest. That flag decides who consumed the token.
	deleted := false
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ? IF EXISTS", KeySpace+"."+schemas.Collections.SessionToken)
		// MapScanCAS, not ScanCAS: an LWT result set carries the row's columns
		// alongside [applied], and ScanCAS with no destinations fails on that
		// ("not enough columns to scan into: have 1 want 8" on ScyllaDB). The
		// map absorbs whatever columns come back. Matches the existing LWT usage
		// in org_domain.go.
		applied, err := p.db.Query(delQuery, id).WithContext(ctx).MapScanCAS(map[string]interface{}{})
		if err != nil {
			return false, err
		}
		if applied {
			deleted = true
		}
	}
	return deleted, nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID.
// The userId parameter can be just the user ID (e.g. "123") or the full stored
// format (e.g. "auth_provider:123"), matching the SQL provider.
//
// The match is **anchored**: the exact value, or a ":<userId>" suffix. It used
// to be an unanchored "%<userId>%" contains-match, which let a delete for "42"
// take out tokens belonging to "142" or "provider:427" — the same defect the
// SQL provider already fixed, left live here because no test covered it on this
// backend.
//
// Matching happens in Go rather than with CQL LIKE: Apache Cassandra only
// accepts LIKE on a SASI index and rejects it against the plain secondary index
// this provider creates, so these two methods failed outright there. The query
// already carried ALLOW FILTERING, i.e. a full table scan, so the scan cost is
// unchanged — but the transfer cost is not: on ScyllaDB, where LIKE did work,
// only matching rows crossed the wire and now every (id, user_id) pair does.
// There is no cheaper option without a SASI index, and no CQL suffix operator,
// so this is the price of a correct anchored match on this backend.
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	ids, err := p.sessionTokenIDsMatching(ctx, func(storedUserID string) bool {
		return storedUserID == userId || strings.HasSuffix(storedUserID, ":"+userId)
	})
	if err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.SessionToken)
		if err := p.db.Query(delQuery, id).DefaultTimestamp(false).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
// (e.g. "auth_provider"). A legitimate prefix match: stored user_id is
// "<namespace>:<userID>". See DeleteAllSessionTokensByUserID for why the match
// is done in Go instead of with CQL LIKE.
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	ids, err := p.sessionTokenIDsMatching(ctx, func(storedUserID string) bool {
		return strings.HasPrefix(storedUserID, namespace+":")
	})
	if err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.SessionToken)
		if err := p.db.Query(delQuery, id).DefaultTimestamp(false).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE expires_at < ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken)
	iter := p.db.Query(query, currentTime).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.SessionToken)
		if err := p.db.Query(delQuery, id).DefaultTimestamp(false).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	query := fmt.Sprintf(`SELECT id, user_id, key_name, token_value, expires_at, created_at, updated_at FROM %s`,
		KeySpace+"."+schemas.Collections.SessionToken)
	iter := p.db.Query(query).Iter()
	for {
		var token schemas.SessionToken
		if !iter.Scan(&token.ID, &token.UserID, &token.KeyName, &token.Token, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt) {
			break
		}
		tokens = append(tokens, &token)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return tokens, nil
}

// AddMFASession adds an MFA session to the database
func (p *provider) AddMFASession(ctx context.Context, session *schemas.MFASession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.CreatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = time.Now().Unix()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, key_name, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		KeySpace+"."+schemas.Collections.MFASession)
	return p.db.Query(query, session.ID, session.UserID, session.KeyName, session.ExpiresAt, session.CreatedAt, session.UpdatedAt).Exec()
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	var session schemas.MFASession
	query := fmt.Sprintf(`SELECT id, user_id, key_name, expires_at, created_at, updated_at FROM %s WHERE user_id = ? AND key_name = ? LIMIT 1 ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession)
	err := p.db.Query(query, userId, key).Consistency(gocql.One).Scan(&session.ID, &session.UserID, &session.KeyName, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.MFASession)
	return p.db.Query(query, id).Exec()
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id = ? AND key_name = ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession)
	iter := p.db.Query(query, userId, key).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.MFASession)
		if err := p.db.Query(delQuery, id).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf(`SELECT id, user_id, key_name, expires_at, created_at, updated_at FROM %s WHERE user_id = ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession)
	iter := p.db.Query(query, userId).Iter()
	for {
		var session schemas.MFASession
		if !iter.Scan(&session.ID, &session.UserID, &session.KeyName, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt) {
			break
		}
		sessions = append(sessions, &session)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE expires_at < ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession)
	iter := p.db.Query(query, currentTime).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.MFASession)
		if err := p.db.Query(delQuery, id).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf(`SELECT id, user_id, key_name, expires_at, created_at, updated_at FROM %s`,
		KeySpace+"."+schemas.Collections.MFASession)
	iter := p.db.Query(query).Iter()
	for {
		var session schemas.MFASession
		if !iter.Scan(&session.ID, &session.UserID, &session.KeyName, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt) {
			break
		}
		sessions = append(sessions, &session)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// AddOAuthState adds an OAuth state to the database (upsert by state_key)
func (p *provider) AddOAuthState(ctx context.Context, state *schemas.OAuthState) error {
	if state.ID == "" {
		state.ID = uuid.New().String()
	}
	if state.CreatedAt == 0 {
		state.CreatedAt = time.Now().Unix()
	}
	if state.UpdatedAt == 0 {
		state.UpdatedAt = time.Now().Unix()
	}
	// Delete existing state with same state_key first
	var existingIDs []string
	selectQuery := fmt.Sprintf(`SELECT id FROM %s WHERE state_key = ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.OAuthState)
	iter := p.db.Query(selectQuery, state.StateKey).Iter()
	var id string
	for iter.Scan(&id) {
		existingIDs = append(existingIDs, id)
	}
	_ = iter.Close()
	for _, eid := range existingIDs {
		_ = p.db.Query(fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.OAuthState), eid).
			DefaultTimestamp(false).Exec()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, state_key, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		KeySpace+"."+schemas.Collections.OAuthState)
	// Server-stamped: the single-use consumption of this row is an LWT delete.
	// See AddSessionToken for the mechanism.
	//
	// Safe only because state.ID is freshly generated per call (memory_store
	// SetState), so the DELETE above and this INSERT touch different partitions
	// — id is the partition key — and cannot order against each other. If a
	// caller ever supplies a stable state.ID (derived from state_key, say),
	// they become a delete-then-insert on ONE partition, two server timestamps
	// from possibly different coordinators, and the insert can be dropped
	// silently. Keep the id fresh, or make that pair an LWT.
	return p.db.Query(query, state.ID, state.StateKey, state.State, state.CreatedAt, state.UpdatedAt).
		DefaultTimestamp(false).Exec()
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var state schemas.OAuthState
	query := fmt.Sprintf(`SELECT id, state_key, state, created_at, updated_at FROM %s WHERE state_key = ? LIMIT 1 ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.OAuthState)
	err := p.db.Query(query, key).Consistency(gocql.One).Scan(&state.ID, &state.StateKey, &state.State, &state.CreatedAt, &state.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) (bool, error) {
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE state_key = ? ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.OAuthState)
	iter := p.db.Query(query, key).WithContext(ctx).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return false, err
	}
	// The SELECT above is NOT the race arbiter — two callers can both read the
	// same id. `IF EXISTS` makes the DELETE a lightweight transaction (Paxos), so
	// the cluster applies exactly one of the concurrent deletes and reports
	// applied=false to every other caller. That `applied` flag is what decides
	// who legitimately consumed the single-use code.
	deleted := false
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ? IF EXISTS", KeySpace+"."+schemas.Collections.OAuthState)
		// MapScanCAS, not ScanCAS: an LWT result set carries the row's columns
		// alongside [applied], and ScanCAS with no destinations fails on that
		// ("not enough columns to scan into: have 1 want 8" on ScyllaDB). The
		// map absorbs whatever columns come back. Matches the existing LWT usage
		// in org_domain.go.
		applied, err := p.db.Query(delQuery, id).WithContext(ctx).MapScanCAS(map[string]interface{}{})
		if err != nil {
			return false, err
		}
		if applied {
			deleted = true
		}
	}
	return deleted, nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []*schemas.OAuthState
	query := fmt.Sprintf(`SELECT id, state_key, state, created_at, updated_at FROM %s`,
		KeySpace+"."+schemas.Collections.OAuthState)
	iter := p.db.Query(query).Iter()
	for {
		var state schemas.OAuthState
		if !iter.Scan(&state.ID, &state.StateKey, &state.State, &state.CreatedAt, &state.UpdatedAt) {
			break
		}
		states = append(states, &state)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return states, nil
}

// sessionTokenIDsMatching scans the session-token table and returns the ids
// whose stored user_id satisfies match. One full scan, same as the ALLOW
// FILTERING query it replaces.
func (p *provider) sessionTokenIDsMatching(ctx context.Context, match func(storedUserID string) bool) ([]string, error) {
	query := fmt.Sprintf(`SELECT id, user_id FROM %s`, KeySpace+"."+schemas.Collections.SessionToken)
	iter := p.db.Query(query).WithContext(ctx).Iter()
	var ids []string
	var id, storedUserID string
	for iter.Scan(&id, &storedUserID) {
		if match(storedUserID) {
			ids = append(ids, id)
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return ids, nil
}
