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
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyEmailResolver is a resolver for verify email mutation
func VerifyEmailResolver(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByToken(ctx, params.Token)
	if err != nil {
		log.Debug("Failed to get verification request by token: ", err)
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	// verify if token exists in db
	hostname := parsers.GetHost(gc)
	claim, err := token.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug("Failed to parse token: ", err)
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	if ok, err := token.ValidateJWTClaims(claim, hostname, verificationRequest.Nonce, verificationRequest.Email); !ok || err != nil {
		log.Debug("Failed to validate jwt claims: ", err)
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	email := claim["sub"].(string)
	log := log.WithFields(log.Fields{
		"email": email,
	})
	user, err := db.Provider.GetUserByEmail(ctx, email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return res, err
	}

	isMFADisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil || !isMFADisabled {
		log.Debug("MFA service not enabled: ", err)
	}

	isTOTPLoginDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableTOTPLogin)
	if err != nil || !isTOTPLoginDisabled {
		log.Debug("totp service not enabled: ", err)
	}

	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return err
		}
		cookie.SetMfaSession(gc, mfaSession)
		return nil
	}

	// If mfa enabled and also totp enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled && !isTOTPLoginDisabled {
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug("Failed to set mfa session: ", err)
			return nil, err
		}
		authenticator, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		if err != nil || authenticator == nil || authenticator.VerifiedAt == nil {
			// generate totp
			// Generate a base64 URL and initiate the registration for TOTP
			authConfig, err := authenticators.Provider.Generate(ctx, user.ID)
			if err != nil {
				log.Debug("error while generating base64 url: ", err)
				return nil, err
			}
			recoveryCodes := []*string{}
			for _, code := range authConfig.RecoveryCodes {
				recoveryCodes = append(recoveryCodes, refs.NewStringRef(code))
			}
			// when user is first time registering for totp
			res = &model.AuthResponse{
				Message:                    `Proceed to totp verification screen`,
				ShouldShowTotpScreen:       refs.NewBoolRef(true),
				AuthenticatorScannerImage:  refs.NewStringRef(authConfig.ScannerImage),
				AuthenticatorSecret:        refs.NewStringRef(authConfig.Secret),
				AuthenticatorRecoveryCodes: recoveryCodes,
			}
			return res, nil
		} else {
			//when user is already register for totp
			res = &model.AuthResponse{
				Message:              `Proceed to totp screen`,
				ShouldShowTotpScreen: refs.NewBoolRef(true),
			}
			return res, nil
		}
	}

	isSignUp := false
	if user.EmailVerifiedAt == nil {
		isSignUp = true
		// update email_verified_at in users table
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
		user, err = db.Provider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug("Failed to update user: ", err)
			return res, err
		}
	}
	// delete from verification table
	err = db.Provider.DeleteVerificationRequest(gc, verificationRequest)
	if err != nil {
		log.Debug("Failed to delete verification request: ", err)
		return res, err
	}

	loginMethod := constants.AuthRecipeMethodBasicAuth
	if loginMethod == constants.VerificationTypeMagicLinkLogin {
		loginMethod = constants.AuthRecipeMethodMagicLinkLogin
	}

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	code := ""
	// Not required as /oauth/token cannot be resumed from other tab
	// codeChallenge := ""
	nonce := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := memorystore.Provider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				// Not required as /oauth/token cannot be resumed from other tab
				// codeChallenge = authorizeStateSplit[1]
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
	// Not required as /oauth/token cannot be resumed from other tab
	// if code != "" {
	// 	if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
	// 		log.Debug("SetState failed: ", err)
	// 		return res, err
	// 	}
	// }
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
	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
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
