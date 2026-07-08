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
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the arangodb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           arangoDriver.Database
}

// for this we need arangodb instance up and running
// for local testing we can use dockerized version of it
// docker run -p 8529:8529 -e ARANGO_ROOT_PASSWORD=root arangodb/arangodb:3.8.4

// NewProvider to initialize arangodb connection
func NewProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
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
	_, _, _ = userCollection.EnsureHashIndex(ctx, []string{"email"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	_, _, _ = userCollection.EnsureHashIndex(ctx, []string{"phone_number"}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = verificationRequestCollection.EnsureHashIndex(ctx, []string{"email", "identifier"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	_, _, _ = verificationRequestCollection.EnsureHashIndex(ctx, []string{"token"}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = sessionCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = webhookCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = webhookLogCollection.EnsureHashIndex(ctx, []string{"webhook_id"}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = emailTemplateCollection.EnsureHashIndex(ctx, []string{"event_name"}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = otpCollection.EnsureHashIndex(ctx, []string{schemas.FieldNameEmail, schemas.FieldNamePhoneNumber}, &arangoDriver.EnsureHashIndexOptions{
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
	_, _, _ = authenticatorsCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	// SessionToken collection and indexes
	sessionTokenCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.SessionToken)
	if err != nil {
		return nil, err
	}
	if !sessionTokenCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.SessionToken, nil)
		if err != nil {
			return nil, err
		}
	}
	sessionTokenCollection, err := arangodb.Collection(ctx, schemas.Collections.SessionToken)
	if err != nil {
		return nil, err
	}
	_, _, _ = sessionTokenCollection.EnsureHashIndex(ctx, []string{"user_id", "key_name"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})
	_, _, _ = sessionTokenCollection.EnsurePersistentIndex(ctx, []string{"expires_at"}, &arangoDriver.EnsurePersistentIndexOptions{
		Sparse: true,
	})

	// MFASession collection and indexes
	mfaSessionCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.MFASession)
	if err != nil {
		return nil, err
	}
	if !mfaSessionCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.MFASession, nil)
		if err != nil {
			return nil, err
		}
	}
	mfaSessionCollection, err := arangodb.Collection(ctx, schemas.Collections.MFASession)
	if err != nil {
		return nil, err
	}
	_, _, _ = mfaSessionCollection.EnsureHashIndex(ctx, []string{"user_id", "key_name"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})
	_, _, _ = mfaSessionCollection.EnsurePersistentIndex(ctx, []string{"expires_at"}, &arangoDriver.EnsurePersistentIndexOptions{
		Sparse: true,
	})

	// OAuthState collection and indexes
	oauthStateCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.OAuthState)
	if err != nil {
		return nil, err
	}
	if !oauthStateCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.OAuthState, nil)
		if err != nil {
			return nil, err
		}
	}
	oauthStateCollection, err := arangodb.Collection(ctx, schemas.Collections.OAuthState)
	if err != nil {
		return nil, err
	}
	_, _, _ = oauthStateCollection.EnsureHashIndex(ctx, []string{"state_key"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	_, _, _ = authenticatorsCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	// AuditLog collection and indexes
	auditLogCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.AuditLog)
	if err != nil {
		return nil, err
	}
	if !auditLogCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.AuditLog, nil)
		if err != nil {
			return nil, err
		}
	}
	auditLogCollection, err := arangodb.Collection(ctx, schemas.Collections.AuditLog)
	if err != nil {
		return nil, err
	}
	_, _, _ = auditLogCollection.EnsureHashIndex(ctx, []string{"actor_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})
	_, _, _ = auditLogCollection.EnsureHashIndex(ctx, []string{"action"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})
	_, _, _ = auditLogCollection.EnsurePersistentIndex(ctx, []string{"timestamp"}, &arangoDriver.EnsurePersistentIndexOptions{
		Sparse: true,
	})

	// Client collection
	clientCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.Client)
	if err != nil {
		return nil, err
	}
	if !clientCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.Client, nil)
		if err != nil {
			return nil, err
		}
	}
	clientCollection, err := arangodb.Collection(ctx, schemas.Collections.Client)
	if err != nil {
		return nil, err
	}
	_, _, _ = clientCollection.EnsureHashIndex(ctx, []string{"client_id"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	_, _, _ = clientCollection.EnsureHashIndex(ctx, []string{"org_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	// TrustedIssuer collection and indexes
	trustedIssuerCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.TrustedIssuer)
	if err != nil {
		return nil, err
	}
	if !trustedIssuerCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.TrustedIssuer, nil)
		if err != nil {
			return nil, err
		}
	}
	trustedIssuerCollection, err := arangodb.Collection(ctx, schemas.Collections.TrustedIssuer)
	if err != nil {
		return nil, err
	}
	_, _, _ = trustedIssuerCollection.EnsureHashIndex(ctx, []string{"issuer_url"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})
	_, _, _ = trustedIssuerCollection.EnsureHashIndex(ctx, []string{"client_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	// Organization collection and indexes
	organizationCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.Organization)
	if err != nil {
		return nil, err
	}
	if !organizationCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.Organization, nil)
		if err != nil {
			return nil, err
		}
	}
	organizationCollection, err := arangodb.Collection(ctx, schemas.Collections.Organization)
	if err != nil {
		return nil, err
	}
	_, _, _ = organizationCollection.EnsureHashIndex(ctx, []string{"name"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: true,
	})

	// OrgMembership collection and indexes
	orgMembershipCollectionExists, err := arangodb.CollectionExists(ctx, schemas.Collections.OrgMembership)
	if err != nil {
		return nil, err
	}
	if !orgMembershipCollectionExists {
		_, err = arangodb.CreateCollection(ctx, schemas.Collections.OrgMembership, nil)
		if err != nil {
			return nil, err
		}
	}
	orgMembershipCollection, err := arangodb.Collection(ctx, schemas.Collections.OrgMembership)
	if err != nil {
		return nil, err
	}
	_, _, _ = orgMembershipCollection.EnsureHashIndex(ctx, []string{"org_id", "user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Unique: true,
	})
	_, _, _ = orgMembershipCollection.EnsureHashIndex(ctx, []string{"user_id"}, &arangoDriver.EnsureHashIndexOptions{
		Sparse: true,
	})

	return &provider{
		config:       cfg,
		dependencies: deps,
		db:           arangodb,
	}, err
}

// Close releases ArangoDB driver resources. The HTTP driver does not expose a pool close;
// connections are reclaimed when the provider is discarded.
func (p *provider) Close() error {
	return nil
}
