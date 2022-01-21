package mongodb

import (
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddSession to save session information in database
func (p *provider) AddSession(session models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	session.Key = session.ID
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	sessionCollection := p.db.Collection(models.Collections.Session, options.Collection())
	_, err := sessionCollection.InsertOne(nil, session)
	if err != nil {
		log.Println(`error saving session`, err)
		return err
	}
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(userId string) error {
	sessionCollection := p.db.Collection(models.Collections.Session, options.Collection())
	_, err := sessionCollection.DeleteMany(nil, bson.M{"user_id": userId}, options.Delete())
	if err != nil {
		log.Println("error deleting session:", err)
		return err
	}
	return nil
}
