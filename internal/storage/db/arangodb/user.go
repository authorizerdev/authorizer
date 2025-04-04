package arangodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddUser to save user information in database
func (p *provider) AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
		user.Key = user.ID
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
	userCollection, _ := p.db.Collection(ctx, schemas.Collections.User)
	meta, err := userCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), user)
	if err != nil {
		return nil, err
	}
	user.Key = meta.Key
	user.ID = meta.ID.String()

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	user.UpdatedAt = time.Now().Unix()

	collection, _ := p.db.Collection(ctx, schemas.Collections.User)
	meta, err := collection.UpdateDocument(ctx, user.Key, user)
	if err != nil {
		return nil, err
	}

	user.Key = meta.Key
	user.ID = meta.ID.String()
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	collection, _ := p.db.Collection(ctx, schemas.Collections.User)
	_, err := collection.RemoveDocument(ctx, user.Key)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(`FOR d IN %s FILTER d.user_id == @user_id REMOVE { _key: d._key } IN %s`, schemas.Collections.Session, schemas.Collections.Session)
	bindVars := map[string]interface{}{
		"user_id": user.Key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) ([]*schemas.User, *model.Pagination, error) {
	var users []*schemas.User
	sctx := arangoDriver.WithQueryFullCount(ctx)

	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.User, pagination.Offset, pagination.Limit)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var user *schemas.User
		meta, err := cursor.ReadDocument(ctx, &user)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			users = append(users, user)
		}
	}
	return users, paginationClone, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	var user *schemas.User
	query := fmt.Sprintf("FOR d in %s FILTER d.email == @email RETURN d", schemas.Collections.User)
	bindVars := map[string]interface{}{
		"email": email,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	var user *schemas.User
	query := fmt.Sprintf("FOR d in %s FILTER d._id == @id LIMIT 1 RETURN d", schemas.Collections.User)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()
	userInfoBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	query := ""
	if len(ids) > 0 {
		keysArray := ""
		for _, id := range ids {
			keysArray += fmt.Sprintf("'%s', ", id)
		}
		keysArray = strings.Trim(keysArray, " ")
		keysArray = strings.TrimSuffix(keysArray, ",")
		query = fmt.Sprintf("FOR u IN %s FILTER u._id IN [%s] UPDATE u._key with %s IN %s", schemas.Collections.User, keysArray, string(userInfoBytes), schemas.Collections.User)
	} else {
		query = fmt.Sprintf("FOR u IN %s UPDATE u._key with %s IN %s", schemas.Collections.User, string(userInfoBytes), schemas.Collections.User)
	}
	_, err = p.db.Query(ctx, query, nil)
	if err != nil {
		return err
	}
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	var user *schemas.User
	query := fmt.Sprintf("FOR d in %s FILTER d.phone_number == @phone_number RETURN d", schemas.Collections.User)
	bindVars := map[string]interface{}{
		"phone_number": phoneNumber,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}
