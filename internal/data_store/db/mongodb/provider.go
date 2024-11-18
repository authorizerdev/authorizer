package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

// Dependencies struct the mongodb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       config.Config
	dependencies Dependencies
	db           *mongo.Database
}

// NewProvider to initialize mongodb connection
func NewProvider(config config.Config, deps Dependencies) (*provider, error) {
	dbURL := config.DatabaseURL
	mongodbOptions := options.Client().ApplyURI(dbURL)
	maxWait := time.Duration(5 * time.Second)
	mongodbOptions.ConnectTimeout = &maxWait
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	mongoClient, err := mongo.Connect(ctx, mongodbOptions)
	if err != nil {
		return nil, err
	}

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	dbName := config.DatabaseName
	if dbName == "" {
		return nil, fmt.Errorf("database name is required for mongodb")
	}
	mongodb := mongoClient.Database(dbName, options.Database())

	mongodb.CreateCollection(ctx, schemas.Collections.User, options.CreateCollection())
	userCollection := mongodb.Collection(schemas.Collections.User, options.Collection())
	userCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"email": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.M{"phone_number": 1},
			Options: options.Index().SetUnique(true).SetSparse(true).SetPartialFilterExpression(map[string]interface{}{
				"phone_number": map[string]string{"$type": "string"},
			}),
		},
	}, options.CreateIndexes())
	mongodb.CreateCollection(ctx, schemas.Collections.VerificationRequest, options.CreateCollection())
	verificationRequestCollection := mongodb.Collection(schemas.Collections.VerificationRequest, options.Collection())
	verificationRequestCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"email": 1, "identifier": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())
	verificationRequestCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"token": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, schemas.Collections.Session, options.CreateCollection())
	sessionCollection := mongodb.Collection(schemas.Collections.Session, options.Collection())
	sessionCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, schemas.Collections.Env, options.CreateCollection())

	mongodb.CreateCollection(ctx, schemas.Collections.Webhook, options.CreateCollection())
	webhookCollection := mongodb.Collection(schemas.Collections.Webhook, options.Collection())
	webhookCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"event_name": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, schemas.Collections.WebhookLog, options.CreateCollection())
	webhookLogCollection := mongodb.Collection(schemas.Collections.WebhookLog, options.Collection())
	webhookLogCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"webhook_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, schemas.Collections.EmailTemplate, options.CreateCollection())
	emailTemplateCollection := mongodb.Collection(schemas.Collections.EmailTemplate, options.Collection())
	emailTemplateCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"event_name": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, schemas.Collections.OTP, options.CreateCollection())
	otpCollection := mongodb.Collection(schemas.Collections.OTP, options.Collection())
	otpCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"email": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())
	otpCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"phone_number": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, schemas.Collections.Authenticators, options.CreateCollection())
	authenticatorsCollection := mongodb.Collection(schemas.Collections.Authenticators, options.Collection())
	authenticatorsCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	return &provider{
		config:       config,
		dependencies: deps,
		db:           mongodb,
	}, nil
}
