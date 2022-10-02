package dynamodb

import (
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

// TODO change following provider to new db provider
type provider struct {
	db *dynamo.DB
}

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
func NewProvider() (*provider, error) {
	config := aws.Config{
		Endpoint: aws.String("http://localhost:8000"),
		Region:   aws.String("us-east-1"),
	}
	session := session.Must(session.NewSession())
	db := dynamo.New(session, &config)

	if err := db.CreateTable(models.Collections.User, models.User{}).Wait(); err != nil {
		// fmt.Println(" User", err)
	}
	if err := db.CreateTable(models.Collections.Session, models.Session{}).Wait(); err != nil {
		// fmt.Println("Session error", err)
	}
	if err := db.CreateTable(models.Collections.EmailTemplate, models.EmailTemplate{}).Wait(); err != nil {
		// fmt.Println(" EmailTemplate", err)
	}
	if err := db.CreateTable(models.Collections.Env, models.Env{}).Wait(); err != nil {
		// fmt.Println(" Env", err)
	}
	if err := db.CreateTable(models.Collections.OTP, models.OTP{}).Wait(); err != nil {
		// fmt.Println(" OTP", err)
	}
	if err := db.CreateTable(models.Collections.VerificationRequest, models.VerificationRequest{}).Wait(); err != nil {
		// fmt.Println(" VerificationRequest", err)
	}
	if err := db.CreateTable(models.Collections.Webhook, models.Webhook{}).Wait(); err != nil {
		// fmt.Println(" Webhook", err)
	}
	if err := db.CreateTable(models.Collections.WebhookLog, models.WebhookLog{}).Wait(); err != nil {
		// fmt.Println(" WebhookLog", err)
	}

	return &provider{
		db: db,
	}, nil
}
