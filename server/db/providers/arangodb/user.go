package arangodb

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
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
	userCollection, _ := p.db.Collection(nil, models.Collections.User)
	meta, err := userCollection.CreateDocument(arangoDriver.WithOverwrite(nil), user)
	if err != nil {
		log.Println("error adding user:", err)
		return user, err
	}
	user.Key = meta.Key
	user.ID = meta.ID.String()

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(user models.User) (models.User, error) {
	user.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(nil, models.Collections.User)
	meta, err := collection.UpdateDocument(nil, user.Key, user)
	if err != nil {
		log.Println("error updating user:", err)
		return user, err
	}

	user.Key = meta.Key
	user.ID = meta.ID.String()
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(user models.User) error {
	collection, _ := p.db.Collection(nil, models.Collections.User)
	_, err := collection.RemoveDocument(nil, user.Key)
	if err != nil {
		log.Println(`error deleting user:`, err)
		return err
	}

	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(pagination model.Pagination) (*model.Users, error) {
	var users []*model.User
	ctx := driver.WithQueryFullCount(context.Background())

	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", models.Collections.User, pagination.Offset, pagination.Limit)

	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()

	for {
		var user models.User
		meta, err := cursor.ReadDocument(nil, &user)

		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}

		if meta.Key != "" {
			users = append(users, user.AsAPIUser())
		}
	}

	return &model.Users{
		Pagination: &paginationClone,
		Users:      users,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(email string) (models.User, error) {
	var user models.User

	query := fmt.Sprintf("FOR d in %s FILTER d.email == @email RETURN d", models.Collections.User)
	bindVars := map[string]interface{}{
		"email": email,
	}

	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		return user, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if user.Key == "" {
				return user, fmt.Errorf("user not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &user)
		if err != nil {
			return user, err
		}
	}

	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(id string) (models.User, error) {
	var user models.User

	query := fmt.Sprintf("FOR d in %s FILTER d._id == @id LIMIT 1 RETURN d", models.Collections.User)
	bindVars := map[string]interface{}{
		"id": id,
	}

	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		return user, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if user.Key == "" {
				return user, fmt.Errorf("user not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &user)
		if err != nil {
			return user, err
		}
	}

	return user, nil
}
