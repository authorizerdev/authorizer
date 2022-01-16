package db

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm/clause"
)

type User struct {
	Key string `json:"_key,omitempty" bson:"_key"` // for arangodb
	ID  string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`

	Email                 string  `gorm:"unique" json:"email" bson:"email"`
	EmailVerifiedAt       *int64  `json:"email_verified_at" bson:"email_verified_at"`
	Password              *string `gorm:"type:text" json:"password" bson:"password"`
	SignupMethods         string  `json:"signup_methods" bson:"signup_methods"`
	GivenName             *string `json:"given_name" bson:"given_name"`
	FamilyName            *string `json:"family_name" bson:"family_name"`
	MiddleName            *string `json:"middle_name" bson:"middle_name"`
	Nickname              *string `json:"nickname" bson:"nickname"`
	Gender                *string `json:"gender" bson:"gender"`
	Birthdate             *string `json:"birthdate" bson:"birthdate"`
	PhoneNumber           *string `gorm:"unique" json:"phone_number" bson:"phone_number"`
	PhoneNumberVerifiedAt *int64  `json:"phone_number_verified_at" bson:"phone_number_verified_at"`
	Picture               *string `gorm:"type:text" json:"picture" bson:"picture"`
	Roles                 string  `json:"roles" bson:"roles"`
	UpdatedAt             int64   `gorm:"autoUpdateTime" json:"updated_at" bson:"updated_at"`
	CreatedAt             int64   `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
}

// AddUser function to add user even with email conflict
func (mgr *manager) AddUser(user User) (User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if user.Roles == "" {
		user.Roles = strings.Join(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDefaultRoles).([]string), ",")
	}

	if IsORMSupported {
		// copy id as value for fields required for mongodb & arangodb
		user.Key = user.ID
		result := mgr.sqlDB.Clauses(
			clause.OnConflict{
				UpdateAll: true,
				Columns:   []clause.Column{{Name: "email"}},
			}).Create(&user)

		if result.Error != nil {
			log.Println("error adding user:", result.Error)
			return user, result.Error
		}
	}

	if IsArangoDB {
		user.CreatedAt = time.Now().Unix()
		user.UpdatedAt = time.Now().Unix()
		userCollection, _ := mgr.arangodb.Collection(nil, Collections.User)
		meta, err := userCollection.CreateDocument(arangoDriver.WithOverwrite(nil), user)
		if err != nil {
			log.Println("error adding user:", err)
			return user, err
		}
		user.Key = meta.Key
		user.ID = meta.ID.String()
	}

	if IsMongoDB {
		user.CreatedAt = time.Now().Unix()
		user.UpdatedAt = time.Now().Unix()
		user.Key = user.ID
		userCollection := mgr.mongodb.Collection(Collections.User, options.Collection())
		_, err := userCollection.InsertOne(nil, user)
		if err != nil {
			log.Println("error adding user:", err)
			return user, err
		}
	}

	return user, nil
}

// UpdateUser function to update user with ID conflict
func (mgr *manager) UpdateUser(user User) (User, error) {
	user.UpdatedAt = time.Now().Unix()

	if IsORMSupported {
		result := mgr.sqlDB.Save(&user)

		if result.Error != nil {
			log.Println("error updating user:", result.Error)
			return user, result.Error
		}
	}

	if IsArangoDB {
		collection, _ := mgr.arangodb.Collection(nil, Collections.User)
		meta, err := collection.UpdateDocument(nil, user.Key, user)
		if err != nil {
			log.Println("error updating user:", err)
			return user, err
		}

		user.Key = meta.Key
		user.ID = meta.ID.String()
	}

	if IsMongoDB {
		userCollection := mgr.mongodb.Collection(Collections.User, options.Collection())
		_, err := userCollection.UpdateOne(nil, bson.M{"_id": bson.M{"$eq": user.ID}}, bson.M{"$set": user}, options.MergeUpdateOptions())
		if err != nil {
			log.Println("error updating user:", err)
			return user, err
		}
	}

	return user, nil
}

// GetUsers function to get all users
func (mgr *manager) GetUsers() ([]User, error) {
	var users []User

	if IsORMSupported {
		result := mgr.sqlDB.Find(&users)
		if result.Error != nil {
			log.Println("error getting users:", result.Error)
			return users, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s RETURN d", Collections.User)

		cursor, err := mgr.arangodb.Query(nil, query, nil)
		if err != nil {
			return users, err
		}
		defer cursor.Close()

		for {
			var user User
			meta, err := cursor.ReadDocument(nil, &user)

			if driver.IsNoMoreDocuments(err) {
				break
			} else if err != nil {
				return users, err
			}

			if meta.Key != "" {
				users = append(users, user)
			}
		}
	}

	if IsMongoDB {
		userCollection := mgr.mongodb.Collection(Collections.User, options.Collection())
		cursor, err := userCollection.Find(nil, bson.M{}, options.Find())
		if err != nil {
			log.Println("error getting users:", err)
			return users, err
		}
		defer cursor.Close(nil)

		for cursor.Next(nil) {
			var user User
			err := cursor.Decode(&user)
			if err != nil {
				return users, err
			}
			users = append(users, user)
		}
	}

	return users, nil
}

// GetUserByEmail function to get user by email
func (mgr *manager) GetUserByEmail(email string) (User, error) {
	var user User

	if IsORMSupported {
		result := mgr.sqlDB.Where("email = ?", email).First(&user)

		if result.Error != nil {
			return user, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s FILTER d.email == @email RETURN d", Collections.User)
		bindVars := map[string]interface{}{
			"email": email,
		}

		cursor, err := mgr.arangodb.Query(nil, query, bindVars)
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
	}

	if IsMongoDB {
		userCollection := mgr.mongodb.Collection(Collections.User, options.Collection())
		err := userCollection.FindOne(nil, bson.M{"email": email}).Decode(&user)
		if err != nil {
			return user, err
		}
	}

	return user, nil
}

// GetUserByID function to get user by ID
func (mgr *manager) GetUserByID(id string) (User, error) {
	var user User

	if IsORMSupported {
		result := mgr.sqlDB.Where("id = ?", id).First(&user)

		if result.Error != nil {
			return user, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s FILTER d._id == @id LIMIT 1 RETURN d", Collections.User)
		bindVars := map[string]interface{}{
			"id": id,
		}

		cursor, err := mgr.arangodb.Query(nil, query, bindVars)
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
	}

	if IsMongoDB {
		userCollection := mgr.mongodb.Collection(Collections.User, options.Collection())
		err := userCollection.FindOne(nil, bson.M{"_id": id}).Decode(&user)
		if err != nil {
			return user, err
		}
	}

	return user, nil
}

// DeleteUser function to delete user
func (mgr *manager) DeleteUser(user User) error {
	if IsORMSupported {
		result := mgr.sqlDB.Delete(&user)

		if result.Error != nil {
			log.Println(`error deleting user:`, result.Error)
			return result.Error
		}
	}

	if IsArangoDB {
		collection, _ := mgr.arangodb.Collection(nil, Collections.User)
		_, err := collection.RemoveDocument(nil, user.Key)
		if err != nil {
			log.Println(`error deleting user:`, err)
			return err
		}
	}

	if IsMongoDB {
		userCollection := mgr.mongodb.Collection(Collections.User, options.Collection())
		_, err := userCollection.DeleteOne(nil, bson.M{"_id": user.ID}, options.Delete())
		if err != nil {
			log.Println("error deleting user:", err)
			return err
		}
	}

	return nil
}
