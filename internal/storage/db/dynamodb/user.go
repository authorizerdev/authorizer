package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// normalizeUserOptionalPtrs clears *int64 fields that round-trip as 0 from DynamoDB where other
// providers use SQL NULL — tests and handlers treat nil as "unset" for these fields.
func normalizeUserOptionalPtrs(u *schemas.User) {
	if u == nil {
		return
	}
	if u.EmailVerifiedAt != nil && *u.EmailVerifiedAt == 0 {
		u.EmailVerifiedAt = nil
	}
	if u.PhoneNumberVerifiedAt != nil && *u.PhoneNumberVerifiedAt == 0 {
		u.PhoneNumberVerifiedAt = nil
	}
	if u.RevokedTimestamp != nil && *u.RevokedTimestamp == 0 {
		u.RevokedTimestamp = nil
	}
}

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
	if err := p.putItem(ctx, schemas.Collections.User, user); err != nil {
		return nil, err
	}
	return user, nil
}

// userDynamoRemoveAttrsIfNil lists attribute names to REMOVE so that optional nil-pointer fields
// match SQL NULL semantics (omitting them from SET in DynamoDB would otherwise leave old values).
func userDynamoRemoveAttrsIfNil(u *schemas.User) []string {
	if u == nil {
		return nil
	}
	var remove []string
	if u.EmailVerifiedAt == nil {
		remove = append(remove, "email_verified_at")
	}
	if u.PhoneNumberVerifiedAt == nil {
		remove = append(remove, "phone_number_verified_at")
	}
	if u.RevokedTimestamp == nil {
		remove = append(remove, "revoked_timestamp")
	}
	return remove
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.ID != "" {
		user.UpdatedAt = time.Now().Unix()
		remove := userDynamoRemoveAttrsIfNil(user)
		if err := p.updateByHashKeyWithRemoves(ctx, schemas.Collections.User, "id", user.ID, user, remove); err != nil {
			return nil, err
		}
	}
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	if user.ID == "" {
		return nil
	}
	if err := p.deleteItemByHash(ctx, schemas.Collections.User, "id", user.ID); err != nil {
		return err
	}
	items, err := p.queryEq(ctx, schemas.Collections.Session, "user_id", "user_id", user.ID, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var s schemas.Session
		if err := unmarshalItem(it, &s); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.Session, "id", s.ID); err != nil {
			return err
		}
	}
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) ([]*schemas.User, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var users []*schemas.User

	count, err := p.scanCount(ctx, schemas.Collections.User, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.User, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var u schemas.User
			if err := unmarshalItem(it, &u); err != nil {
				return nil, nil, err
			}
			normalizeUserOptionalPtrs(&u)
			if paginationClone.Offset == iteration {
				users = append(users, &u)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return users, paginationClone, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	items, err := p.queryEq(ctx, schemas.Collections.User, "email", "email", email, nil)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no record found")
	}
	var u schemas.User
	if err := unmarshalItem(items[0], &u); err != nil {
		return nil, err
	}
	normalizeUserOptionalPtrs(&u)
	return &u, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	var user schemas.User
	err := p.getItemByHash(ctx, schemas.Collections.User, "id", id, &user)
	if err != nil {
		return nil, errors.New("no documets found")
	}
	normalizeUserOptionalPtrs(&user)
	return &user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	var res int64
	var err error
	if len(ids) > 0 {
		for _, v := range ids {
			err = p.updateByHashKey(ctx, schemas.Collections.User, "id", v, data)
		}
	} else {
		items, errScan := p.scanAllRaw(ctx, schemas.Collections.User, nil, nil)
		if errScan != nil {
			return errScan
		}
		for _, it := range items {
			var user schemas.User
			if err := unmarshalItem(it, &user); err != nil {
				return err
			}
			err = p.updateByHashKey(ctx, schemas.Collections.User, "id", user.ID, data)
			if err == nil {
				res++
			}
		}
	}
	if err != nil {
		return err
	}
	p.dependencies.Log.Info().Int64("modified_count", res).Msg("users updated")
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	f := expression.Name("phone_number").Equal(expression.Value(phoneNumber))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.User, nil, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no record found")
	}
	var u schemas.User
	if err := unmarshalItem(items[0], &u); err != nil {
		return nil, err
	}
	normalizeUserOptionalPtrs(&u)
	return &u, nil
}
