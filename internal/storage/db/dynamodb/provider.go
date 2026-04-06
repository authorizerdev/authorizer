package dynamodb

import (
	"context"
	"fmt"
	"time"

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

	awsCfg := aws.Config{
		MaxRetries:                    aws.Int(3),
		CredentialsChainVerboseErrors: aws.Bool(true), // for full error logs
	}

	if awsRegion != "" {
		awsCfg.Region = aws.String(awsRegion)
	}
	// custom awsAccessKeyID, awsSecretAccessKey took first priority, if not then fetch config from aws credentials
	if awsAccessKeyID != "" && awsSecretAccessKey != "" {
		awsCfg.Credentials = credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")
	} else if dbURL != "" {
		deps.Log.Info().Msg("Using DB URL for dynamodb")
		// static config in case of testing or local-setup
		awsCfg.Credentials = credentials.NewStaticCredentials("key", "key", "")
		awsCfg.Endpoint = aws.String(dbURL)
	} else {
		deps.Log.Info().Msg("Using default AWS credentials config from system for dynamodb")
	}
	sess, err := session.NewSession(&awsCfg)
	if err != nil {
		return nil, fmt.Errorf("dynamodb session: %w", err)
	}
	db := dynamo.New(sess)

	createCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tables := []struct {
		name  string
		model interface{}
	}{
		{schemas.Collections.User, schemas.User{}},
		{schemas.Collections.Session, schemas.Session{}},
		{schemas.Collections.EmailTemplate, schemas.EmailTemplate{}},
		{schemas.Collections.Env, schemas.Env{}},
		{schemas.Collections.OTP, schemas.OTP{}},
		{schemas.Collections.VerificationRequest, schemas.VerificationRequest{}},
		{schemas.Collections.Webhook, schemas.Webhook{}},
		{schemas.Collections.WebhookLog, schemas.WebhookLog{}},
		{schemas.Collections.Authenticators, schemas.Authenticator{}},
		{schemas.Collections.SessionToken, schemas.SessionToken{}},
		{schemas.Collections.MFASession, schemas.MFASession{}},
		{schemas.Collections.OAuthState, schemas.OAuthState{}},
		{schemas.Collections.AuditLog, schemas.AuditLog{}},
	}
	for _, tbl := range tables {
		if werr := db.CreateTable(tbl.name, tbl.model).WaitWithContext(createCtx); werr != nil {
			return nil, fmt.Errorf("dynamodb create/wait table %q: %w", tbl.name, werr)
		}
	}
	return &provider{
		db:           db,
		config:       cfg,
		dependencies: deps,
	}, nil
}

// Close is a no-op; the AWS SDK session needs no explicit shutdown for typical use.
func (p *provider) Close() error {
	return nil
}
