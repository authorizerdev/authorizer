package arangodb

import (
	"context"
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
	doc, err := structToDocument(user)
	if err != nil {
		return nil, err
	}
	meta, err := userCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), doc)
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
	doc, err := structToDocument(user)
	if err != nil {
		return nil, err
	}
	meta, err := collection.UpdateDocument(ctx, user.Key, doc)
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
		// Session.UserID is stored as the full document handle (collection/key),
		// which is what user.ID holds after Add/Get. Binding user.Key (bare key)
		// would match zero session rows. This cascade is the only session-cleanup
		// path on user deletion.
		"user_id": user.ID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination, query string) ([]*schemas.User, *model.Pagination, error) {
	var users []*schemas.User
	sctx := arangoDriver.WithQueryFullCount(ctx)

	bindVars := map[string]interface{}{
		"offset": pagination.Offset,
		"limit":  pagination.Limit,
	}
	filter := ""
	if q := strings.TrimSpace(query); q != "" {
		// LIKE(..., true) is case-insensitive; %term% matches substrings.
		filter = "FILTER LIKE(d._id, @q, true) OR LIKE(d.email, @q, true) OR LIKE(d.given_name, @q, true) OR LIKE(d.family_name, @q, true) OR LIKE(d.nickname, @q, true) "
		bindVars["q"] = "%" + q + "%"
	}
	aql := fmt.Sprintf("FOR d in %s %sSORT d.created_at DESC LIMIT @offset, @limit RETURN d", schemas.Collections.User, filter)
	cursor, err := p.db.Query(sctx, aql, bindVars)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		user := &schemas.User{}
		meta, err := readDocument(ctx, cursor, user)
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
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		u := &schemas.User{}
		if _, err := readDocument(ctx, cursor, u); err != nil {
			return nil, err
		}
		user = u
	}
	return user, nil
}

// GetUserByExternalID to get user information from database using the
// org-namespaced external id. external_id is stored as "<orgID>:<externalID>"
// so the same IdP external id can map to distinct users across organizations.
func (p *provider) GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error) {
	var user *schemas.User
	query := fmt.Sprintf("FOR d in %s FILTER d.external_id == @extid RETURN d", schemas.Collections.User)
	bindVars := map[string]interface{}{
		"extid": orgID + ":" + externalID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		u := &schemas.User{}
		if _, err := readDocument(ctx, cursor, u); err != nil {
			return nil, err
		}
		user = u
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
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		u := &schemas.User{}
		if _, err := readDocument(ctx, cursor, u); err != nil {
			return nil, err
		}
		user = u
	}
	return user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()
	bindVars := map[string]interface{}{
		"data": data,
	}
	query := ""
	if len(ids) > 0 {
		bindVars["ids"] = ids
		query = fmt.Sprintf("FOR u IN %s FILTER u._id IN @ids UPDATE u._key WITH @data IN %s", schemas.Collections.User, schemas.Collections.User)
	} else {
		query = fmt.Sprintf("FOR u IN %s UPDATE u._key WITH @data IN %s", schemas.Collections.User, schemas.Collections.User)
	}
	_, err := p.db.Query(ctx, query, bindVars)
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
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if user == nil {
				return nil, fmt.Errorf("user not found")
			}
			break
		}
		u := &schemas.User{}
		if _, err := readDocument(ctx, cursor, u); err != nil {
			return nil, err
		}
		user = u
	}
	return user, nil
}
