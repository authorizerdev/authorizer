package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/models/schemas"
)

// AddUser to save user information in database
func (p *provider) AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.Roles == "" {
		defaultRoles, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
		if err != nil {
			return nil, err
		}
		user.Roles = defaultRoles
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
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	_, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser to update user information in database
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	user.UpdatedAt = time.Now().Unix()
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	_, err := userCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": user.ID}}, bson.M{"$set": user}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser to delete user information from database
func (p *provider) DeleteUser(ctx context.Context, user *schemas.User) error {
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	_, err := userCollection.DeleteOne(ctx, bson.M{"_id": user.ID}, options.Delete())
	if err != nil {
		return err
	}
	sessionCollection := p.db.Collection(schemas.Collections.Session, options.Collection())
	_, err = sessionCollection.DeleteMany(ctx, bson.M{"user_id": user.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// ListUsers to get list of users from database
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination) (*model.Users, error) {
	var users []*model.User
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	count, err := userCollection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, err
	}
	paginationClone.Total = count
	cursor, err := userCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var user *schemas.User
		err := cursor.Decode(&user)
		if err != nil {
			return nil, err
		}
		users = append(users, user.AsAPIUser())
	}
	return &model.Users{
		Pagination: paginationClone,
		Users:      users,
	}, nil
}

// GetUserByEmail to get user information from database using email address
func (p *provider) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	var user *schemas.User
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	err := userCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID to get user information from database using user ID
func (p *provider) GetUserByID(ctx context.Context, id string) (*schemas.User, error) {
	var user *schemas.User
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	err := userCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUsers to update multiple users, with parameters of user IDs slice
// If ids set to nil / empty all the users will be updated
func (p *provider) UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error {
	// set updated_at time for all users
	data["updated_at"] = time.Now().Unix()
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	var res *mongo.UpdateResult
	var err error
	if len(ids) > 0 {
		res, err = userCollection.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": ids}}, bson.M{"$set": data})
	} else {
		res, err = userCollection.UpdateMany(ctx, bson.M{}, bson.M{"$set": data})
	}
	if err != nil {
		return err
	} else {
		log.Info("Updated users: ", res.ModifiedCount)
	}
	return nil
}

// GetUserByPhoneNumber to get user information from database using phone number
func (p *provider) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error) {
	var user *schemas.User
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	err := userCollection.FindOne(ctx, bson.M{"phone_number": phoneNumber}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
