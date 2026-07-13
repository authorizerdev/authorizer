package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
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

	// ponytail: check-then-insert has no atomic uniqueness guard on email/phone_number.
	// Collection.Insert only enforces uniqueness on the document key (user.ID), so two
	// concurrent AddUser calls can both pass this pre-check and insert duplicate emails.
	// Accepted for now (shared by the other NoSQL providers); a real fix needs a Couchbase
	// unique secondary index plus a conditional/index-backed insert.
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(user)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.User).Insert(user.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	user.UpdatedAt = time.Now().Unix()
	upsertOpt := gocb.UpsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(user)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.User).Upsert(user.ID, doc, &upsertOpt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.User).Remove(user.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// ListUsers to get list of users from database
// countUsers returns the total user count, honouring the optional WHERE filter
// used by ListUsers search. When whereClause is empty it delegates to the
// unfiltered GetTotalDocs; otherwise it runs a filtered COUNT with the same
// positional parameters.
func (p *provider) countUsers(ctx context.Context, whereClause string, params []interface{}) (int64, error) {
	if whereClause == "" {
		return p.GetTotalDocs(ctx, schemas.Collections.User)
	}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s.%s%s", p.scopeName, schemas.Collections.User, whereClause)
	res, err := p.db.Query(countQuery, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: params,
	})
	if err != nil {
		return 0, err
	}
	totalDocs := TotalDocs{}
	_ = res.One(&totalDocs)
	return totalDocs.Total, nil
}

func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination, query string) ([]*schemas.User, *model.Pagination, error) {
	users := []*schemas.User{}
	paginationClone := pagination

	// Build an optional case-insensitive substring filter. Positional
	// parameter indices are computed dynamically so offset/limit shift when the
	// search pattern occupies $1.
	whereClause := ""
	params := []interface{}{}
	if q := strings.TrimSpace(query); q != "" {
		whereClause = " WHERE LOWER(_id) LIKE $1 OR LOWER(email) LIKE $1 OR LOWER(given_name) LIKE $1 OR LOWER(family_name) LIKE $1 OR LOWER(nickname) LIKE $1"
		params = append(params, "%"+strings.ToLower(q)+"%")
	}

	userQuery := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, `roles`, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, app_data, is_active, external_id, created_at, updated_at FROM %s.%s%s ORDER BY id OFFSET $%d LIMIT $%d", p.scopeName, schemas.Collections.User, whereClause, len(params)+1, len(params)+2)
	queryResult, err := p.db.Query(userQuery, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: append(append([]interface{}{}, params...), paginationClone.Offset, paginationClone.Limit),
	})
	if err != nil {
		return nil, nil, err
	}
	total, err := p.countUsers(ctx, whereClause, params)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, nil, err
		}
		user := &schemas.User{}
		if err := decodeDocument(raw, user); err != nil {
			return nil, nil, err
		}
		users = append(users, user)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return users, paginationClone, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, `roles`, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, app_data, is_active, external_id, created_at, updated_at FROM %s.%s WHERE email = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{email},
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	user := &schemas.User{}
	if err := decodeDocument(raw, user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByExternalID to get user information from database using the
// org-namespaced external ID. external_id is stored as "<orgID>:<externalID>"
// so IdP identifiers never collide across organizations.
func (p *provider) GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error) {
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, `roles`, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, app_data, is_active, external_id, created_at, updated_at FROM %s.%s WHERE external_id = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{orgID + ":" + externalID},
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	user := &schemas.User{}
	if err := decodeDocument(raw, user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, `roles`, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, app_data, is_active, external_id, created_at, updated_at FROM %s.%s WHERE _id = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{id},
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	user := &schemas.User{}
	if err := decodeDocument(raw, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()
	updateFields, params := GetSetFields(data)
	if len(ids) > 0 {
		for _, id := range ids {
			params["id"] = id
			userQuery := fmt.Sprintf("UPDATE %s.%s SET %s WHERE _id = $id", p.scopeName, schemas.Collections.User, updateFields)

			_, err := p.db.Query(userQuery, &gocb.QueryOptions{
				ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
				Context:         ctx,
				NamedParameters: params,
			})
			if err != nil {
				return err
			}
		}
	} else {
		userQuery := fmt.Sprintf("UPDATE %s.%s SET %s WHERE _id IS NOT NULL", p.scopeName, schemas.Collections.User, updateFields)
		_, err := p.db.Query(userQuery, &gocb.QueryOptions{
			ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
			Context:         ctx,
			NamedParameters: params,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, `roles`, revoked_timestamp, is_multi_factor_auth_enabled, has_skipped_mfa_setup_at, app_data, is_active, external_id, created_at, updated_at FROM %s.%s WHERE phone_number = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{phoneNumber},
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	user := &schemas.User{}
	if err := decodeDocument(raw, user); err != nil {
		return nil, err
	}
	return user, nil
}
