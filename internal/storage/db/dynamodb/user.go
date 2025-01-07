package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddUser to save user information in database
func (p *provider) AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	collection := p.db.Table(schemas.Collections.User)
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
	err := collection.Put(user).RunWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	collection := p.db.Table(schemas.Collections.User)
	if user.ID != "" {
		user.UpdatedAt = time.Now().Unix()
		err := UpdateByHashKey(collection, "id", user.ID, user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	collection := p.db.Table(schemas.Collections.User)
	sessionCollection := p.db.Table(schemas.Collections.Session)
	if user.ID != "" {
		err := collection.Delete("id", user.ID).Run()
		if err != nil {
			return err
		}
		_, err = sessionCollection.Batch("id").Write().Delete(dynamo.Keys{"user_id", user.ID}).RunWithContext(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) (*model.Users, error) {
	var user *schemas.User
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	collection := p.db.Table(schemas.Collections.User)
	users := []*model.User{}
	paginationClone := pagination
	scanner := collection.Scan()
	count, err := scanner.Count()
	if err != nil {
		return nil, err
	}
	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &user) {
			if paginationClone.Offset == iteration {
				users = append(users, user.AsAPIUser())
			}
		}
		lastEval = iter.LastEvaluatedKey()
		iteration += paginationClone.Limit
	}
	err = iter.Err()
	if err != nil {
		return nil, err
	}
	paginationClone.Total = count
	return &model.Users{
		Pagination: paginationClone,
		Users:      users,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	var users []*schemas.User
	var user *schemas.User
	collection := p.db.Table(schemas.Collections.User)
	err := collection.Scan().Index("email").Filter("'email' = ?", email).AllWithContext(ctx, &users)
	if err != nil {
		return user, nil
	}
	if len(users) > 0 {
		user = users[0]
		return user, nil
	} else {
		return nil, errors.New("no record found")
	}
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	collection := p.db.Table(schemas.Collections.User)
	var user *schemas.User
	err := collection.Get("id", id).OneWithContext(ctx, &user)
	if err != nil {
		if refs.StringValue(user.Email) == "" {
			return nil, errors.New("no documets found")
		} else {
			return user, nil
		}
	}
	return user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	userCollection := p.db.Table(schemas.Collections.User)
	var allUsers []schemas.User
	var res int64 = 0
	var err error
	if len(ids) > 0 {
		for _, v := range ids {
			err = UpdateByHashKey(userCollection, "id", v, data)
		}
	} else {
		// as there is no facility to update all doc - https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/SQLtoNoSQL.UpdateData.html
		userCollection.Scan().All(&allUsers)
		for _, user := range allUsers {
			err = UpdateByHashKey(userCollection, "id", user.ID, data)
			if err == nil {
				res = res + 1
			}
		}
	}
	if err != nil {
		return err
	} else {
		p.dependencies.Log.Info().Int64("modified_count", res).Msg("users updated")
	}
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	var users []*schemas.User
	var user *schemas.User
	collection := p.db.Table(schemas.Collections.User)
	err := collection.Scan().Filter("'phone_number' = ?", phoneNumber).AllWithContext(ctx, &users)
	if err != nil {
		return nil, err
	}
	if len(users) > 0 {
		user = users[0]
		return user, nil
	} else {
		return nil, errors.New("no record found")
	}
}
