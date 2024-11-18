package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/data_store/db"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
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

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug("Email or phone number is required")
		return res, fmt.Errorf(`email or phone number is required`)
	}
	isEmailVerification := email != ""
	isMobileVerification := phoneNumber != ""
	// Get user by email or phone number
	var user *models.User
	if isEmailVerification {
		user, err = db.Provider.GetUserByEmail(ctx, refs.StringValue(params.Email))
		if err != nil {
			log.Debug("Failed to get user by email: ", err)
		}
	} else {
		user, err = db.Provider.GetUserByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
		if err != nil {
			log.Debug("Failed to get user by phone number: ", err)
		}
	}
	if user == nil || err != nil {
		return res, fmt.Errorf(`user not found`)
	}
	// Verify OTP based on TOPT or OTP
	if refs.BoolValue(params.IsTotp) {
		status, err := authenticators.Provider.Validate(ctx, params.Otp, user.ID)
		if err != nil {
			log.Debug("Failed to validate totp: ", err)
			return nil, fmt.Errorf("error while validating passcode")
		}
		if !status {
			log.Debug("Failed to verify otp request: Incorrect value")
			log.Info("Checking if otp is recovery code")
			// Check if otp is recovery code
			isValidRecoveryCode, err := authenticators.Provider.ValidateRecoveryCode(ctx, params.Otp, user.ID)
			if err != nil {
				log.Debug("Failed to validate recovery code: ", err)
				return nil, fmt.Errorf("error while validating recovery code")
			}
			if !isValidRecoveryCode {
				log.Debug("Failed to verify otp request: Incorrect value")
				return res, fmt.Errorf(`invalid otp`)
			}
		}
	} else {
		var otp *models.OTP
		if isEmailVerification {
			otp, err = db.Provider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
			if err != nil {
				log.Debug(`Failed to get otp request for email: `, err.Error())
			}
		} else {
			otp, err = db.Provider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
			if err != nil {
				log.Debug(`Failed to get otp request for phone number: `, err.Error())
			}
		}
		if otp == nil && err != nil {
			return res, fmt.Errorf(`OTP not found`)
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

	isSignUp := false
	if user.EmailVerifiedAt == nil && isEmailVerification {
		isSignUp = true
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	if user.PhoneNumberVerifiedAt == nil && isMobileVerification {
		isSignUp = true
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	if isSignUp {
		user, err = db.Provider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug("Failed to update user: ", err)
			return res, err
		}
	}
	loginMethod := constants.AuthRecipeMethodBasicAuth
	if isMobileVerification {
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
