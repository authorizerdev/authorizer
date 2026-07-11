package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebauthnCredential persists a newly registered passkey.
func (p *provider) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.Key = cred.ID
	now := time.Now().Unix()
	cred.CreatedAt = now
	cred.UpdatedAt = now
	credCollection := p.db.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	_, err := credCollection.InsertOne(ctx, cred)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// UpdateWebauthnCredential writes back the record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	credCollection := p.db.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	_, err := credCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": cred.ID}}, bson.M{"$set": cred}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	credCollection := p.db.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	_, err := credCollection.DeleteOne(ctx, bson.M{"_id": cred.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	var cred *schemas.WebauthnCredential
	credCollection := p.db.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	err := credCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&cred)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique WebAuthn
// credential id — the usernameless-login lookup.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	var cred *schemas.WebauthnCredential
	credCollection := p.db.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	err := credCollection.FindOne(ctx, bson.M{"credential_id": credentialID}).Decode(&cred)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys, newest first.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	creds := []*schemas.WebauthnCredential{}
	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})
	credCollection := p.db.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	cursor, err := credCollection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var cred *schemas.WebauthnCredential
		err := cursor.Decode(&cred)
		if err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}
	return creds, nil
}
