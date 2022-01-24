package faunadb

import (
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/google/uuid"
)

// AddUser to save user information in database
func (p *provider) AddUser(user models.User) (models.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		user.Roles = strings.Join(envstore.EnvInMemoryStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles), ",")
	}

	user.CreatedAt = time.Now().Unix()
	user.UpdatedAt = time.Now().Unix()

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(user models.User) (models.User, error) {
	user.UpdatedAt = time.Now().Unix()

	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(user models.User) error {
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers() ([]models.User, error) {
	var users []models.User

	return users, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(email string) (models.User, error) {
	var user models.User

	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(id string) (models.User, error) {
	var user models.User

	return user, nil
}
