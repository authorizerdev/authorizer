package arangodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

// Dependencies struct the arangodb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       config.Config
	dependencies Dependencies
	db           arangoDriver.Database
}

// for this we need arangodb instance up and running
// for local testing we can use dockerized version of it
// docker run -p 8529:8529 -e ARANGO_ROOT_PASSWORD=root arangodb/arangodb:3.8.4

// NewProvider to initialize arangodb connection
func NewProvider(cfg config.Config, deps Dependencies) (*provider, error) {
	ctx := context.Background()
	dbURL := cfg.DatabaseURL
	dbUsername := cfg.DatabaseUsername
	dbPassword := cfg.DatabasePassword
	dbCACertificate := cfg.DatabaseCACert
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
	dbName := cfg.DatabaseName
	if dbName == "" {
		return nil, fmt.Errorf("database name is required")
	}
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
	userCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.User)
	if err != nil {
		return nil, err
	}
	if !userCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.User, nil)
		if err != nil {
			return nil, err
		}
	}
	userCollection, err := arangodb.Collection(ctx, schemas.Collections.User)
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

	verificationRequestCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.VerificationRequest)
	if err != nil {
		return nil, err
	}
	if !verificationRequestCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.VerificationRequest, nil)
		if err != nil {
			return nil, err
		}
	}
	verificationRequestCollection, err := arangodb.Collection(ctx, schemas.Collections.VerificationRequest)
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

	sessionCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.Session)
	if err != nil {
		return nil, err
	}
	if !sessionCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.Session, nil)
		if err != nil {
			return nil, err
		}
	}
	sessionCollection, err := arangodb.Collection(ctx, schemas.Collections.Session)
	if err != nil {
		return nil, err
	}
	sessionCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})
	envCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.Env)
	if err != nil {
		return nil, err
	}
	if !envCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.Env, nil)
		if err != nil {
			return nil, err
		}
	}
	webhookCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.Webhook)
	if err != nil {
		return nil, err
	}
	if !webhookCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.Webhook, nil)
		if err != nil {
			return nil, err
		}
	}
	webhookCollection, err := arangodb.Collection(ctx, schemas.Collections.Webhook)
	if err != nil {
		return nil, err
	}
	webhookCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	webhookLogCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.WebhookLog)
	if err != nil {
		return nil, err
	}
	if !webhookLogCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.WebhookLog, nil)
		if err != nil {
			return nil, err
		}
	}
	webhookLogCollection, err := arangodb.Collection(ctx, schemas.Collections.WebhookLog)
	if err != nil {
		return nil, err
	}
	webhookLogCollection.EnsureHashIndex(ctx, []string{"webhook_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	emailTemplateCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.EmailTemplate)
	if err != nil {
		return nil, err
	}
	if !emailTemplateCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.EmailTemplate, nil)
		if err != nil {
			return nil, err
		}
	}
	emailTemplateCollection, err := arangodb.Collection(ctx, schemas.Collections.EmailTemplate)
	if err != nil {
		return nil, err
	}
	emailTemplateCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	otpCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.OTP)
	if err != nil {
		return nil, err
	}
	if !otpCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.OTP, nil)
		if err != nil {
			return nil, err
		}
	}
	otpCollection, err := arangodb.Collection(ctx, schemas.Collections.OTP)
	if err != nil {
		return nil, err
	}
	otpCollection.EnsureHashIndex(ctx, []string{schemas.FieldNameEmail, schemas.FieldNamePhoneNumber}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	//authenticators table define
	authenticatorsCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.Authenticators)
	if err != nil {
		return nil, err
	}
	if !authenticatorsCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.Authenticators, nil)
		if err != nil {
			return nil, err
		}
	}
	authenticatorsCollection, err := arangodb.Collection(ctx, schemas.Collections.Authenticators)
	if err != nil {
		return nil, err
	}
	authenticatorsCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	return &provider{
		config:       cfg,
		dependencies: deps,
		db:           arangodb,
	}, err
}
