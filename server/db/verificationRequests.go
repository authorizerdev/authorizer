package db

import (
	"fmt"
	"log"

	"github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type VerificationRequest struct {
	Key        string `json:"_key,omitempty"` // for arangodb
	ObjectID   string `json:"_id,omitempty"`  // for arangodb & mongodb
	ID         string `gorm:"primaryKey;type:char(36)" json:"id"`
	Token      string `gorm:"type:text" json:"token"`
	Identifier string `gorm:"uniqueIndex:idx_email_identifier" json:"identifier"`
	ExpiresAt  int64  `json:"expires_at"`
	CreatedAt  int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  int64  `gorm:"autoUpdateTime" json:"updated_at"`
	Email      string `gorm:"uniqueIndex:idx_email_identifier" json:"email"`
}

// AddVerification function to add verification record
func (mgr *manager) AddVerification(verification VerificationRequest) (VerificationRequest, error) {
	if verification.ID == "" {
		verification.ID = uuid.New().String()
	}
	if IsSQL {
		// copy id as value for fields required for mongodb & arangodb
		verification.Key = verification.ID
		verification.ObjectID = verification.ID
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
		verificationRequestCollection, _ := mgr.arangodb.Collection(nil, Collections.VerificationRequest)
		meta, err := verificationRequestCollection.CreateDocument(nil, verification)
		if err != nil {
			return verification, err
		}
		verification.Key = meta.Key
		verification.ObjectID = meta.ID.String()
	}
	return verification, nil
}

// GetVerificationRequests function to get all verification requests
func (mgr *manager) GetVerificationRequests() ([]VerificationRequest, error) {
	var verificationRequests []VerificationRequest

	if IsSQL {
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
				verificationRequest.Key = meta.Key
				verificationRequest.ObjectID = meta.ID.String()
				verificationRequests = append(verificationRequests, verificationRequest)
			}

		}
	}
	return verificationRequests, nil
}

func (mgr *manager) GetVerificationByToken(token string) (VerificationRequest, error) {
	var verification VerificationRequest

	if IsSQL {
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

	return verification, nil
}

func (mgr *manager) GetVerificationByEmail(email string) (VerificationRequest, error) {
	var verification VerificationRequest
	if IsSQL {
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

	return verification, nil
}

func (mgr *manager) DeleteVerificationRequest(verificationRequest VerificationRequest) error {
	if IsSQL {
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

	return nil
}
