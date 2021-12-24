package test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/middlewares"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/gin-contrib/location"
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
	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
	if err == nil {
		err = db.Mgr.DeleteVerificationRequest(verificationRequest)
	}

	verificationRequest, err = db.Mgr.GetVerificationByEmail(email, enum.ForgotPassword.String())
	if err == nil {
		err = db.Mgr.DeleteVerificationRequest(verificationRequest)
	}

	verificationRequest, err = db.Mgr.GetVerificationByEmail(email, enum.UpdateEmail.String())
	if err == nil {
		err = db.Mgr.DeleteVerificationRequest(verificationRequest)
	}

	dbUser, err := db.Mgr.GetUserByEmail(email)
	if err == nil {
		db.Mgr.DeleteUser(dbUser)
		db.Mgr.DeleteUserSession(dbUser.ID)
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
		Email:    "authorizer_tester@yopmail.com",
		Password: "test",
	}

	constants.ENV_PATH = "../../.env.sample"
	env.InitEnv()
	session.InitSession()

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.Use(location.Default())
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
