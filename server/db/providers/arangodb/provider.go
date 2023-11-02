package arangodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"

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
	dbUsername := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseUsername
	dbPassword := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabasePassword
	dbCACertificate := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseCACert
	httpConfig := http.ConnectionConfig{
		Endpoints: []string{dbURL},
	}
	// If ca certificate if present, create tls config
	if dbCACertificate != "" {
		caCert, err := base64.StdEncoding.DecodeString(dbCACertificate)
		if err != nil {
			return nil, err
		}
		// Prepare TLS Config
		tlsConfig := &tls.Config{}
		certPool := x509.NewCertPool()
		if success := certPool.AppendCertsFromPEM(caCert); !success {
			return nil, fmt.Errorf("invalid certificate")
		}
		tlsConfig.RootCAs = certPool
		httpConfig.TLSConfig = tlsConfig
	}
	// Create new http connection
	conn, err := http.NewConnection(httpConfig)
	if err != nil {
		return nil, err
	}
	clientConfig := arangoDriver.ClientConfig{
		Connection: conn,
	}
	if dbUsername != "" && dbPassword != "" {
		clientConfig.Authentication = arangoDriver.BasicAuthentication(dbUsername, dbPassword)
	}
	arangoClient, err := arangoDriver.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}
	var arangodb arangoDriver.Database
	dbName := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseName
	arangodb_exists, err := arangoClient.DatabaseExists(ctx, dbName)
	if err != nil {
		return nil, err
	}
	if arangodb_exists {
		arangodb, err = arangoClient.Database(ctx, dbName)
		if err != nil {
			return nil, err
		}
	} else {
		arangodb, err = arangoClient.CreateDatabase(ctx, dbName, nil)
		if err != nil {
			return nil, err
		}
	}
	userCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.User)
	if err != nil {
		return nil, err
	}
	if !userCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.User, nil)
		if err != nil {
			return nil, err
		}
	}
	userCollection, err := arangodb.Collection(ctx, models.Collections.User)
	if err != nil {
		return nil, err
	}
	userCollection.EnsureHashIndex(ctx, []string{"email"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	userCollection.EnsureHashIndex(ctx, []string{"phone_number"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	verificationRequestCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.VerificationRequest)
	if err != nil {
		return nil, err
	}
	if !verificationRequestCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.VerificationRequest, nil)
		if err != nil {
			return nil, err
		}
	}
	verificationRequestCollection, err := arangodb.Collection(ctx, models.Collections.VerificationRequest)
	if err != nil {
		return nil, err
	}
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"email", "identifier"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	verificationRequestCollection.EnsureHashIndex(ctx, []string{"token"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	sessionCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Session)
	if err != nil {
		return nil, err
	}
	if !sessionCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Session, nil)
		if err != nil {
			return nil, err
		}
	}
	sessionCollection, err := arangodb.Collection(ctx, models.Collections.Session)
	if err != nil {
		return nil, err
	}
	sessionCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})
	envCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Env)
	if err != nil {
		return nil, err
	}
	if !envCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Env, nil)
		if err != nil {
			return nil, err
		}
	}
	webhookCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Webhook)
	if err != nil {
		return nil, err
	}
	if !webhookCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Webhook, nil)
		if err != nil {
			return nil, err
		}
	}
	webhookCollection, err := arangodb.Collection(ctx, models.Collections.Webhook)
	if err != nil {
		return nil, err
	}
	webhookCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	webhookLogCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.WebhookLog)
	if err != nil {
		return nil, err
	}
	if !webhookLogCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.WebhookLog, nil)
		if err != nil {
			return nil, err
		}
	}
	webhookLogCollection, err := arangodb.Collection(ctx, models.Collections.WebhookLog)
	if err != nil {
		return nil, err
	}
	webhookLogCollection.EnsureHashIndex(ctx, []string{"webhook_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	emailTemplateCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.EmailTemplate)
	if err != nil {
		return nil, err
	}
	if !emailTemplateCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.EmailTemplate, nil)
		if err != nil {
			return nil, err
		}
	}
	emailTemplateCollection, err := arangodb.Collection(ctx, models.Collections.EmailTemplate)
	if err != nil {
		return nil, err
	}
	emailTemplateCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	otpCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.OTP)
	if err != nil {
		return nil, err
	}
	if !otpCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.OTP, nil)
		if err != nil {
			return nil, err
		}
	}
	otpCollection, err := arangodb.Collection(ctx, models.Collections.OTP)
	if err != nil {
		return nil, err
	}
	otpCollection.EnsureHashIndex(ctx, []string{models.FieldNameEmail, models.FieldNamePhoneNumber}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	//authenticators table define
	authenticatorsCollectionExists, err := arangodb.CollectionExists(ctx, models.Collections.Authenticators)
	if err != nil {
		return nil, err
	}
	if !authenticatorsCollectionExists {
		_, err = arangodb.CreateCollection(ctx, models.Collections.Authenticators, nil)
		if err != nil {
			return nil, err
		}
	}
	authenticatorsCollection, err := arangodb.Collection(ctx, models.Collections.Authenticators)
	if err != nil {
		return nil, err
	}
	authenticatorsCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	return &provider{
		db: arangodb,
	}, err
}
