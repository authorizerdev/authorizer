package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ResendVerifyEmailResolver is a resolver for resend verify email mutation
func ResendVerifyEmailResolver(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	var res *model.Response
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return res, err
	}
	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		return res, fmt.Errorf("invalid email")
	}

	if !utils.IsValidVerificationIdentifier(params.Identifier) {
		return res, fmt.Errorf("invalid identifier")
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByEmail(params.Email, params.Identifier)
	if err != nil {
		return res, fmt.Errorf(`verification request not found`)
	}

	// delete current verification and create new one
	err = db.Provider.DeleteVerificationRequest(verificationRequest)
	if err != nil {
		log.Println("error deleting verification request:", err)
	}

	hostname := utils.GetHost(gc)
	verificationToken, err := token.CreateVerificationToken(params.Email, params.Identifier, hostname)
	if err != nil {
		log.Println(`error generating token`, err)
	}
	db.Provider.AddVerificationRequest(models.VerificationRequest{
		Token:      verificationToken,
		Identifier: params.Identifier,
		ExpiresAt:  time.Now().Add(time.Minute * 30).Unix(),
		Email:      params.Email,
	})

	// exec it as go routin so that we can reduce the api latency
	go func() {
		email.SendVerificationMail(params.Email, verificationToken, hostname)
	}()

	res = &model.Response{
		Message: `Verification email has been sent. Please check your inbox`,
	}

	return res, nil
}
