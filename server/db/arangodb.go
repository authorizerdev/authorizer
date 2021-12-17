package db

import (
	"context"
	"log"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/authorizerdev/authorizer/server/constants"
)

func initArangodb() (*arangoDriver.Database, error) {
	ctx := context.Background()
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{constants.DATABASE_URL},
	})
	if err != nil {
		return nil, err
	}

	// TODO add support for authentication option in clientConfig or check if
	// basic auth pattern works here in DB_URL
	client, err := arangoDriver.NewClient(arangoDriver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		return nil, err
	}

	var arangodb driver.Database
	var arangodb_exists bool

	// TODO use dynamic name based on env
	dbName := "authorizer"
	arangodb_exists, err = client.DatabaseExists(nil, dbName)

	if arangodb_exists {
		log.Println(dbName + " db exists already")

		arangodb, err = client.Database(nil, dbName)

		if err != nil {
			return nil, err
		}

	} else {
		arangodb, err = client.CreateDatabase(nil, dbName, nil)

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

	verificationRequestsColumnExists, err := arangodb.CollectionExists(ctx, Collections.VerificationRequest)
	if verificationRequestsColumnExists {
		log.Println(Collections.VerificationRequest + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, Collections.VerificationRequest, nil)
		if err != nil {
			log.Println("error creating collection("+Collections.VerificationRequest+"):", err)
		}
	}

	rolesExists, err := arangodb.CollectionExists(ctx, Collections.Role)
	if rolesExists {
		log.Println(Collections.Role + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, Collections.Role, nil)
		if err != nil {
			log.Println("error creating collection("+Collections.Role+"):", err)
		}
	}

	sessionExists, err := arangodb.CollectionExists(ctx, Collections.Session)
	if sessionExists {
		log.Println(Collections.Session + " collection exists already")
	} else {
		_, err = arangodb.CreateCollection(ctx, Collections.Session, nil)
		if err != nil {
			log.Println("error creating collection("+Collections.Session+"):", err)
		}
	}

	return &arangodb, err
}
