package db

import (
	"fmt"
	"log"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm/clause"
)

type VerificationRequest struct {
	Key        string `json:"_key,omitempty" bson:"_key"` // for arangodb
	ID         string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`
	Token      string `gorm:"type:text" json:"token" bson:"token"`
	Identifier string `gorm:"uniqueIndex:idx_email_identifier" json:"identifier" bson:"identifier"`
	ExpiresAt  int64  `json:"expires_at" bson:"expires_at"`
	CreatedAt  int64  `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
	UpdatedAt  int64  `gorm:"autoUpdateTime" json:"updated_at" bson:"updated_at"`
	Email      string `gorm:"uniqueIndex:idx_email_identifier" json:"email" bson:"email"`
}

// AddVerification function to add verification record
func (mgr *manager) AddVerification(verification VerificationRequest) (VerificationRequest, error) {
	if verification.ID == "" {
		verification.ID = uuid.New().String()
	}
	if IsORMSupported {
		// copy id as value for fields required for mongodb & arangodb
		verification.Key = verification.ID
		result := mgr.sqlDB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}, {Name: "identifier"}},
			DoUpdates: clause.AssignmentColumns([]string{"token", "expires_at"}),
		}).Create(&verification)

		if result.Error != nil {
			log.Println(`error saving verification record`, result.Error)
			return verification, result.Error
		}
	}

	if IsArangoDB {
		verification.CreatedAt = time.Now().Unix()
		verification.UpdatedAt = time.Now().Unix()
		verificationRequestCollection, _ := mgr.arangodb.Collection(nil, Collections.VerificationRequest)
		meta, err := verificationRequestCollection.CreateDocument(nil, verification)
		if err != nil {
			log.Println("error saving verification record:", err)
			return verification, err
		}
		verification.Key = meta.Key
		verification.ID = meta.ID.String()
	}

	if IsMongoDB {
		verification.CreatedAt = time.Now().Unix()
		verification.UpdatedAt = time.Now().Unix()
		verification.Key = verification.ID
		verificationRequestCollection := mgr.mongodb.Collection(Collections.VerificationRequest, options.Collection())
		_, err := verificationRequestCollection.InsertOne(nil, verification)
		if err != nil {
			log.Println("error saving verification record:", err)
			return verification, err
		}
	}

	return verification, nil
}

// GetVerificationRequests function to get all verification requests
func (mgr *manager) GetVerificationRequests() ([]VerificationRequest, error) {
	var verificationRequests []VerificationRequest

	if IsORMSupported {
		result := mgr.sqlDB.Find(&verificationRequests)
		if result.Error != nil {
			log.Println("error getting verification requests:", result.Error)
			return verificationRequests, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s RETURN d", Collections.VerificationRequest)

		cursor, err := mgr.arangodb.Query(nil, query, nil)
		if err != nil {
			return verificationRequests, err
		}
		defer cursor.Close()

		for {
			var verificationRequest VerificationRequest
			meta, err := cursor.ReadDocument(nil, &verificationRequest)

			if driver.IsNoMoreDocuments(err) {
				break
			} else if err != nil {
				return verificationRequests, err
			}

			if meta.Key != "" {
				verificationRequests = append(verificationRequests, verificationRequest)
			}

		}
	}

	if IsMongoDB {
		verificationRequestCollection := mgr.mongodb.Collection(Collections.VerificationRequest, options.Collection())
		cursor, err := verificationRequestCollection.Find(nil, bson.M{}, options.Find())
		if err != nil {
			log.Println("error getting verification requests:", err)
			return verificationRequests, err
		}
		defer cursor.Close(nil)

		for cursor.Next(nil) {
			var verificationRequest VerificationRequest
			err := cursor.Decode(&verificationRequest)
			if err != nil {
				return verificationRequests, err
			}
			verificationRequests = append(verificationRequests, verificationRequest)
		}
	}

	return verificationRequests, nil
}

func (mgr *manager) GetVerificationByToken(token string) (VerificationRequest, error) {
	var verification VerificationRequest

	if IsORMSupported {
		result := mgr.sqlDB.Where("token = ?", token).First(&verification)

		if result.Error != nil {
			log.Println(`error getting verification request:`, result.Error)
			return verification, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s FILTER d.token == @token LIMIT 1 RETURN d", Collections.VerificationRequest)
		bindVars := map[string]interface{}{
			"token": token,
		}

		cursor, err := mgr.arangodb.Query(nil, query, bindVars)
		if err != nil {
			return verification, err
		}
		defer cursor.Close()

		for {
			if !cursor.HasMore() {
				if verification.Key == "" {
					return verification, fmt.Errorf("verification request not found")
				}
				break
			}
			_, err := cursor.ReadDocument(nil, &verification)
			if err != nil {
				return verification, err
			}
		}
	}

	if IsMongoDB {
		verificationRequestCollection := mgr.mongodb.Collection(Collections.VerificationRequest, options.Collection())
		err := verificationRequestCollection.FindOne(nil, bson.M{"token": token}).Decode(&verification)
		if err != nil {
			return verification, err
		}
	}

	return verification, nil
}

func (mgr *manager) GetVerificationByEmail(email string) (VerificationRequest, error) {
	var verification VerificationRequest
	if IsORMSupported {
		result := mgr.sqlDB.Where("email = ?", email).First(&verification)

		if result.Error != nil {
			log.Println(`error getting verification token:`, result.Error)
			return verification, result.Error
		}
	}

	if IsArangoDB {
		query := fmt.Sprintf("FOR d in %s FILTER d.email == @email LIMIT 1 RETURN d", Collections.VerificationRequest)
		bindVars := map[string]interface{}{
			"email": email,
		}

		cursor, err := mgr.arangodb.Query(nil, query, bindVars)
		if err != nil {
			return verification, err
		}
		defer cursor.Close()

		for {
			if !cursor.HasMore() {
				if verification.Key == "" {
					return verification, fmt.Errorf("verification request not found")
				}
				break
			}
			_, err := cursor.ReadDocument(nil, &verification)
			if err != nil {
				return verification, err
			}
		}
	}

	if IsMongoDB {
		verificationRequestCollection := mgr.mongodb.Collection(Collections.VerificationRequest, options.Collection())
		err := verificationRequestCollection.FindOne(nil, bson.M{"email": email}).Decode(&verification)
		if err != nil {
			return verification, err
		}
	}

	return verification, nil
}

func (mgr *manager) DeleteVerificationRequest(verificationRequest VerificationRequest) error {
	if IsORMSupported {
		result := mgr.sqlDB.Delete(&verificationRequest)

		if result.Error != nil {
			log.Println(`error deleting verification request:`, result.Error)
			return result.Error
		}
	}

	if IsArangoDB {
		collection, _ := mgr.arangodb.Collection(nil, Collections.VerificationRequest)
		_, err := collection.RemoveDocument(nil, verificationRequest.Key)
		if err != nil {
			log.Println(`error deleting verification request:`, err)
			return err
		}
	}

	if IsMongoDB {
		verificationRequestCollection := mgr.mongodb.Collection(Collections.VerificationRequest, options.Collection())
		_, err := verificationRequestCollection.DeleteOne(nil, bson.M{"id": verificationRequest.ID}, options.Delete())
		if err != nil {
			log.Println("error deleting verification request::", err)
			return err
		}
	}

	return nil
}
