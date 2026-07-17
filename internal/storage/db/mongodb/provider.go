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
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the mongodb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *mongo.Database
}

// NewProvider to initialize mongodb connection
func NewProvider(config *config.Config, deps *Dependencies) (*provider, error) {
	dbURL := config.DatabaseURL
	mongodbOptions := options.Client().ApplyURI(dbURL)
	maxWait := time.Duration(5 * time.Second)
	mongodbOptions.ConnectTimeout = &maxWait
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
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

	_ = mongodb.CreateCollection(ctx, schemas.Collections.User, options.CreateCollection())
	userCollection := mongodb.Collection(schemas.Collections.User, options.Collection())
	_, _ = userCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
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
	_ = mongodb.CreateCollection(ctx, schemas.Collections.VerificationRequest, options.CreateCollection())
	verificationRequestCollection := mongodb.Collection(schemas.Collections.VerificationRequest, options.Collection())
	if _, err := verificationRequestCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Compound index keys MUST use the ordered bson.D — the driver
			// rejects a multi-key bson.M ("multi-key map passed in for ordered
			// parameter keys"), so this unique constraint was silently never
			// created; same bug already found and fixed for the authenticator
			// index below.
			Keys:    bson.D{{Key: "email", Value: 1}, {Key: "identifier", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes()); err != nil {
		// Unlike the rest of this file's index creation, this one is worth a
		// loud warning rather than a silent discard: a database that already
		// accumulated duplicate (email, identifier) rows from the bson.M bug
		// this replaces will fail this unique-index build and stay
		// unprotected until an operator dedupes and retries - swallowing the
		// error would hide that a fresh install still needs attention.
		if deps != nil && deps.Log != nil {
			deps.Log.Warn().Err(err).Msg("failed to create unique index on verification_requests(email, identifier) - if this database has pre-existing duplicates, dedupe them and restart")
		}
	}
	_, _ = verificationRequestCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"token": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.Session, options.CreateCollection())
	sessionCollection := mongodb.Collection(schemas.Collections.Session, options.Collection())
	_, _ = sessionCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.Env, options.CreateCollection())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.Webhook, options.CreateCollection())
	webhookCollection := mongodb.Collection(schemas.Collections.Webhook, options.Collection())
	_, _ = webhookCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"event_name": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.WebhookLog, options.CreateCollection())
	webhookLogCollection := mongodb.Collection(schemas.Collections.WebhookLog, options.Collection())
	_, _ = webhookLogCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"webhook_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.EmailTemplate, options.CreateCollection())
	emailTemplateCollection := mongodb.Collection(schemas.Collections.EmailTemplate, options.Collection())
	_, _ = emailTemplateCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"event_name": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.OTP, options.CreateCollection())
	otpCollection := mongodb.Collection(schemas.Collections.OTP, options.Collection())
	_, _ = otpCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"email": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())
	_, _ = otpCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"phone_number": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	_ = mongodb.CreateCollection(ctx, schemas.Collections.Authenticators, options.CreateCollection())
	authenticatorsCollection := mongodb.Collection(schemas.Collections.Authenticators, options.Collection())
	_, _ = authenticatorsCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Unique per (user_id, method) — prevents the check-then-insert race
			// in AddAuthenticator from creating duplicate MFA enrollments.
			// Also serves user_id-prefix lookups, so no separate user_id index.
			// Compound index keys MUST use the ordered bson.D — the driver rejects
			// a multi-key bson.M ("multi-key map passed in for ordered parameter keys").
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "method", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	// SessionToken collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.SessionToken, options.CreateCollection())
	sessionTokenCollection := mongodb.Collection(schemas.Collections.SessionToken, options.Collection())
	_, _ = sessionTokenCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// bson.D, not bson.M - see the verification-request index comment above.
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "key_name", Value: 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.M{"expires_at": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// MFASession collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.MFASession, options.CreateCollection())
	mfaSessionCollection := mongodb.Collection(schemas.Collections.MFASession, options.Collection())
	_, _ = mfaSessionCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// bson.D, not bson.M - see the verification-request index comment above.
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "key_name", Value: 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.M{"expires_at": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// OAuthState collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.OAuthState, options.CreateCollection())
	oauthStateCollection := mongodb.Collection(schemas.Collections.OAuthState, options.Collection())
	_, _ = oauthStateCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"state_key": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	// AuditLog collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.AuditLog, options.CreateCollection())
	auditLogCollection := mongodb.Collection(schemas.Collections.AuditLog, options.Collection())
	_, _ = auditLogCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"actor_id": 1},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.M{"action": 1},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.M{"timestamp": -1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// Client collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.Client, options.CreateCollection())
	clientCollection := mongodb.Collection(schemas.Collections.Client, options.Collection())
	_, _ = clientCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"client_id": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.M{"org_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// TrustedIssuer collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.TrustedIssuer, options.CreateCollection())
	trustedIssuerCollection := mongodb.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	_, _ = trustedIssuerCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"issuer_url": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.M{"client_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// WebauthnCredential collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.WebauthnCredential, options.CreateCollection())
	webauthnCredentialCollection := mongodb.Collection(schemas.Collections.WebauthnCredential, options.Collection())
	_, _ = webauthnCredentialCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"credential_id": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// Organization collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.Organization, options.CreateCollection())
	organizationCollection := mongodb.Collection(schemas.Collections.Organization, options.Collection())
	_, _ = organizationCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"name": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())

	// OrgMembership collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.OrgMembership, options.CreateCollection())
	orgMembershipCollection := mongodb.Collection(schemas.Collections.OrgMembership, options.Collection())
	_, _ = orgMembershipCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// FederatedIdentity collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.FederatedIdentity, options.CreateCollection())
	federatedIdentityCollection := mongodb.Collection(schemas.Collections.FederatedIdentity, options.Collection())
	_, _ = federatedIdentityCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "issuer", Value: 1}, {Key: "subject", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.M{"user_id": 1},
			Options: options.Index().SetSparse(true),
		},
	}, options.CreateIndexes())

	// ScimEndpoint collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.ScimEndpoint, options.CreateCollection())
	scimEndpointCollection := mongodb.Collection(schemas.Collections.ScimEndpoint, options.Collection())
	_, _ = scimEndpointCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"org_id": 1},
			Options: options.Index().SetUnique(true),
		},
	}, options.CreateIndexes())

	// OrgDomain collection and indexes. Uniqueness is enforced by _id being the
	// normalized domain (no unique index needed); org_id is indexed for listing.
	_ = mongodb.CreateCollection(ctx, schemas.Collections.OrgDomain, options.CreateCollection())
	orgDomainCollection := mongodb.Collection(schemas.Collections.OrgDomain, options.Collection())
	_, _ = orgDomainCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.M{"org_id": 1},
		},
	}, options.CreateIndexes())

	return &provider{
		config:       config,
		dependencies: deps,
		db:           mongodb,
	}, nil
}

// Close disconnects the MongoDB client.
func (p *provider) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return p.db.Client().Disconnect(ctx)
}
