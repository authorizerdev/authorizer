package sql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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

	// Check email and phone uniqueness independently: a signup supplying both
	// must be rejected if EITHER already exists. These were previously chained
	// with else-if, which skipped the email check whenever a phone number was
	// also supplied and let duplicate emails persist.
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
	result := p.db.Create(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

// UpdateUser to update user information in database.
//
// Callers MUST load the existing record (e.g. GetUserByID / GetUserByEmail)
// before mutating and passing it here. GORM's Save writes every column, so a
// partially-populated struct would silently blank Password, Roles and any other
// unset field. A zero CreatedAt means the struct was never loaded from the
// database, so reject it to prevent that data loss.
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.CreatedAt == 0 {
		return user, fmt.Errorf("cannot update user: record not loaded (created_at is zero — partial struct detected)")
	}
	user.UpdatedAt = time.Now().Unix()

	result := p.db.Save(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	// Delete the user and their sessions atomically so a failure cannot leave
	// orphaned session rows behind.
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", user.ID).Delete(&schemas.Session{}).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	})
}

// ListUsers to get list of users from database. When query is non-empty it is
// applied as a case-insensitive substring filter (indexed columns use plain
// LIKE; id/email/given_name/family_name/nickname are matched with LOWER(...) LIKE).
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination, query string) ([]*schemas.User, *model.Pagination, error) {
	var users []*schemas.User
	listQuery := p.db.Model(&schemas.User{})
	countQuery := p.db.Model(&schemas.User{})
	if q := strings.TrimSpace(query); q != "" {
		pattern := "%" + strings.ToLower(q) + "%"
		const where = "LOWER(id) LIKE ? OR LOWER(email) LIKE ? OR LOWER(given_name) LIKE ? OR LOWER(family_name) LIKE ? OR LOWER(nickname) LIKE ?"
		listQuery = listQuery.Where(where, pattern, pattern, pattern, pattern, pattern)
		countQuery = countQuery.Where(where, pattern, pattern, pattern, pattern, pattern)
	}

	result := listQuery.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&users)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	var total int64
	totalRes := countQuery.Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}

	paginationClone := pagination
	paginationClone.Total = total

	return users, paginationClone, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	var user *schemas.User
	result := p.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	var user *schemas.User
	result := p.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

// UpdateUsers to update multiple users, identified by the ids slice.
// If ids is nil / empty NO update is performed: GORM's AllowGlobalUpdate is
// disabled (see NewProvider), so the call returns gorm.ErrMissingWhereClause
// rather than silently updating every user — a deliberate fail-safe.
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()
	var res *gorm.DB
	if len(ids) > 0 {
		res = p.db.Model(&schemas.User{}).Where("id in ?", ids).Updates(data)
	} else {
		res = p.db.Model(&schemas.User{}).Updates(data)
	}
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// GetUserByExternalID fetches an IdP-provisioned user by its org-namespaced
// external ID. The lookup key is the composite "<orgID>:<externalID>".
func (p *provider) GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error) {
	var user *schemas.User
	result := p.db.Where("external_id = ?", orgID+":"+externalID).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	var user *schemas.User
	result := p.db.Where("phone_number = ?", phoneNumber).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}
