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

type Config struct {
	Key       string `json:"_key,omitempty" bson:"_key"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	Config    []byte `gorm:"type:text" json:"config" bson:"config"`
	Hash      string `gorm:"type:hash" json:"hash" bson:"hash"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"updated_at" bson:"updated_at"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
}

// AddConfig function to add config
func (mgr *manager) AddConfig(config Config) (Config, error) {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	if IsORMSupported {
		// copy id as value for fields required for mongodb & arangodb
		config.Key = config.ID
		result := mgr.sqlDB.Create(&config)

		if result.Error != nil {
			log.Println("error adding config:", result.Error)
			return config, result.Error
		}
	}

	if IsArangoDB {
		config.CreatedAt = time.Now().Unix()
		config.UpdatedAt = time.Now().Unix()
		configCollection, _ := mgr.arangodb.Collection(nil, Collections.Config)
		meta, err := configCollection.CreateDocument(arangoDriver.WithOverwrite(nil), config)
		if err != nil {
			log.Println("error adding config:", err)
			return config, err
		}
		config.Key = meta.Key
		config.ID = meta.ID.String()
	}

	if IsMongoDB {
		config.CreatedAt = time.Now().Unix()
		config.UpdatedAt = time.Now().Unix()
		config.Key = config.ID
		configCollection := mgr.mongodb.Collection(Collections.Config, options.Collection())
		_, err := configCollection.InsertOne(nil, config)
		if err != nil {
			log.Println("error adding config:", err)
			return config, err
		}
	}

	return config, nil
}

// UpdateConfig function to update config
func (mgr *manager) UpdateConfig(config Config) (Config, error) {
	config.UpdatedAt = time.Now().Unix()

	if IsORMSupported {
		result := mgr.sqlDB.Save(&config)

		if result.Error != nil {
			log.Println("error updating config:", result.Error)
			return config, result.Error
		}
	}

	if IsArangoDB {
		collection, _ := mgr.arangodb.Collection(nil, Collections.Config)
		meta, err := collection.UpdateDocument(nil, config.Key, config)
		if err != nil {
			log.Println("error updating config:", err)
			return config, err
		}

		config.Key = meta.Key
		config.ID = meta.ID.String()
	}

	if IsMongoDB {
		configCollection := mgr.mongodb.Collection(Collections.Config, options.Collection())
		_, err := configCollection.UpdateOne(nil, bson.M{"_id": bson.M{"$eq": config.ID}}, bson.M{"$set": config}, options.MergeUpdateOptions())
		if err != nil {
			log.Println("error updating config:", err)
			return config, err
		}
	}

	return config, nil
}

// GetConfig function to get config
func (mgr *manager) GetConfig() (Config, error) {
	var config Config

	if IsORMSupported {
		result := mgr.sqlDB.First(&config)

		if result.Error != nil {
			return config, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s RETURN d", Collections.Config)

		cursor, err := mgr.arangodb.Query(nil, query, nil)
		if err != nil {
			return config, err
		}
		defer cursor.Close()

		for {
			if !cursor.HasMore() {
				if config.Key == "" {
					return config, fmt.Errorf("config not found")
				}
				break
			}
			_, err := cursor.ReadDocument(nil, &config)
			if err != nil {
				return config, err
			}
		}
	}

	if IsMongoDB {
		configCollection := mgr.mongodb.Collection(Collections.Config, options.Collection())
		cursor, err := configCollection.Find(nil, bson.M{}, options.Find())
		if err != nil {
			return config, err
		}
		defer cursor.Close(nil)

		for cursor.Next(nil) {
			err := cursor.Decode(&config)
			if err != nil {
				return config, err
			}
		}

		if config.ID == "" {
			return config, fmt.Errorf("config not found")
		}
	}

	return config, nil
}
