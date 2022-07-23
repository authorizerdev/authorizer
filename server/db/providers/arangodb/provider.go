package arangodb

import (
	"context"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
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
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{dbURL},
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
	dbName := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseName
	arangodb_exists, err := arangoClient.DatabaseExists(nil, dbName)

	if arangodb_exists {
		arangodb, err = arangoClient.Database(nil, dbName)
		if err != nil {
			return nil, err
		}
	} else {
		arangodb, err = arangoClient.CreateDatabase(nil, dbName, nil)
		if err != nil {
			return nil, err
		}
	}

	userCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.User)
	if !userCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.User, nil)
		if err != nil {
			return nil, err
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
	if !verificationRequestCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.VerificationRequest, nil)
		if err != nil {
			return nil, err
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
	if !sessionCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Session, nil)
		if err != nil {
			return nil, err
		}
	}

	sessionCollection, _ := arangodb.Collection(nil, models.Collections.Session)
	sessionCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	configCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Env)
	if !configCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Env, nil)
		if err != nil {
			return nil, err
		}
	}

	webhookCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Webhook)
	if !webhookCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Webhook, nil)
		if err != nil {
			return nil, err
		}
	}

	webhookCollection, _ := arangodb.Collection(nil, models.Collections.Webhook)
	webhookCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	webhookLogCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.WebhookLog)
	if !webhookLogCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.WebhookLog, nil)
		if err != nil {
			return nil, err
		}
	}

	webhookLogCollection, _ := arangodb.Collection(nil, models.Collections.WebhookLog)
	webhookLogCollection.EnsureHashIndex(ctx, []string{"webhook_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	emailTemplateCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.EmailTemplate)
	if !emailTemplateCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.EmailTemplate, nil)
		if err != nil {
			return nil, err
		}
	}

	emailTemplateCollection, _ := arangodb.Collection(nil, models.Collections.EmailTemplate)
	emailTemplateCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	otpCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.OTP)
	if !otpCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.OTP, nil)
		if err != nil {
			return nil, err
		}
	}

	otpCollection, _ := arangodb.Collection(nil, models.Collections.OTP)
	otpCollection.EnsureHashIndex(ctx, []string{"email"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	return &provider{
		db: arangodb,
	}, err
}
