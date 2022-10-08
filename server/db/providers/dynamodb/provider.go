package dynamodb

import (
	"os"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	log "github.com/sirupsen/logrus"
)

type provider struct {
	db *dynamo.DB
}

// NewProvider returns a new Dynamo provider
func NewProvider() (*provider, error) {
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	awsRegion := os.Getenv(constants.EnvAwsRegion)
	accessKey := os.Getenv(constants.EnvAwsAccessKey)
	secretKey := os.Getenv(constants.EnvAwsSecretKey)

	config := aws.Config{
		MaxRetries:                    aws.Int(3),
		CredentialsChainVerboseErrors: aws.Bool(true), // for full error logs
	}

	if awsRegion != "" {
		config.Region = aws.String(awsRegion)
	}

	// custom accessKey, secretkey took first priority, if not then fetch config from aws credentials
	if accessKey != "" && secretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	} else if dbURL != "" {
		// static config in case of testing or local-setup
		config.Credentials = credentials.NewStaticCredentials("key", "key", "")
		config.Endpoint = aws.String(dbURL)
	} else {
		log.Info("REGION, AWS_ACCESS_KEY and AWS_SECRET_KEY not found in .env, trying to load default profile from aws credentials")
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
