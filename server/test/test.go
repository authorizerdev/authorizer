package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/middlewares"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/gin-gonic/gin"
)

// common user data to share across tests
type TestData struct {
	Email    string
	Password string
}

type TestSetup struct {
	GinEngine  *gin.Engine
	GinContext *gin.Context
	Server     *httptest.Server
	TestInfo   TestData
}

func cleanData(email string) {
	verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeBasicAuthSignup)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(verificationRequest)
	}

	verificationRequest, err = db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeForgotPassword)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(verificationRequest)
	}

	verificationRequest, err = db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeUpdateEmail)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(verificationRequest)
	}

	verificationRequest, err = db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeMagicLinkLogin)
	if err == nil {
		err = db.Provider.DeleteVerificationRequest(verificationRequest)
	}

	dbUser, err := db.Provider.GetUserByEmail(email)
	if err == nil {
		db.Provider.DeleteUser(dbUser)
		db.Provider.DeleteSession(dbUser.ID)
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
		Email:    fmt.Sprintf("%d_authorizer_tester@yopmail.com", time.Now().Unix()),
		Password: "Test@123",
	}

	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEnvPath, "../../.env.sample")
	env.InitRequiredEnv()
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeySmtpHost, "smtp.yopmail.com")
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeySmtpPort, "2525")
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeySmtpUsername, "lakhan@yopmail.com")
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeySmtpPassword, "test")
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeySenderEmail, "info@yopmail.com")
	envstore.EnvStoreObj.UpdateEnvVariable(constants.SliceStoreIdentifier, constants.EnvKeyProtectedRoles, []string{"admin"})
	db.InitDB()
	env.InitAllEnv()
	sessionstore.InitSession()

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
