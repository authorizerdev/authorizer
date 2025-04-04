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
	user.Key = user.ID
	result := p.db.Create(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	user.UpdatedAt = time.Now().Unix()

	result := p.db.Save(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	result := p.db.Where("user_id = ?", user.ID).Delete(&schemas.Session{})
	if result.Error != nil {
		return result.Error
	}

	result = p.db.Delete(&user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) ([]*schemas.User, *model.Pagination, error) {
	var users []*schemas.User
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&users)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	var total int64
	totalRes := p.db.Model(&schemas.User{}).Count(&total)
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

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
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

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	var user *schemas.User
	result := p.db.Where("phone_number = ?", phoneNumber).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}
