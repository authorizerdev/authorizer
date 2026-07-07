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
	_, _ = verificationRequestCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"email": 1, "identifier": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}, options.CreateIndexes())
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
			Keys:    bson.M{"user_id": 1, "key_name": 1},
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
			Keys:    bson.M{"user_id": 1, "key_name": 1},
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

	// ServiceAccount collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.ServiceAccount, options.CreateCollection())

	// TrustedIssuer collection and indexes
	_ = mongodb.CreateCollection(ctx, schemas.Collections.TrustedIssuer, options.CreateCollection())
	trustedIssuerCollection := mongodb.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	_, _ = trustedIssuerCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"issuer_url": 1},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.M{"service_account_id": 1},
			Options: options.Index().SetSparse(true),
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
