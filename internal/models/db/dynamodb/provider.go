package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/models/config"
	"github.com/authorizerdev/authorizer/internal/models/schemas"
)

type provider struct {
	Config config.Config
	Deps   config.Dependencies
	db     *dynamo.DB
}

// NewProvider returns a new Dynamo provider
func NewProvider(cfg config.Config, deps config.Dependencies) (*provider, error) {
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	awsRegion := memorystore.RequiredEnvStoreObj.GetRequiredEnv().AwsRegion
	awsAccessKeyID := memorystore.RequiredEnvStoreObj.GetRequiredEnv().AwsAccessKeyID
	awsSecretAccessKey := memorystore.RequiredEnvStoreObj.GetRequiredEnv().AwsSecretAccessKey

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
		log.Debug("Tring to use database url for dynamodb")
		// static config in case of testing or local-setup
		config.Credentials = credentials.NewStaticCredentials("key", "key", "")
		config.Endpoint = aws.String(dbURL)
	} else {
		log.Debugf("%s or %s or %s not found. Trying to load default credentials from aws config", constants.EnvAwsRegion, constants.EnvAwsAccessKeyID, constants.EnvAwsSecretAccessKey)
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
	return &provider{
		db: db,
	}, nil
}
