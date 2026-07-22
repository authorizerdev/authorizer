package cassandradb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddUser to save user information in database
func (p *provider) AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		user.Roles = strings.Join(p.config.DefaultRoles, ",")
	}

	if user.PhoneNumber != nil && strings.TrimSpace(refs.StringValue(user.PhoneNumber)) != "" {
		if u, _ := p.GetUserByPhoneNumber(ctx, refs.StringValue(user.PhoneNumber)); u != nil && u.ID != user.ID {
			return user, fmt.Errorf("user with given phone number already exists")
		}
	} else if user.Email != nil && strings.TrimSpace(refs.StringValue(user.Email)) != "" {
		if u, _ := p.GetUserByEmail(ctx, refs.StringValue(user.Email)); u != nil && u.ID != user.ID {
			return user, fmt.Errorf("user with given email already exists")
		}
	}

	user.CreatedAt = time.Now().Unix()
	user.UpdatedAt = time.Now().Unix()

	// Column names are sourced from the `cql` struct tag (not json.Marshal, which
	// drops json:"-" fields such as password — see buildCQLColumnMap).
	userMap := buildCQLColumnMap(user)

	fields := "("
	placeholders := "("
	var insertValues []interface{}
	for key, value := range userMap {
		if value != nil {
			fields += key + ","
			placeholders += "?,"
			insertValues = append(insertValues, value)
		}
	}

	fields = fields[:len(fields)-1] + ")"
	placeholders = placeholders[:len(placeholders)-1] + ")"

	// IF NOT EXISTS only guards the partition key (id) — a freshly generated UUID that
	// never collides — so it does NOT enforce email/phone uniqueness. That is enforced
	// by the GetUserByEmail/GetUserByPhoneNumber check-then-insert above, which carries
	// the same inherent TOCTOU race as any non-partition-key guard in Cassandra.
	query := fmt.Sprintf("INSERT INTO %s %s VALUES %s IF NOT EXISTS", KeySpace+"."+schemas.Collections.User, fields, placeholders)
	err := p.db.Query(query, insertValues...).Exec()

	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser to update user information in database
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateUser: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	user.UpdatedAt = time.Now().Unix()

	// Column names are sourced from the `cql` struct tag (not json.Marshal, which
	// drops json:"-" fields such as password — see buildCQLColumnMap).
	userMap := buildCQLColumnMap(user)

	updateFields := ""
	var updateValues []interface{}
	for key, value := range userMap {
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

	updateValues = append(updateValues, user.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.User, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.User)
	err := p.db.Query(query, user.ID).Exec()
	if err != nil {
		return err
	}
	getSessionsQuery := fmt.Sprintf("SELECT id FROM %s WHERE user_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.Session)
	scanner := p.db.Query(getSessionsQuery, user.ID).Iter().Scanner()
	var sessionIDList []string
	for scanner.Next() {
		var wlID string
		err = scanner.Scan(&wlID)
		if err != nil {
			return err
		}
		sessionIDList = append(sessionIDList, wlID)
	}
	if len(sessionIDList) > 0 {
		placeholders := strings.Repeat("?,", len(sessionIDList))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(sessionIDList))
		for i, id := range sessionIDList {
			deleteValues[i] = id
		}
		deleteSessionQuery := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.Session, placeholders)
		err = p.db.Query(deleteSessionQuery, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination, query string) ([]*schemas.User, *model.Pagination, error) {
	responseUsers := []*schemas.User{}
	paginationClone := pagination
	const columns = "id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, mfa_locked_at, app_data, external_id, is_active, created_at, updated_at"
	scanUser := func(scanner gocql.Scanner) (*schemas.User, error) {
		var user schemas.User
		err := scanner.Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods,
			&user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber,
			&user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled,
			&user.HasSkippedMFASetupAt, &user.MFALockedAt, &user.AppData, &user.ExternalID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
		return &user, err
	}

	if search := strings.TrimSpace(query); search != "" {
		// ponytail: Cassandra/ScyllaDB has no substring index on non-key
		// columns, so search does a full-table scan and filters in application
		// code — O(n). Acceptable for an admin search surface; upgrade path is a
		// SASI/secondary index or an external search service at scale.
		q := fmt.Sprintf("SELECT %s FROM %s", columns, KeySpace+"."+schemas.Collections.User)
		scanner := p.db.Query(q).Iter().Scanner()
		matched := []*schemas.User{}
		for scanner.Next() {
			user, err := scanUser(scanner)
			if err != nil {
				return nil, nil, err
			}
			if user.MatchesSearch(search) {
				matched = append(matched, user)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, nil, err
		}
		paginationClone.Total = int64(len(matched))
		start := pagination.Offset
		if start > int64(len(matched)) {
			start = int64(len(matched))
		}
		end := start + pagination.Limit
		if end > int64(len(matched)) {
			end = int64(len(matched))
		}
		return matched[start:end], paginationClone, nil
	}

	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.User)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	q := fmt.Sprintf("SELECT %s FROM %s LIMIT %d", columns, KeySpace+"."+schemas.Collections.User,
		pagination.Limit+pagination.Offset)
	scanner := p.db.Query(q).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			user, err := scanUser(scanner)
			if err != nil {
				return nil, nil, err
			}
			responseUsers = append(responseUsers, user)
		}
		counter++
	}
	return responseUsers, paginationClone, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	var user schemas.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, mfa_locked_at, app_data, external_id, is_active, created_at, updated_at FROM %s WHERE email = ? LIMIT 1 ALLOW FILTERING", KeySpace+"."+schemas.Collections.User)
	err := p.db.Query(query, email).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.HasSkippedMFASetupAt, &user.MFALockedAt, &user.AppData, &user.ExternalID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	var user schemas.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, mfa_locked_at, app_data, external_id, is_active, created_at, updated_at FROM %s WHERE id = ? LIMIT 1", KeySpace+"."+schemas.Collections.User)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.HasSkippedMFASetupAt, &user.MFALockedAt, &user.AppData, &user.ExternalID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUsers updates the users identified by ids. An empty ids slice is
