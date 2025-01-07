package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/env"
	"github.com/authorizerdev/authorizer/internal/handlers"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/middlewares"
	"github.com/gin-gonic/gin"
)

// common user data to share across tests
type TestData struct {
	Email                       string
	Password                    string
	WebhookEndpoint             string
	TestWebhookEventTypes       []string
	TestEmailTemplateEventTypes []string
}

type TestSetup struct {
	GinEngine  *gin.Engine
	GinContext *gin.Context
	Server     *httptest.Server
	TestInfo   TestData
}

func cleanData(email string) {
	ctx := context.Background()
	verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug("DeleteVerificationRequest err", err)
		}
	}

	verificationRequest, err = db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug("DeleteVerificationRequest err", err)
		}
	}

	verificationRequest, err = db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeUpdateEmail)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug("DeleteVerificationRequest err", err)
		}
	}

	verificationRequest, err = db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug("DeleteVerificationRequest err", err)
		}
	}

	otp, err := db.Provider.GetOTPByEmail(ctx, email)
	if err == nil {
		err = db.Provider.DeleteOTP(ctx, otp)
		if err != nil {
			log.Debug("DeleteOTP err", err)
		}
	}

	dbUser, err := db.Provider.GetUserByEmail(ctx, email)
	if err == nil {
		err = db.Provider.DeleteUser(ctx, dbUser)
		if err != nil {
			log.Debug("DeleteUser err", err)
		}
	}
}

func createContext(s TestSetup) (*http.Request, context.Context) {
	req, _ := http.NewRequest(
		"POST",
		"http://"+s.Server.Listener.Addr().String()+"/graphql",
		nil,
	)

	ctx := context.WithValue(req.Context(), "GinContextKey", s.GinContext)
	s.GinContext.Request = req
	return req, ctx
}

func testSetup() TestSetup {
	testData := TestData{
		Email:                       fmt.Sprintf("%d_authorizer_tester@yopmail.com", time.Now().Unix()),
		Password:                    "Test@123",
		WebhookEndpoint:             "https://62f93101e05644803533cf36.mockapi.io/authorizer/webhook",
		TestWebhookEventTypes:       []string{constants.UserAccessEnabledWebhookEvent, constants.UserAccessRevokedWebhookEvent, constants.UserCreatedWebhookEvent, constants.UserDeletedWebhookEvent, constants.UserLoginWebhookEvent, constants.UserSignUpWebhookEvent, constants.UserDeactivatedWebhookEvent},
		TestEmailTemplateEventTypes: []string{constants.VerificationTypeBasicAuthSignup, constants.VerificationTypeForgotPassword, constants.VerificationTypeMagicLinkLogin, constants.VerificationTypeUpdateEmail},
	}

	err := os.Setenv(constants.EnvKeyEnvPath, "../../.env.test")
	if err != nil {
		log.Fatal("Error loading .env.sample file")
	}
	err = memorystore.InitRequiredEnv()
	if err != nil {
		log.Fatal("Error loading required env: ", err)
	}

	err = memorystore.InitMemStore()
	if err != nil {
		log.Fatal("Error loading memory store: ", err)
	}
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeySmtpHost, "smtp.yopmail.com")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeySmtpPort, "2525")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeySmtpUsername, "lakhan@yopmail.com")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeySmtpPassword, "test")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeySenderEmail, "info@yopmail.com")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyProtectedRoles, "admin")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyTwilioAPIKey, "test")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyTwilioAPISecret, "test")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyTwilioAccountSID, "test")
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyTwilioSender, "1234567890")

	err = db.InitDB()
	if err != nil {
		log.Fatal("Error loading db: ", err)
	}

	err = env.InitAllEnv()
	if err != nil {
		log.Fatal("Error loading env: ", err)
	}

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.Use(middlewares.GinContextToContextMiddleware())
	r.Use(middlewares.CORSMiddleware())

	r.POST("/graphql", handlers.GraphqlHandler())

	server := httptest.NewServer(r)

	return TestSetup{
		GinEngine:  r,
		GinContext: c,
		Server:     server,
		TestInfo:   testData,
	}
}
