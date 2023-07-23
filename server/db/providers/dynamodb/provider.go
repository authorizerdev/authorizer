package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

type provider struct {
	db *dynamo.DB
}

// NewProvider returns a new Dynamo provider
func NewProvider() (*provider, error) {
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

	db.CreateTable(models.Collections.User, models.User{}).Wait()
	db.CreateTable(models.Collections.Session, models.Session{}).Wait()
	db.CreateTable(models.Collections.EmailTemplate, models.EmailTemplate{}).Wait()
	db.CreateTable(models.Collections.Env, models.Env{}).Wait()
	db.CreateTable(models.Collections.OTP, models.OTP{}).Wait()
	db.CreateTable(models.Collections.VerificationRequest, models.VerificationRequest{}).Wait()
	db.CreateTable(models.Collections.Webhook, models.Webhook{}).Wait()
	db.CreateTable(models.Collections.WebhookLog, models.WebhookLog{}).Wait()

	return &provider{
		db: db,
	}, nil
}
