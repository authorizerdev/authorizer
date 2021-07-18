package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/utils"
)

func ResendVerifyEmail(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	var res *model.Response
	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		return res, fmt.Errorf("invalid email")
	}

	verificationRequest, err := db.Mgr.GetVerificationByEmail(params.Email)
	if err != nil {
		return res, fmt.Errorf(`verification request not found`)
	}

	token, err := utils.CreateVerificationToken(params.Email, verificationRequest.Identifier)
	if err != nil {
		log.Println(`Error generating token`, err)
	}
	db.Mgr.AddVerification(db.VerificationRequest{
		Token:      token,
		Identifier: verificationRequest.Identifier,
		ExpiresAt:  time.Now().Add(time.Minute * 30).Unix(),
		Email:      params.Email,
	})

	// exec it as go routin so that we can reduce the api latency
	go func() {
		utils.SendVerificationMail(params.Email, token)
	}()

	res = &model.Response{
		Message: `Verification email has been sent. Please check your inbox`,
	}

	return res, nil
}
