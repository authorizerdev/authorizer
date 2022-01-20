package db

import (
	"fmt"
	"log"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Env struct {
	Key       string `json:"_key,omitempty" bson:"_key"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	EnvData   []byte `gorm:"type:text" json:"env" bson:"env"`
	Hash      string `gorm:"type:hash" json:"hash" bson:"hash"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"updated_at" bson:"updated_at"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
}

// AddEnv function to add env to db
func (mgr *manager) AddEnv(env Env) (Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	if IsORMSupported {
		// copy id as value for fields required for mongodb & arangodb
		env.Key = env.ID
		result := mgr.sqlDB.Create(&env)

		if result.Error != nil {
			log.Println("error adding config:", result.Error)
			return env, result.Error
		}
	}

	if IsArangoDB {
		env.CreatedAt = time.Now().Unix()
		env.UpdatedAt = time.Now().Unix()
		configCollection, _ := mgr.arangodb.Collection(nil, Collections.Env)
		meta, err := configCollection.CreateDocument(arangoDriver.WithOverwrite(nil), env)
		if err != nil {
			log.Println("error adding config:", err)
			return env, err
		}
		env.Key = meta.Key
		env.ID = meta.ID.String()
	}

	if IsMongoDB {
		env.CreatedAt = time.Now().Unix()
		env.UpdatedAt = time.Now().Unix()
		env.Key = env.ID
		configCollection := mgr.mongodb.Collection(Collections.Env, options.Collection())
		_, err := configCollection.InsertOne(nil, env)
		if err != nil {
			log.Println("error adding config:", err)
			return env, err
		}
	}

	return env, nil
}

// UpdateEnv function to update env in db
func (mgr *manager) UpdateEnv(env Env) (Env, error) {
	env.UpdatedAt = time.Now().Unix()

	if IsORMSupported {
		result := mgr.sqlDB.Save(&env)

		if result.Error != nil {
			log.Println("error updating config:", result.Error)
			return env, result.Error
		}
	}

	if IsArangoDB {
		collection, _ := mgr.arangodb.Collection(nil, Collections.Env)
		meta, err := collection.UpdateDocument(nil, env.Key, env)
		if err != nil {
			log.Println("error updating config:", err)
			return env, err
		}

		env.Key = meta.Key
		env.ID = meta.ID.String()
	}

	if IsMongoDB {
		configCollection := mgr.mongodb.Collection(Collections.Env, options.Collection())
		_, err := configCollection.UpdateOne(nil, bson.M{"_id": bson.M{"$eq": env.ID}}, bson.M{"$set": env}, options.MergeUpdateOptions())
		if err != nil {
			log.Println("error updating config:", err)
			return env, err
		}
	}

	return env, nil
}

// GetConfig function to get config
func (mgr *manager) GetEnv() (Env, error) {
	var env Env

	if IsORMSupported {
		result := mgr.sqlDB.First(&env)

		if result.Error != nil {
			return env, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s RETURN d", Collections.Env)

		cursor, err := mgr.arangodb.Query(nil, query, nil)
		if err != nil {
			return env, err
		}
		defer cursor.Close()

		for {
			if !cursor.HasMore() {
				if env.Key == "" {
					return env, fmt.Errorf("config not found")
				}
				break
			}
			_, err := cursor.ReadDocument(nil, &env)
			if err != nil {
				return env, err
			}
		}
	}

	if IsMongoDB {
		configCollection := mgr.mongodb.Collection(Collections.Env, options.Collection())
		cursor, err := configCollection.Find(nil, bson.M{}, options.Find())
		if err != nil {
			return env, err
		}
		defer cursor.Close(nil)

		for cursor.Next(nil) {
			err := cursor.Decode(&env)
			if err != nil {
				return env, err
			}
		}

		if env.ID == "" {
			return env, fmt.Errorf("config not found")
		}
	}

	return env, nil
}
