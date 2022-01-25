package mongodb

import (
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	user.Key = user.ID
	userCollection := p.db.Collection(models.Collections.User, options.Collection())
	_, err := userCollection.InsertOne(nil, user)
	if err != nil {
		log.Println("error adding user:", err)
		return user, err
	}

	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(user models.User) (models.User, error) {
	user.UpdatedAt = time.Now().Unix()
	userCollection := p.db.Collection(models.Collections.User, options.Collection())
	_, err := userCollection.UpdateOne(nil, bson.M{"_id": bson.M{"$eq": user.ID}}, bson.M{"$set": user}, options.MergeUpdateOptions())
	if err != nil {
		log.Println("error updating user:", err)
		return user, err
	}
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(user models.User) error {
	userCollection := p.db.Collection(models.Collections.User, options.Collection())
	_, err := userCollection.DeleteOne(nil, bson.M{"_id": user.ID}, options.Delete())
	if err != nil {
		log.Println("error deleting user:", err)
		return err
	}

	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(pagination model.Pagination) (*model.Users, error) {
	var users []*model.User
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})

	paginationClone := pagination
	// TODO add pagination total

	userCollection := p.db.Collection(models.Collections.User, options.Collection())
	count, err := userCollection.CountDocuments(nil, bson.M{}, options.Count())
	if err != nil {
		log.Println("error getting total users:", err)
		return nil, err
	}

	paginationClone.Total = count

	cursor, err := userCollection.Find(nil, bson.M{}, opts)
	if err != nil {
		log.Println("error getting users:", err)
		return nil, err
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
		var user models.User
		err := cursor.Decode(&user)
		if err != nil {
			return nil, err
		}
		users = append(users, user.AsAPIUser())
	}

	return &model.Users{
		Pagination: &paginationClone,
		Users:      users,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(email string) (models.User, error) {
	var user models.User
	userCollection := p.db.Collection(models.Collections.User, options.Collection())
	err := userCollection.FindOne(nil, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return user, err
	}

	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(id string) (models.User, error) {
	var user models.User

	userCollection := p.db.Collection(models.Collections.User, options.Collection())
	err := userCollection.FindOne(nil, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return user, err
	}

	return user, nil
}
