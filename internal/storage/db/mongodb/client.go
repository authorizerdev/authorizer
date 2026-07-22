package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddClient creates a new service account record.
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	if sa.ClientID == "" {
		sa.ClientID = sa.ID
	}
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	saCollection := p.db.Collection(schemas.Collections.Client, options.Collection())
	_, err := saCollection.InsertOne(ctx, sa)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateClient updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	saCollection := p.db.Collection(schemas.Collections.Client, options.Collection())
	_, err := saCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": sa.ID}}, bson.M{"$set": sa}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteClient removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	saCollection := p.db.Collection(schemas.Collections.Client, options.Collection())
	_, err := saCollection.DeleteOne(ctx, bson.M{"_id": sa.ID}, options.Delete())
	if err != nil {
		return err
	}
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	_, err = issuerCollection.DeleteMany(ctx, bson.M{"client_id": sa.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetClientByID fetches a service account by primary key.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	var sa *schemas.Client
	saCollection := p.db.Collection(schemas.Collections.Client, options.Collection())
	err := saCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&sa)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// GetClientByClientID fetches a client by its unique public client_id.
func (p *provider) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	var sa *schemas.Client
	saCollection := p.db.Collection(schemas.Collections.Client, options.Collection())
	err := saCollection.FindOne(ctx, bson.M{"client_id": clientID}).Decode(&sa)
	if err != nil {
		// No matching document is a normal negative result, not a storage
		// failure — callers (e.g. clientauth.ResolveClient) distinguish "no
		// such client" from "couldn't check" by whether err is nil, so a
		// genuinely absent document must come back as (nil, nil), never a
		// wrapped ErrNoDocuments.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return sa, nil
}

// ListClients returns a paginated list of service accounts.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	clients := []*schemas.Client{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	saCollection := p.db.Collection(schemas.Collections.Client, options.Collection())
	count, err := saCollection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := saCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var sa *schemas.Client
		err := cursor.Decode(&sa)
		if err != nil {
			return nil, nil, err
		}
		clients = append(clients, sa)
	}
	return clients, paginationClone, nil
}
