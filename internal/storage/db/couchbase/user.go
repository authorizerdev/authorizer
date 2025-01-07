package couchbase

import (
	"context"
	"fmt"
	"log"
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
	_, err := p.db.Collection(schemas.Collections.User).Insert(user.ID, user, &insertOpt)
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
	_, err := p.db.Collection(schemas.Collections.User).Upsert(user.ID, user, &upsertOpt)
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
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) (*model.Users, error) {
	users := []*model.User{}
	paginationClone := pagination
	userQuery := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s.%s ORDER BY id OFFSET $1 LIMIT $2", p.scopeName, schemas.Collections.User)
	queryResult, err := p.db.Query(userQuery, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{paginationClone.Offset, paginationClone.Limit},
	})
	if err != nil {
		return nil, err
	}
	total, err := p.GetTotalDocs(ctx, schemas.Collections.User)
	if err != nil {
		return nil, err
	}
	paginationClone.Total = total
	for queryResult.Next() {
		var user schemas.User
		err := queryResult.Row(&user)
		if err != nil {
			log.Fatal(err)
		}
		users = append(users, user.AsAPIUser())
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return &model.Users{
		Pagination: paginationClone,
		Users:      users,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	var user *schemas.User
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s.%s WHERE email = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{email},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	var user *schemas.User
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s.%s WHERE _id = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{id},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&user)
	if err != nil {
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
	var user *schemas.User
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, app_data, created_at, updated_at FROM %s.%s WHERE phone_number = $1 LIMIT 1", p.scopeName, schemas.Collections.User)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{phoneNumber},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
