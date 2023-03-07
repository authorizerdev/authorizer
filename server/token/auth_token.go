package token

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/robertkrimen/otto"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
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
	Subject     string   `json:"sub"`
	Roles       []string `json:"roles"`
	Scope       []string `json:"scope"`
	Nonce       string   `json:"nonce"`
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
	LoginMethod string   `json:"login_method"`
}

// CreateAuthToken creates a new auth token when userlogs in
func CreateAuthToken(gc *gin.Context, user models.User, roles, scope []string, loginMethod, nonce string, code string) (*Token, error) {
	hostname := parsers.GetHost(gc)
	_, fingerPrintHash, err := CreateSessionToken(user, nonce, roles, scope, loginMethod)
	if err != nil {
		return nil, err
	}
	accessToken, accessTokenExpiresAt, err := CreateAccessToken(user, roles, scope, hostname, nonce, loginMethod)
	if err != nil {
		return nil, err
	}

	atHash := sha256.New()
	atHash.Write([]byte(accessToken))
	atHashBytes := atHash.Sum(nil)
	// hashedToken := string(bs)
	atHashDigest := atHashBytes[0 : len(atHashBytes)/2]
	atHashString := base64.RawURLEncoding.EncodeToString(atHashDigest)

	codeHashString := ""
	if code != "" {
		codeHash := sha256.New()
		codeHash.Write([]byte(code))
		codeHashBytes := codeHash.Sum(nil)
		codeHashDigest := codeHashBytes[0 : len(codeHashBytes)/2]
		codeHashString = base64.RawURLEncoding.EncodeToString(codeHashDigest)
	}

	idToken, idTokenExpiresAt, err := CreateIDToken(user, roles, hostname, nonce, atHashString, codeHashString, loginMethod)
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
		refreshToken, refreshTokenExpiresAt, err := CreateRefreshToken(user, roles, scope, hostname, nonce, loginMethod)
		if err != nil {
			return nil, err
		}

		res.RefreshToken = &JWTToken{Token: refreshToken, ExpiresAt: refreshTokenExpiresAt}
	}

	return res, nil
}

// CreateSessionToken creates a new session token
func CreateSessionToken(user models.User, nonce string, roles, scope []string, loginMethod string) (*SessionData, string, error) {
	fingerPrintMap := &SessionData{
		Nonce:       nonce,
		Roles:       roles,
		Subject:     user.ID,
		Scope:       scope,
		LoginMethod: loginMethod,
		IssuedAt:    time.Now().Unix(),
		ExpiresAt:   time.Now().AddDate(1, 0, 0).Unix(),
	}
	fingerPrintBytes, _ := json.Marshal(fingerPrintMap)
	fingerPrintHash, err := crypto.EncryptAES(string(fingerPrintBytes))
	if err != nil {
		return nil, "", err
	}

	return fingerPrintMap, fingerPrintHash, nil
}

// CreateRefreshToken util to create JWT token
func CreateRefreshToken(user models.User, roles, scopes []string, hostname, nonce, loginMethod string) (string, int64, error) {
	// expires in 1 year
	expiryBound := time.Hour * 8760
	expiresAt := time.Now().Add(expiryBound).Unix()
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return "", 0, err
	}
	customClaims := jwt.MapClaims{
		"iss":           hostname,
		"aud":           clientID,
		"sub":           user.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeRefreshToken,
		"roles":         roles,
		"scope":         scopes,
		"nonce":         nonce,
		"login_method":  loginMethod,
		"allowed_roles": strings.Split(user.Roles, ","),
	}

	token, err := SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// CreateAccessToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
