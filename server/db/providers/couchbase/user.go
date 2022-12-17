package couchbase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// AddUser to save user information in database
func (p *provider) AddUser(ctx context.Context, user models.User) (models.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		defaultRoles, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
		if err != nil {
			return user, err
		}
		user.Roles = defaultRoles
	}

	user.CreatedAt = time.Now().Unix()
	user.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.User).Insert(user.ID, user, &insertOpt)
	if err != nil {
		return user, err
	}
	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user models.User) (models.User, error) {
	user.UpdatedAt = time.Now().Unix()
	unsertOpt := gocb.UpsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.User).Upsert(user.ID, user, &unsertOpt)
	if err != nil {
		return user, err
	}
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user models.User) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.User).Remove(user.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination model.Pagination) (*model.Users, error) {
	users := []*model.User{}
	paginationClone := pagination
	scope := p.db.Scope("_default")
	userQuery := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, created_at, updated_at FROM auth._default.%s ORDER BY id OFFSET $1 LIMIT $2", models.Collections.User)

	queryResult, err := scope.Query(userQuery, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{paginationClone.Offset, paginationClone.Limit},
	})

	_, paginationClone.Total = GetTotalDocs(ctx, scope, models.Collections.User)

	if err != nil {
		return nil, err
	}

	for queryResult.Next() {
		var user models.User
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
		Pagination: &paginationClone,
		Users:      users,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	user := models.User{}
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, created_at, updated_at FROM auth._default.%s WHERE email = $1 LIMIT 1", models.Collections.User)
	q, err := p.db.Scope("_default").Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{email},
	})

	if err != nil {
		return user, err
	}
	err = q.One(&user)
	if err != nil {
		return user, err
	}

	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (models.User, error) {
	user := models.User{}
	query := fmt.Sprintf("SELECT _id, email, email_verified_at, `password`, signup_methods, given_name, family_name, middle_name, nickname, birthdate, phone_number, phone_number_verified_at, picture, roles, revoked_timestamp, is_multi_factor_auth_enabled, created_at, updated_at FROM auth._default.%s WHERE _id = $1 LIMIT 1", models.Collections.User)
	q, err := p.db.Scope("_default").Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		Context:              ctx,
		PositionalParameters: []interface{}{id},
	})
	if err != nil {
		return user, err
	}
	err = q.One(&user)
	if err != nil {
		return user, err
	}

	return user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()

	updateFields, params := GetSetFields(data)

	if ids != nil && len(ids) > 0 {
		for _, id := range ids {
			params["id"] = id
			userQuery := fmt.Sprintf("UPDATE auth._default.%s SET %s WHERE _id = $id", models.Collections.User, updateFields)

			_, err := p.db.Scope("_default").Query(userQuery, &gocb.QueryOptions{
				ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
				Context:         ctx,
				NamedParameters: params,
			})
			if err != nil {
				return err
			}
		}
	} else {
		userQuery := fmt.Sprintf("UPDATE auth._default.%s SET %s WHERE _id IS NOT NULL", models.Collections.User, updateFields)
		_, err := p.db.Scope("_default").Query(userQuery, &gocb.QueryOptions{
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
