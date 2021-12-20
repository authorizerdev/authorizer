package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type User struct {
	Key             string `json:"_key,omitempty"` // for arangodb
	ObjectID        string `json:"_id,omitempty"`  // for arangodb & mongodb
	ID              string `gorm:"primaryKey;type:char(36)" json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `gorm:"unique" json:"email"`
	Password        string `gorm:"type:text" json:"password"`
	SignupMethod    string `json:"signup_method"`
	EmailVerifiedAt int64  `json:"email_verified_at"`
	CreatedAt       int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       int64  `gorm:"autoUpdateTime" json:"updated_at"`
	Image           string `gorm:"type:text" json:"image"`
	Roles           string `json:"roles"`
}

// AddUser function to add user even with email conflict
func (mgr *manager) AddUser(user User) (User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if IsSQL {
		// copy id as value for fields required for mongodb & arangodb
		user.Key = user.ID
		user.ObjectID = user.ID
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
		ctx := context.Background()
		userCollection, _ := mgr.arangodb.Collection(nil, Collections.User)
		meta, err := userCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), user)
		if err != nil {
			log.Println("error adding user:", err)
			return user, err
		}
		user.Key = meta.Key
		user.ObjectID = meta.ID.String()
	}
	return user, nil
}

// UpdateUser function to update user with ID conflict
func (mgr *manager) UpdateUser(user User) (User, error) {
	user.UpdatedAt = time.Now().Unix()

	if IsSQL {
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
		user.ObjectID = meta.ID.String()
	}
	return user, nil
}

// GetUsers function to get all users
func (mgr *manager) GetUsers() ([]User, error) {
	var users []User

	if IsSQL {
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
				user.Key = meta.Key
				user.ObjectID = meta.ID.String()
				users = append(users, user)
			}

		}
	}
	return users, nil
}

func (mgr *manager) GetUserByEmail(email string) (User, error) {
	var user User

	if IsSQL {
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

	return user, nil
}

func (mgr *manager) GetUserByID(id string) (User, error) {
	var user User

	if IsSQL {
		result := mgr.sqlDB.Where("id = ?", id).First(&user)

		if result.Error != nil {
			return user, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s FILTER d.id == @id LIMIT 1 RETURN d", Collections.User)
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

	return user, nil
}

func (mgr *manager) DeleteUser(user User) error {
	if IsSQL {
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

	return nil
}
