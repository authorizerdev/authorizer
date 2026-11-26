package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerificationRequest saves a verification request. It is an upsert on
// (email, identifier), matching every other backend: a pending request for the
// same pair is replaced in place (new token, expiry, nonce, redirect URI)
// rather than rejected by the unique index.
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}
	now := time.Now().Unix()
	verificationRequestCollection := p.db.Collection(schemas.Collections.VerificationRequest, options.Collection())
	filter := bson.D{
		{Key: "email", Value: verificationRequest.Email},
		{Key: "identifier", Value: verificationRequest.Identifier},
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "token", Value: verificationRequest.Token},
			{Key: "expires_at", Value: verificationRequest.ExpiresAt},
			{Key: "nonce", Value: verificationRequest.Nonce},
			{Key: "redirect_uri", Value: verificationRequest.RedirectURI},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: verificationRequest.ID},
			{Key: "_key", Value: verificationRequest.ID},
			{Key: "created_at", Value: now},
		}},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	if err := verificationRequestCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(verificationRequest); err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest

	verificationRequestCollection := p.db.Collection(schemas.Collections.VerificationRequest, options.Collection())
	err := verificationRequestCollection.FindOne(ctx, bson.M{"token": token}).Decode(&verificationRequest)
	if err != nil {
		return nil, err
	}

	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest

	verificationRequestCollection := p.db.Collection(schemas.Collections.VerificationRequest, options.Collection())
	err := verificationRequestCollection.FindOne(ctx, bson.M{"email": email, "identifier": identifier}).Decode(&verificationRequest)
	if err != nil {
		return nil, err
	}

	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error) {
	var verificationRequests []*schemas.VerificationRequest
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})

	verificationRequestCollection := p.db.Collection(schemas.Collections.VerificationRequest, options.Collection())

	verificationRequestCollectionCount, err := verificationRequestCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, nil, err
	}
	paginationClone := pagination
	paginationClone.Total = verificationRequestCollectionCount

	cursor, err := verificationRequestCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	for cursor.Next(ctx) {
		var verificationRequest *schemas.VerificationRequest
		err := cursor.Decode(&verificationRequest)
		if err != nil {
			return nil, nil, err
		}
		verificationRequests = append(verificationRequests, verificationRequest)
	}

	return verificationRequests, paginationClone, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	verificationRequestCollection := p.db.Collection(schemas.Collections.VerificationRequest, options.Collection())
	_, err := verificationRequestCollection.DeleteOne(ctx, bson.M{"_id": verificationRequest.ID}, options.Delete())
	if err != nil {
		return err
	}

	return nil
}
