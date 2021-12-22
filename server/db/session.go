package db

import (
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm/clause"
)

type Session struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty"` // for arangodb
	// ObjectID  string `json:"_id,omitempty" bson:"_id"`             // for arangodb & mongodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	UserID    string `gorm:"type:char(36)" json:"user_id" bson:"user_id"`
	User      User   `json:"-" bson:"-"`
	UserAgent string `json:"user_agent" bson:"user_agent"`
	IP        string `json:"ip" bson:"ip"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"updated_at" bson:"updated_at"`
}

// AddSession function to save user sessiosn
func (mgr *manager) AddSession(session Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	if IsORMSupported {
		session.Key = session.ID
		// session.ObjectID = session.ID
		res := mgr.sqlDB.Clauses(
			clause.OnConflict{
				DoNothing: true,
			}).Create(&session)
		if res.Error != nil {
			log.Println(`error saving session`, res.Error)
			return res.Error
		}
	}

	if IsArangoDB {
		session.CreatedAt = time.Now().Unix()
		session.UpdatedAt = time.Now().Unix()
		sessionCollection, _ := mgr.arangodb.Collection(nil, Collections.Session)
		_, err := sessionCollection.CreateDocument(nil, session)
		if err != nil {
			log.Println(`error saving session`, err)
			return err
		}
	}

	if IsMongoDB {
		session.Key = session.ID
		// session.ObjectID = session.ID
		session.CreatedAt = time.Now().Unix()
		session.UpdatedAt = time.Now().Unix()
		sessionCollection := mgr.mongodb.Collection(Collections.Session, options.Collection())
		_, err := sessionCollection.InsertOne(nil, session)
		if err != nil {
			log.Println(`error saving session`, err)
			return err
		}
	}

	return nil
}
