package test

import (
	"context"
	"log"
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
	Ctx        context.Context
	Server     *httptest.Server
	Req        *http.Request
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
	if err != nil {
		log.Println("error getting user:", err)
	} else {
		err = db.Mgr.DeleteUser(dbUser)
		if err != nil {
			log.Println("error deleting user:", err)
		}

		err = db.Mgr.DeleteUserSession(dbUser.ID)
		if err != nil {
			log.Println("error deleting user session:", err)
		}
	}
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

	req, _ := http.NewRequest(
		"POST",
		"http://"+server.Listener.Addr().String()+"/graphql",
		nil,
	)
	req.Header.Add("x-authorizer-admin-secret", constants.ADMIN_SECRET)
	c.Request = req
	ctx := context.WithValue(req.Context(), "GinContextKey", c)

	return TestSetup{
		GinEngine:  r,
		GinContext: c,
		Ctx:        ctx,
		Server:     server,
		Req:        req,
		TestInfo:   testData,
	}
}
