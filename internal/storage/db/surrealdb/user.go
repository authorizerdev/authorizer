package surrealdb

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	// Email and phone uniqueness is enforced here, as on every v2 backend,
	// so both stay optional. Each is checked independently: a signup with
	// both must be rejected if EITHER already exists.
	if user.PhoneNumber != nil && strings.TrimSpace(refs.StringValue(user.PhoneNumber)) != "" {
		if u, _ := p.GetUserByPhoneNumber(ctx, refs.StringValue(user.PhoneNumber)); u != nil && u.ID != user.ID {
			return user, fmt.Errorf("user with given phone number already exists")
		}
	}
	if user.Email != nil && strings.TrimSpace(refs.StringValue(user.Email)) != "" {
		if u, _ := p.GetUserByEmail(ctx, refs.StringValue(user.Email)); u != nil && u.ID != user.ID {
			return user, fmt.Errorf("user with given email already exists")
		}
	}
	user.CreatedAt = time.Now().Unix()
	user.UpdatedAt = time.Now().Unix()
	user.Key = user.ID
	if err := p.create(ctx, schemas.Collections.User, user.ID, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateUser: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	user.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.User, user.ID, user); err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser deletes the user and every row it owns, atomically.
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	var stmts strings.Builder
	vars := map[string]any{"uid": user.ID, "utb": schemas.Collections.User}
	for i, collection := range schemas.UserOwnedCollections {
		key := fmt.Sprintf("tb%d", i)
		vars[key] = collection
		fmt.Fprintf(&stmts, "DELETE type::table($%s) WHERE user_id = $uid RETURN NONE;\n", key)
	}
	stmts.WriteString("DELETE type::record($utb, $uid) RETURN NONE;")
	return p.transaction(ctx, stmts.String(), vars)
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination, query string) ([]*schemas.User, *model.Pagination, error) {
	where := ""
	vars := map[string]any{}
	if q := strings.TrimSpace(query); q != "" {
		// Case-insensitive substring match; the input is bound and matched
		// literally, so there are no wildcards to escape.
		where = `string::contains(string::lowercase(record::id(id)), $q)
			OR string::contains(string::lowercase(email ?? ""), $q)
			OR string::contains(string::lowercase(given_name ?? ""), $q)
			OR string::contains(string::lowercase(family_name ?? ""), $q)
			OR string::contains(string::lowercase(nickname ?? ""), $q)`
		vars["q"] = strings.ToLower(q)
	}
	rows, pag, err := page[schemas.User](ctx, p, schemas.Collections.User, where, vars, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	return getWhere[schemas.User](ctx, p, schemas.Collections.User, "email = $v", "user", map[string]any{"v": email})
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	return getByID[schemas.User](ctx, p, schemas.Collections.User, id, "user")
}

// GetUserByExternalID to get user information from database using the
// org-namespaced external ID. external_id is stored as "<orgID>:<externalID>"
// so a SCIM externalId is only ever matched within its own organization.
func (p *provider) GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error) {
	return getWhere[schemas.User](ctx, p, schemas.Collections.User, "external_id = $v", "user", map[string]any{"v": orgID + ":" + externalID})
}

// userFields is the allow-list for UpdateUsers: the stored field names of
// schemas.User.
var userFields = func() map[string]bool {
	out := map[string]bool{}
	for name := range doc(&schemas.User{}) {
		out[name] = true
	}
	return out
}()

// UpdateUsers updates the users identified by ids. An empty ids slice is
// rejected with schemas.ErrUpdateUsersEmptyIDs, so a missing filter can never
// mutate every user; so is any key that is not a User field.
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	if len(ids) == 0 {
		return schemas.ErrUpdateUsersEmptyIDs
	}
	for key := range data {
		if !userFields[key] {
			return fmt.Errorf("UpdateUsers: unknown user field %q", key)
		}
	}
	data["updated_at"] = time.Now().Unix()
	return p.exec(ctx, "UPDATE type::table($tb) MERGE $data WHERE record::id(id) IN $ids RETURN NONE;",
		map[string]any{"tb": schemas.Collections.User, "data": data, "ids": ids})
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	return getWhere[schemas.User](ctx, p, schemas.Collections.User, "phone_number = $v", "user", map[string]any{"v": phoneNumber})
}
