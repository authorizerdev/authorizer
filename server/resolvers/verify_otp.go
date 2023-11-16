package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/authenticators"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyOtpResolver resolver for verify otp mutation
func VerifyOtpResolver(ctx context.Context, params model.VerifyOTPRequest) (*model.AuthResponse, error) {
	var res *model.AuthResponse
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		log.Debug("Failed to get otp request by email: ", err)
		return res, fmt.Errorf(`invalid session: %s`, err.Error())
	}

	if refs.StringValue(params.Email) == "" && refs.StringValue(params.PhoneNumber) == "" {
		log.Debug("Email or phone number is required")
		return res, fmt.Errorf(`email or phone_number is required`)
	}
	currentField := models.FieldNameEmail
	if refs.StringValue(params.Email) == "" {
		currentField = models.FieldNamePhoneNumber
	}
	// Get user by email or phone number
	var user *models.User
	if currentField == models.FieldNameEmail {
		user, err = db.Provider.GetUserByEmail(ctx, refs.StringValue(params.Email))
	} else {
		user, err = db.Provider.GetUserByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
	}
	if user == nil || err != nil {
		log.Debug("Failed to get user by email or phone number: ", err)
		return res, err
	}
	// Verify OTP based on TOPT or OTP
	if refs.BoolValue(params.Totp) {
		status, err := authenticators.Provider.Validate(ctx, params.Otp, user.ID)
		if err != nil || !status {
			log.Debug("Failed to validate totp: ", err)
			return nil, fmt.Errorf("error while validating passcode")
		}
	} else {
		var otp *models.OTP
		if currentField == models.FieldNameEmail {
			otp, err = db.Provider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
		} else {
			otp, err = db.Provider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
		}
		if otp == nil && err != nil {
			log.Debugf("Failed to get otp request for %s: %s", currentField, err.Error())
			return res, fmt.Errorf(`invalid %s: %s`, currentField, err.Error())
		}
		if params.Otp != otp.Otp {
			log.Debug("Failed to verify otp request: Incorrect value")
			return res, fmt.Errorf(`invalid otp`)
		}
		expiresIn := otp.ExpiresAt - time.Now().Unix()
		if expiresIn < 0 {
			log.Debug("Failed to verify otp request: Timeout")
			return res, fmt.Errorf("otp expired")
		}
		db.Provider.DeleteOTP(gc, otp)
	}

	if _, err := memorystore.Provider.GetMfaSession(user.ID, mfaSession); err != nil {
		log.Debug("Failed to get mfa session: ", err)
		return res, fmt.Errorf(`invalid session: %s`, err.Error())
	}

	isSignUp := user.EmailVerifiedAt == nil && user.PhoneNumberVerifiedAt == nil
	// TODO - Add Login method in DB when we introduce OTP for social media login
	loginMethod := constants.AuthRecipeMethodBasicAuth
	if currentField == models.FieldNamePhoneNumber {
		loginMethod = constants.AuthRecipeMethodMobileOTP
	}
	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	code := ""
	codeChallenge := ""
	nonce := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := memorystore.Provider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go memorystore.Provider.RemoveState(refs.StringValue(params.State))
		}
	}
	if nonce == "" {
		nonce = uuid.New().String()
	}
	authToken, err := token.CreateAuthToken(gc, user, roles, scope, loginMethod, nonce, code)
	if err != nil {
		log.Debug("Failed to create auth token: ", err)
		return res, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
			log.Debug("Failed to set code state: ", err)
			return res, err
		}
	}

	go func() {
		if isSignUp {
			utils.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
			// User is also logged in with signup
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		} else {
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		db.Provider.AddSession(ctx, &models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	authTokenExpiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if authTokenExpiresIn <= 0 {
		authTokenExpiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `OTP verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &authTokenExpiresIn,
		User:        user.AsAPIUser(),
	}

	sessionKey := loginMethod + ":" + user.ID
	cookie.SetSession(gc, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, nil
}
