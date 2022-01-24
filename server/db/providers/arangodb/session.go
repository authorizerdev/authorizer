package arangodb

import (
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddSession to save session information in database
func (p *provider) AddSession(session models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	sessionCollection, _ := p.db.Collection(nil, models.Collections.Session)
	_, err := sessionCollection.CreateDocument(nil, session)
	if err != nil {
		log.Println(`error saving session`, err)
		return err
	}
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(userId string) error {
	query := fmt.Sprintf(`FOR d IN %s FILTER d.user_id == @userId REMOVE { _key: d._key } IN %s`, models.Collections.Session, models.Collections.Session)
	bindVars := map[string]interface{}{
		"userId": userId,
	}
	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		log.Println("=> error deleting arangodb session:", err)
		return err
	}
	defer cursor.Close()
	return nil
}
