package sql

import (
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// AddUser to save user information in database
func (p *provider) AddUser(user models.User) (models.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		user.Roles = strings.Join(envstore.EnvInMemoryStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles), ",")
	}

	user.Key = user.ID
	result := p.db.Clauses(
		clause.OnConflict{
			UpdateAll: true,
			Columns:   []clause.Column{{Name: "email"}},
		}).Create(&user)

	if result.Error != nil {
		log.Println("error adding user:", result.Error)
		return user, result.Error
	}

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(user models.User) (models.User, error) {
	user.UpdatedAt = time.Now().Unix()

	result := p.db.Save(&user)

	if result.Error != nil {
		log.Println("error updating user:", result.Error)
		return user, result.Error
	}

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(user models.User) error {
	result := p.db.Delete(&user)

	if result.Error != nil {
		log.Println(`error deleting user:`, result.Error)
		return result.Error
	}

	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(pagination model.Pagination) (*model.Users, error) {
	var users []models.User
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&users)
	if result.Error != nil {
		log.Println("error getting users:", result.Error)
		return nil, result.Error
	}

	responseUsers := []*model.User{}
	for _, user := range users {
		responseUsers = append(responseUsers, user.AsAPIUser())
	}

	var total int64
	totalRes := p.db.Model(&models.User{}).Count(&total)
	if totalRes.Error != nil {
		return nil, totalRes.Error
	}

	paginationClone := pagination
	paginationClone.Total = total

	return &model.Users{
		Pagination: &paginationClone,
		Users:      responseUsers,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(email string) (models.User, error) {
	var user models.User
	result := p.db.Where("email = ?", email).First(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(id string) (models.User, error) {
	var user models.User

	result := p.db.Where("id = ?", id).First(&user)

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}
