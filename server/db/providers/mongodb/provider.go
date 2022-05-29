package mongodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type provider struct {
	db *mongo.Database
}

// NewProvider to initialize mongodb connection
func NewProvider() (*provider, error) {
	dbURL, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)
	if err != nil {
		return nil, err
	}
	mongodbOptions := options.Client().ApplyURI(dbURL)
	maxWait := time.Duration(5 * time.Second)
	mongodbOptions.ConnectTimeout = &maxWait
	mongoClient, err := mongo.NewClient(mongodbOptions)
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = mongoClient.Connect(ctx)
	if err != nil {
		return nil, err
	}

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	dbName, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName)
	if err != nil {
		return nil, err
	}
	mongodb := mongoClient.Database(dbName, options.Database())

	mongodb.CreateCollection(ctx, models.Collections.User, options.CreateCollection())
	userCollection := mongodb.Collection(models.Collections.User, options.Collection())
	userCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"email": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())
	userCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.M{"phone_number": 1},
			Options: options.Index().SetUnique(true).SetSparse(true).SetPartialFilterExpression(map[string]interface{}{
				"phone_number": map[string]string{"$type": "string"},
			}),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, models.Collections.VerificationRequest, options.CreateCollection())
	verificationRequestCollection := mongodb.Collection(models.Collections.VerificationRequest, options.Collection())
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

	mongodb.CreateCollection(ctx, models.Collections.Session, options.CreateCollection())
	sessionCollection := mongodb.Collection(models.Collections.Session, options.Collection())
	sessionCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	mongodb.CreateCollection(ctx, models.Collections.Env, options.CreateCollection())

	return &provider{
		db: mongodb,
	}, nil
}
