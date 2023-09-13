package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyTotpResolver resolver for verify totp mutation
func VerifyTotpResolver(ctx context.Context, params model.VerifyTOTPRequest) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	encryptedkey := params.Token

	pvtKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey)
	if err != nil {
		log.Debug("error while getting private key")
	}

	privateKey, err := crypto.ParseRsaPrivateKeyFromPemStr(pvtKey)
	if err != nil {
		log.Debug("error while parsing private key")
	}

	userID, err := crypto.DecryptRSA(encryptedkey, *privateKey)
	if err != nil {
		log.Debug("error while decrypting userId")
	}

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	user, err := db.Provider.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	status, err := db.Provider.ValidatePasscode(ctx, params.Otp, userID)
	if err != nil || !status {
		return nil, fmt.Errorf("error while validating passcode")
	}

	code := ""
	codeChallenge := ""
	nonce := ""

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}

	// Get state from store
	if params.State != nil {
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

	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodBasicAuth, nonce, code)
	if err != nil {
		log.Debug("Failed to create auth token", err)
		return res, err
	}

	// TODO add to other login options as well
	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
			log.Debug("SetState failed: ", err)
			return res, err
		}
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}

	cookie.SetSession(gc, authToken.FingerPrintHash)
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		db.Provider.AddSession(ctx, &models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}
