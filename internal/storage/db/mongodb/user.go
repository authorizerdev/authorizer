package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	_, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser to update user information in database
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	if user.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateUser: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
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
func (p *provider) ListUsers(ctx context.Context, pagination *model.Pagination, query string) ([]*schemas.User, *model.Pagination, error) {
	var users []*schemas.User
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	filter := bson.M{}
	if q := strings.TrimSpace(query); q != "" {
		// QuoteMeta so user input is treated as a literal substring, not a regex.
		rx := bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}
		filter = bson.M{"$or": []bson.M{
			{"email": rx},
			{"given_name": rx},
			{"family_name": rx},
			{"nickname": rx},
		}}
	}
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	count, err := userCollection.CountDocuments(ctx, filter, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := userCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var user *schemas.User
		err := cursor.Decode(&user)
		if err != nil {
			return nil, nil, err
		}
		users = append(users, user)
	}
	return users, paginationClone, nil
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

// GetUserByExternalID to get user information from database using the
// org-namespaced external ID. external_id is stored as "<orgID>:<externalID>"
// so a SCIM externalId is only ever matched within its own organization.
func (p *provider) GetUserByExternalID(ctx context.Context, orgID, externalID string) (*schemas.User, error) {
	var user *schemas.User
	userCollection := p.db.Collection(schemas.Collections.User, options.Collection())
	err := userCollection.FindOne(ctx, bson.M{"external_id": orgID + ":" + externalID}).Decode(&user)
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
		p.dependencies.Log.Info().Int64("modified_count", res.ModifiedCount).Msg("users updated")
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
