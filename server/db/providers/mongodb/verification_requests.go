package mongodb

import (
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(verificationRequest models.VerificationRequest) (models.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()

		verificationRequest.CreatedAt = time.Now().Unix()
		verificationRequest.UpdatedAt = time.Now().Unix()
		verificationRequest.Key = verificationRequest.ID
		verificationRequestCollection := p.db.Collection(models.Collections.VerificationRequest, options.Collection())
		_, err := verificationRequestCollection.InsertOne(nil, verificationRequest)
		if err != nil {
			log.Println("error saving verification record:", err)
			return verificationRequest, err
		}
	}

	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(token string) (models.VerificationRequest, error) {
	var verificationRequest models.VerificationRequest

	verificationRequestCollection := p.db.Collection(models.Collections.VerificationRequest, options.Collection())
	err := verificationRequestCollection.FindOne(nil, bson.M{"token": token}).Decode(&verificationRequest)
	if err != nil {
		return verificationRequest, err
	}

	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(email string, identifier string) (models.VerificationRequest, error) {
	var verificationRequest models.VerificationRequest

	verificationRequestCollection := p.db.Collection(models.Collections.VerificationRequest, options.Collection())
	err := verificationRequestCollection.FindOne(nil, bson.M{"email": email, "identifier": identifier}).Decode(&verificationRequest)
	if err != nil {
		return verificationRequest, err
	}

	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests() ([]models.VerificationRequest, error) {
	var verificationRequests []models.VerificationRequest
	verificationRequestCollection := p.db.Collection(models.Collections.VerificationRequest, options.Collection())
	cursor, err := verificationRequestCollection.Find(nil, bson.M{}, options.Find())
	if err != nil {
		log.Println("error getting verification requests:", err)
		return verificationRequests, err
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
		var verificationRequest models.VerificationRequest
		err := cursor.Decode(&verificationRequest)
		if err != nil {
			return verificationRequests, err
		}
		verificationRequests = append(verificationRequests, verificationRequest)
	}

	return verificationRequests, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(verificationRequest models.VerificationRequest) error {
	verificationRequestCollection := p.db.Collection(models.Collections.VerificationRequest, options.Collection())
	_, err := verificationRequestCollection.DeleteOne(nil, bson.M{"_id": verificationRequest.ID}, options.Delete())
	if err != nil {
		log.Println("error deleting verification request::", err)
		return err
	}

	return nil
}
