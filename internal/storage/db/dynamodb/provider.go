package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the dynamodb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *dynamo.DB
}

// NewProvider returns a new Dynamo provider
func NewProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	dbURL := cfg.DatabaseURL
	awsRegion := cfg.AWSRegion
	awsAccessKeyID := cfg.AWSAccessKeyID
	awsSecretAccessKey := cfg.AWSSecretAccessKey

	config := aws.Config{
		MaxRetries:                    aws.Int(3),
		CredentialsChainVerboseErrors: aws.Bool(true), // for full error logs
	}

	if awsRegion != "" {
		config.Region = aws.String(awsRegion)
	}
	// custom awsAccessKeyID, awsSecretAccessKey took first priority, if not then fetch config from aws credentials
	if awsAccessKeyID != "" && awsSecretAccessKey != "" {
		config.Credentials = credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")
	} else if dbURL != "" {
		deps.Log.Info().Msg("Using DB URL for dynamodb")
		// static config in case of testing or local-setup
		config.Credentials = credentials.NewStaticCredentials("key", "key", "")
		config.Endpoint = aws.String(dbURL)
	} else {
		deps.Log.Info().Msg("Using default AWS credentials config from system for dynamodb")
	}
	session := session.Must(session.NewSession(&config))
	db := dynamo.New(session)
	db.CreateTable(schemas.Collections.User, schemas.User{}).Wait()
	db.CreateTable(schemas.Collections.Session, schemas.Session{}).Wait()
	db.CreateTable(schemas.Collections.EmailTemplate, schemas.EmailTemplate{}).Wait()
	db.CreateTable(schemas.Collections.Env, schemas.Env{}).Wait()
	db.CreateTable(schemas.Collections.OTP, schemas.OTP{}).Wait()
	db.CreateTable(schemas.Collections.VerificationRequest, schemas.VerificationRequest{}).Wait()
	db.CreateTable(schemas.Collections.Webhook, schemas.Webhook{}).Wait()
	db.CreateTable(schemas.Collections.WebhookLog, schemas.WebhookLog{}).Wait()
	db.CreateTable(schemas.Collections.Authenticators, schemas.Authenticator{}).Wait()
	db.CreateTable(schemas.Collections.SessionToken, schemas.SessionToken{}).Wait()
	db.CreateTable(schemas.Collections.MFASession, schemas.MFASession{}).Wait()
	db.CreateTable(schemas.Collections.OAuthState, schemas.OAuthState{}).Wait()
	return &provider{
		db:           db,
		config:       cfg,
		dependencies: deps,
	}, nil
}
