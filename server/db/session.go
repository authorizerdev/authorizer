package db

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm/clause"
)

type Session struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	UserID    string `gorm:"type:char(36),index:" json:"user_id" bson:"user_id"`
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

func (mgr *manager) DeleteUserSession(userId string) error {
	if IsORMSupported {
		result := mgr.sqlDB.Where("user_id = ?", userId).Delete(&Session{})

		if result.Error != nil {
			log.Println(`error deleting session:`, result.Error)
			return result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf(`FOR d IN %s FILTER d.user_id == @userId REMOVE { _key: d._key } IN %s`, Collections.Session, Collections.Session)
		bindVars := map[string]interface{}{
			"userId": userId,
		}
		cursor, err := mgr.arangodb.Query(nil, query, bindVars)
		if err != nil {
			log.Println("=> error deleting arangodb session:", err)
			return err
		}
		defer cursor.Close()
	}

	if IsMongoDB {
		sessionCollection := mgr.mongodb.Collection(Collections.Session, options.Collection())
		_, err := sessionCollection.DeleteMany(nil, bson.M{"user_id": userId}, options.Delete())
		if err != nil {
			log.Println("error deleting session:", err)
			return err
		}
	}

	return nil
}
