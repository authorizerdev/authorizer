package arangodb

import (
	"context"
	"log"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
)

type provider struct {
	db arangoDriver.Database
}

// for this we need arangodb instance up and running
// for local testing we can use dockerized version of it
// docker run -p 8529:8529 -e ARANGO_ROOT_PASSWORD=root arangodb/arangodb:3.8.4

// NewProvider to initialize arangodb connection
func NewProvider() (*provider, error) {
	ctx := context.Background()
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)},
	})
	if err != nil {
		return nil, err
	}

	arangoClient, err := arangoDriver.NewClient(arangoDriver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		return nil, err
	}

	var arangodb driver.Database

	arangodb_exists, err := arangoClient.DatabaseExists(nil, envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName))

	if arangodb_exists {
		log.Println(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName) + " db exists already")
		arangodb, err = arangoClient.Database(nil, envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName))
		if err != nil {
			return nil, err
		}
	} else {
		arangodb, err = arangoClient.CreateDatabase(nil, envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName), nil)
		if err != nil {
			return nil, err
		}
	}

	userCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.User)
	if userCollectionExists {
		log.Println(models.Collections.User + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, models.Collections.User, nil)
		if err != nil {
			log.Println("error creating collection("+models.Collections.User+"):", err)
		}
	}
	userCollection, _ := arangodb.Collection(nil, models.Collections.User)
	userCollection.EnsureHashIndex(ctx, []string{"email"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	userCollection.EnsureHashIndex(ctx, []string{"phone_number"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	verificationRequestCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.VerificationRequest)
	if verificationRequestCollectionExists {
		log.Println(models.Collections.VerificationRequest + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, models.Collections.VerificationRequest, nil)
		if err != nil {
			log.Println("error creating collection("+models.Collections.VerificationRequest+"):", err)
		}
	}

	verificationRequestCollection, _ := arangodb.Collection(nil, models.Collections.VerificationRequest)
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"email", "identifier"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"token"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	sessionCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Session)
	if sessionCollectionExists {
		log.Println(models.Collections.Session + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Session, nil)
		if err != nil {
			log.Println("error creating collection("+models.Collections.Session+"):", err)
		}
	}

	sessionCollection, _ := arangodb.Collection(nil, models.Collections.Session)
	sessionCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	configCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Env)
	if configCollectionExists {
		log.Println(models.Collections.Env + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Env, nil)
		if err != nil {
			log.Println("error creating collection("+models.Collections.Env+"):", err)
		}
	}

	return &provider{
		db: arangodb,
	}, err
}
