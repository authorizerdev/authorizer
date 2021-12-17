package db

import (
	"context"
	"log"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/authorizerdev/authorizer/server/constants"
)

// for this we need arangodb instance up and running
// for local testing we can use dockerized version of it
// docker run -p 8529:8529 -e ARANGO_ROOT_PASSWORD=root arangodb/arangodb:3.8.4

func initArangodb() (arangoDriver.Database, error) {
	ctx := context.Background()
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{constants.DATABASE_URL},
	})
	if err != nil {
		return nil, err
	}

	client, err := arangoDriver.NewClient(arangoDriver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		return nil, err
	}

	var arangodb driver.Database

	arangodb_exists, err := client.DatabaseExists(nil, constants.DATABASE_NAME)

	if arangodb_exists {
		log.Println(constants.DATABASE_NAME + " db exists already")

		arangodb, err = client.Database(nil, constants.DATABASE_NAME)

		if err != nil {
			return nil, err
		}

	} else {
		arangodb, err = client.CreateDatabase(nil, constants.DATABASE_NAME, nil)

		if err != nil {
			return nil, err
		}
	}

	userCollectionExists, err := arangodb.CollectionExists(ctx, Collections.User)
	if userCollectionExists {
		log.Println(Collections.User + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, Collections.User, nil)
		if err != nil {
			log.Println("error creating collection("+Collections.User+"):", err)
		}
	}
	userCollection, _ := arangodb.Collection(nil, Collections.User)
	userCollection.EnsureHashIndex(ctx, []string{"id"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	userCollection.EnsureHashIndex(ctx, []string{"email"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	verificationRequestCollectionExists, err := arangodb.CollectionExists(ctx, Collections.VerificationRequest)
	if verificationRequestCollectionExists {
		log.Println(Collections.VerificationRequest + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, Collections.VerificationRequest, nil)
		if err != nil {
			log.Println("error creating collection("+Collections.VerificationRequest+"):", err)
		}
	}
	verificationRequestCollection, _ := arangodb.Collection(nil, Collections.VerificationRequest)
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"id"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"email", "identifier"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"token"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	sessionCollectionExists, err := arangodb.CollectionExists(ctx, Collections.Session)
	if sessionCollectionExists {
		log.Println(Collections.Session + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, Collections.Session, nil)
		if err != nil {
			log.Println("error creating collection("+Collections.Session+"):", err)
		}
	}

	sessionCollection, _ := arangodb.Collection(nil, Collections.Session)
	sessionCollection.EnsureHashIndex(ctx, []string{"id"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	return arangodb, err
}
