package token

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/robertkrimen/otto"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// JWTToken is a struct to hold JWT token and its expiration time
type JWTToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// Token object to hold the finger print and refresh token information
type Token struct {
	FingerPrint     string    `json:"fingerprint"`
	FingerPrintHash string    `json:"fingerprint_hash"`
	RefreshToken    *JWTToken `json:"refresh_token"`
	AccessToken     *JWTToken `json:"access_token"`
	IDToken         *JWTToken `json:"id_token"`
}

// SessionData
type SessionData struct {
	Subject   string   `json:"sub"`
	Roles     []string `json:"roles"`
	Scope     []string `json:"scope"`
	Nonce     string   `json:"nonce"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

// CreateSessionToken creates a new session token
func CreateSessionToken(user models.User, nonce string, roles, scope []string) (*SessionData, string, error) {
	fingerPrintMap := &SessionData{
		Nonce:     nonce,
		Roles:     roles,
		Subject:   user.ID,
		Scope:     scope,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().AddDate(1, 0, 0).Unix(),
	}
	fingerPrintBytes, _ := json.Marshal(fingerPrintMap)
	fingerPrintHash, err := crypto.EncryptAES(string(fingerPrintBytes))
	if err != nil {
		return nil, "", err
	}

	return fingerPrintMap, fingerPrintHash, nil
}

// CreateAuthToken creates a new auth token when userlogs in
func CreateAuthToken(gc *gin.Context, user models.User, roles, scope []string) (*Token, error) {
	hostname := utils.GetHost(gc)
	nonce := uuid.New().String()
	_, fingerPrintHash, err := CreateSessionToken(user, nonce, roles, scope)
	if err != nil {
		return nil, err
	}
	accessToken, accessTokenExpiresAt, err := CreateAccessToken(user, roles, scope, hostname, nonce)
	if err != nil {
		return nil, err
	}

	idToken, idTokenExpiresAt, err := CreateIDToken(user, roles, hostname, nonce)
	if err != nil {
		return nil, err
	}

	res := &Token{
		FingerPrint:     nonce,
		FingerPrintHash: fingerPrintHash,
		AccessToken:     &JWTToken{Token: accessToken, ExpiresAt: accessTokenExpiresAt},
		IDToken:         &JWTToken{Token: idToken, ExpiresAt: idTokenExpiresAt},
	}

	if utils.StringSliceContains(scope, "offline_access") {
		refreshToken, refreshTokenExpiresAt, err := CreateRefreshToken(user, roles, scope, hostname, nonce)
		if err != nil {
			return nil, err
		}

		res.RefreshToken = &JWTToken{Token: refreshToken, ExpiresAt: refreshTokenExpiresAt}
	}

	return res, nil
}

// CreateRefreshToken util to create JWT token
func CreateRefreshToken(user models.User, roles, scopes []string, hostname, nonce string) (string, int64, error) {
	// expires in 1 year
	expiryBound := time.Hour * 8760
	expiresAt := time.Now().Add(expiryBound).Unix()
	customClaims := jwt.MapClaims{
		"iss":        hostname,
		"aud":        envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		"sub":        user.ID,
		"exp":        expiresAt,
		"iat":        time.Now().Unix(),
		"token_type": constants.TokenTypeRefreshToken,
		"roles":      roles,
		"scope":      scopes,
		"nonce":      nonce,
	}

	token, err := SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// CreateAccessToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
func CreateAccessToken(user models.User, roles, scopes []string, hostName, nonce string) (string, int64, error) {
	expiryBound := time.Minute * 30
	expiresAt := time.Now().Add(expiryBound).Unix()

	customClaims := jwt.MapClaims{
		"iss":        hostName,
		"aud":        envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		"nonce":      nonce,
		"sub":        user.ID,
		"exp":        expiresAt,
		"iat":        time.Now().Unix(),
		"token_type": constants.TokenTypeAccessToken,
		"scope":      scopes,
		"roles":      roles,
	}

	token, err := SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// GetAccessToken returns the access token from the request (either from header or cookie)
func GetAccessToken(gc *gin.Context) (string, error) {
	// try to check in auth header for cookie
	auth := gc.Request.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf(`unauthorized`)
	}

	authSplit := strings.Split(auth, " ")
	if len(authSplit) != 2 {
		return "", fmt.Errorf(`unauthorized`)
	}

	if strings.ToLower(authSplit[0]) != "bearer" {
		return "", fmt.Errorf(`not a bearer token`)
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	return token, nil
}

// Function to validate access token for authorizer apis (profile, update_profile)
func ValidateAccessToken(gc *gin.Context, accessToken string) (map[string]interface{}, error) {
	var res map[string]interface{}

	if accessToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	savedSession := sessionstore.GetState(accessToken)
	if savedSession == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	savedSessionSplit := strings.Split(savedSession, "@")
	nonce := savedSessionSplit[0]
	userID := savedSessionSplit[1]

	hostname := utils.GetHost(gc)
	res, err := ParseJWTToken(accessToken, hostname, nonce, userID)
	if err != nil {
		return res, err
	}

	if res["token_type"] != constants.TokenTypeAccessToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}

// Function to validate refreshToken
func ValidateRefreshToken(gc *gin.Context, refreshToken string) (map[string]interface{}, error) {
	var res map[string]interface{}

	if refreshToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	savedSession := sessionstore.GetState(refreshToken)
	if savedSession == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	savedSessionSplit := strings.Split(savedSession, "@")
	nonce := savedSessionSplit[0]
	userID := savedSessionSplit[1]

	hostname := utils.GetHost(gc)
	res, err := ParseJWTToken(refreshToken, hostname, nonce, userID)
	if err != nil {
		return res, err
	}

	if res["token_type"] != constants.TokenTypeRefreshToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}

func ValidateBrowserSession(gc *gin.Context, encryptedSession string) (*SessionData, error) {
	if encryptedSession == "" {
		return nil, fmt.Errorf(`unauthorized`)
	}

	savedSession := sessionstore.GetState(encryptedSession)
	if savedSession == "" {
		return nil, fmt.Errorf(`unauthorized`)
	}

	savedSessionSplit := strings.Split(savedSession, "@")
	nonce := savedSessionSplit[0]
	userID := savedSessionSplit[1]

	decryptedFingerPrint, err := crypto.DecryptAES(encryptedSession)
	if err != nil {
		return nil, err
	}

	var res SessionData
	err = json.Unmarshal([]byte(decryptedFingerPrint), &res)
	if err != nil {
		return nil, err
	}

	if res.Nonce != nonce {
		return nil, fmt.Errorf(`unauthorized: invalid nonce`)
	}

	if res.Subject != userID {
		return nil, fmt.Errorf(`unauthorized: invalid user id`)
	}

	if res.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf(`unauthorized: token expired`)
	}

	// TODO validate scope
	// if !reflect.DeepEqual(res.Roles, roles) {
	// 	return res, "", fmt.Errorf(`unauthorized`)
	// }

	return &res, nil
}

// CreateIDToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
func CreateIDToken(user models.User, roles []string, hostname, nonce string) (string, int64, error) {
	expiryBound := time.Minute * 30
	expiresAt := time.Now().Add(expiryBound).Unix()

	resUser := user.AsAPIUser()
	userBytes, _ := json.Marshal(&resUser)
	var userMap map[string]interface{}
	json.Unmarshal(userBytes, &userMap)

	claimKey := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtRoleClaim)
	customClaims := jwt.MapClaims{
		"iss":           hostname,
		"aud":           envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		"nonce":         nonce,
		"sub":           user.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeIdentityToken,
		"allowed_roles": strings.Split(user.Roles, ","),
		claimKey:        roles,
	}

	for k, v := range userMap {
		if k != "roles" {
			customClaims[k] = v
		}
	}

	// check for the extra access token script
	accessTokenScript := os.Getenv(constants.EnvKeyCustomAccessTokenScript)
	if accessTokenScript != "" {
		vm := otto.New()

		claimBytes, _ := json.Marshal(customClaims)
		vm.Run(fmt.Sprintf(`
			var user = %s;
			var tokenPayload = %s;
			var customFunction = %s;
			var functionRes = JSON.stringify(customFunction(user, tokenPayload));
		`, string(userBytes), string(claimBytes), accessTokenScript))

		val, err := vm.Get("functionRes")

		if err != nil {
			log.Println("error getting custom access token script:", err)
		} else {
			extraPayload := make(map[string]interface{})
			err = json.Unmarshal([]byte(fmt.Sprintf("%s", val)), &extraPayload)
			if err != nil {
				log.Println("error converting accessTokenScript response to map:", err)
			} else {
				for k, v := range extraPayload {
					customClaims[k] = v
				}
			}
		}
	}

	token, err := SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// GetIDToken returns the id token from the request header
func GetIDToken(gc *gin.Context) (string, error) {
	// try to check in auth header for cookie
	auth := gc.Request.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf(`unauthorized`)
	}

	authSplit := strings.Split(auth, " ")
	if len(authSplit) != 2 {
		return "", fmt.Errorf(`unauthorized`)
	}

	if strings.ToLower(authSplit[0]) != "bearer" {
		return "", fmt.Errorf(`not a bearer token`)
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	return token, nil
}