// rejected with schemas.ErrUpdateUsersEmptyIDs — global updates are disabled so
// a missing filter can never mutate every user row.
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	if len(ids) == 0 {
		return schemas.ErrUpdateUsersEmptyIDs
	}
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()
	convertMapValues(data)

	updateFields := ""
	var updateValues []interface{}
	for key, value := range data {
		if key == "_id" {
			continue
		}

		if key == "_key" {
			continue
		}

		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}

		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	for _, id := range ids {
		vals := make([]interface{}, len(updateValues))
		copy(vals, updateValues)
		vals = append(vals, id)
		query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.User, updateFields)
		err := p.db.Query(query, vals...).Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	var user schemas.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, mfa_locked_at, app_data, external_id, is_active, created_at, updated_at FROM %s WHERE phone_number = ? LIMIT 1 ALLOW FILTERING", KeySpace+"."+schemas.Collections.User)
	err := p.db.Query(query, phoneNumber).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.HasSkippedMFASetupAt, &user.MFALockedAt, &user.AppData, &user.ExternalID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByExternalID fetches an IdP-provisioned user by its org-namespaced
// external ID. The lookup key is the composite "<orgID>:<externalID>".
func (p *provider) GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error) {
	var user schemas.User
	query := fmt.Sprintf("SELECT id, email, email_verified_at, password, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, mfa_locked_at, app_data, external_id, is_active, created_at, updated_at FROM %s WHERE external_id = ? LIMIT 1 ALLOW FILTERING", KeySpace+"."+schemas.Collections.User)
	err := p.db.Query(query, orgID+":"+externalID).Consistency(gocql.One).Scan(&user.ID, &user.Email, &user.EmailVerifiedAt, &user.Password, &user.SignupMethods, &user.GivenName, &user.FamilyName, &user.MiddleName, &user.Nickname, &user.Birthdate, &user.PhoneNumber, &user.PhoneNumberVerifiedAt, &user.Picture, &user.Roles, &user.RevokedTimestamp, &user.IsMultiFactorAuthEnabled, &user.HasSkippedMFASetupAt, &user.MFALockedAt, &user.AppData, &user.ExternalID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
