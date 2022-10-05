package dynamodb

import (
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

type provider struct {
	db *dynamo.DB
}

// NewProvider returns a new Dynamo provider
func NewProvider() (*provider, error) {
	region := memorystore.RequiredEnvStoreObj.GetRequiredEnv().REGION
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	accessKey := memorystore.RequiredEnvStoreObj.GetRequiredEnv().AWS_ACCESS_KEY
	secretKey := memorystore.RequiredEnvStoreObj.GetRequiredEnv().AWS_SECRET_KEY
	config := aws.Config{
		Region:     aws.String(region),
		MaxRetries: aws.Int(3),
	}

	// custom accessKey, secretkey took first priority, if not then fetch config from aws credentials
	if accessKey != "" && secretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	} else if dbURL != "" {
		// static config in case of testing or local-setup
		config.Credentials = credentials.NewStaticCredentials("key", "key", "")
		config.Endpoint = aws.String(dbURL)
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