func CreateAccessToken(user models.User, roles, scopes []string, hostName, nonce, loginMethod string) (string, int64, error) {
	expireTime, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAccessTokenExpiryTime)
	if err != nil {
		return "", 0, err
	}
	expiryBound, err := utils.ParseDurationInSeconds(expireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return "", 0, err
	}
	customClaims := jwt.MapClaims{
		"iss":           hostName,
		"aud":           clientID,
		"nonce":         nonce,
		"sub":           user.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeAccessToken,
		"scope":         scopes,
		"roles":         roles,
		"login_method":  loginMethod,
		"allowed_roles": strings.Split(user.Roles, ","),
	}
	// check for the extra access token script
	accessTokenScript, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyCustomAccessTokenScript)
	if err != nil {
		log.Debug("Failed to get custom access token script: ", err)
		accessTokenScript = ""
	}
	if accessTokenScript != "" {
		resUser := user.AsAPIUser()
		userBytes, _ := json.Marshal(&resUser)
		var userMap map[string]interface{}
		json.Unmarshal(userBytes, &userMap)
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
			log.Debug("error getting custom access token script: ", err)
		} else {
			extraPayload := make(map[string]interface{})
			err = json.Unmarshal([]byte(fmt.Sprintf("%s", val)), &extraPayload)
			if err != nil {
				log.Debug("error converting accessTokenScript response to map: ", err)
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
	res := make(map[string]interface{})

	if accessToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	res, err := ParseJWTToken(accessToken)
	if err != nil {
		return res, err
	}

	userID := res["sub"].(string)
	nonce := res["nonce"].(string)
	loginMethod := res["login_method"]
	sessionKey := userID
	if loginMethod != nil && loginMethod != "" {
		sessionKey = loginMethod.(string) + ":" + userID
	}

	token, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce)
	if nonce == "" || err != nil {
		return res, fmt.Errorf(`unauthorized`)
	}

	if token != accessToken {
		return res, fmt.Errorf(`unauthorized`)
	}

	hostname := parsers.GetHost(gc)
	if ok, err := ValidateJWTClaims(res, hostname, nonce, userID); !ok || err != nil {
		return res, err
	}

	if res["token_type"] != constants.TokenTypeAccessToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}

// Function to validate refreshToken
func ValidateRefreshToken(gc *gin.Context, refreshToken string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	if refreshToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	res, err := ParseJWTToken(refreshToken)
	if err != nil {
		return res, err
	}

	userID := res["sub"].(string)
	nonce := res["nonce"].(string)
	loginMethod := res["login_method"]
	sessionKey := userID
	if loginMethod != nil && loginMethod != "" {
		sessionKey = loginMethod.(string) + ":" + userID
	}
	token, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+nonce)
	if nonce == "" || err != nil {
		return res, fmt.Errorf(`unauthorized`)
	}

	if token != refreshToken {
		return res, fmt.Errorf(`unauthorized`)
	}

	hostname := parsers.GetHost(gc)
	if ok, err := ValidateJWTClaims(res, hostname, nonce, userID); !ok || err != nil {
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

	decryptedFingerPrint, err := crypto.DecryptAES(encryptedSession)
	if err != nil {
		return nil, err
	}

	var res SessionData
	err = json.Unmarshal([]byte(decryptedFingerPrint), &res)
	if err != nil {
		return nil, err
	}

	sessionStoreKey := res.Subject
	if res.LoginMethod != "" {
		sessionStoreKey = res.LoginMethod + ":" + res.Subject
	}
	token, err := memorystore.Provider.GetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+res.Nonce)
	if token == "" || err != nil {
		log.Debug("invalid browser session:", err)
		return nil, fmt.Errorf(`unauthorized`)
	}

	if encryptedSession != token {
		return nil, fmt.Errorf(`unauthorized: invalid nonce`)
	}

	if res.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf(`unauthorized: token expired`)
	}

	return &res, nil
}

// CreateIDToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
// For response_type (code) / authorization_code grant nonce should be empty
// for implicit flow it should be present to verify with actual state
func CreateIDToken(user models.User, roles []string, hostname, nonce, atHash, cHash, loginMethod string) (string, int64, error) {
	expireTime, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAccessTokenExpiryTime)
	if err != nil {
		return "", 0, err
	}
	expiryBound, err := utils.ParseDurationInSeconds(expireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	resUser := user.AsAPIUser()
	userBytes, _ := json.Marshal(&resUser)
	var userMap map[string]interface{}
	json.Unmarshal(userBytes, &userMap)
	claimKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtRoleClaim)
	if err != nil {
		claimKey = "roles"
	}

	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return "", 0, err
	}

	customClaims := jwt.MapClaims{
		"iss":           hostname,
		"aud":           clientID,
		"sub":           user.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeIdentityToken,
		"allowed_roles": strings.Split(user.Roles, ","),
		"login_method":  loginMethod,
		claimKey:        roles,
	}
	// split nonce to see if its authorization code grant method
	if cHash != "" {
		customClaims["at_hash"] = atHash
		customClaims["c_hash"] = cHash
	} else {
		customClaims["nonce"] = nonce
		customClaims["at_hash"] = atHash
	}
	for k, v := range userMap {
		if k != "roles" {
			customClaims[k] = v
		}
	}
	// check for the extra access token script
	accessTokenScript, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyCustomAccessTokenScript)
	if err != nil {
		log.Debug("Failed to get custom access token script: ", err)
		accessTokenScript = ""
	}
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
			log.Debug("error getting custom access token script: ", err)
		} else {
			extraPayload := make(map[string]interface{})
			err = json.Unmarshal([]byte(fmt.Sprintf("%s", val)), &extraPayload)
			if err != nil {
				log.Debug("error converting accessTokenScript response to map: ", err)
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
