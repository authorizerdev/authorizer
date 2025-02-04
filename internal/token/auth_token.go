package token

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/robertkrimen/otto"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AuthTokenConfig is the configuration for auth token
type AuthTokenConfig struct {
	LoginMethod string
	Nonce       string
	Code        string
	AtHash      string
	CodeHash    string
	ExpireTime  string
	User        *schemas.User
	HostName    string
	Roles       []string
	Scope       []string
}

// JWTToken is a struct to hold JWT token and its expiration time
type JWTToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// AuthToken object to hold the finger print, access token, id token and refresh token information
type AuthToken struct {
	FingerPrint string `json:"fingerprint"`
	// Session Token
	FingerPrintHash       string    `json:"fingerprint_hash"`
	SessionTokenExpiresAt int64     `json:"expires_at"`
	RefreshToken          *JWTToken `json:"refresh_token"`
	AccessToken           *JWTToken `json:"access_token"`
	IDToken               *JWTToken `json:"id_token"`
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
func (p *provider) CreateAuthToken(gc *gin.Context, cfg *AuthTokenConfig) (*AuthToken, error) {
	_, fingerPrintHash, sessionTokenExpiresAt, err := p.CreateSessionToken(cfg)
	if err != nil {
		return nil, err
	}
	accessToken, accessTokenExpiresAt, err := p.CreateAccessToken(cfg)
	if err != nil {
		return nil, err
	}

	atHash := sha256.New()
	atHash.Write([]byte(accessToken))
	atHashBytes := atHash.Sum(nil)
	// hashedToken := string(bs)
	atHashDigest := atHashBytes[0 : len(atHashBytes)/2]
	atHashString := base64.RawURLEncoding.EncodeToString(atHashDigest)
	cfg.AtHash = atHashString
	codeHashString := ""
	if cfg.Code != "" {
		codeHash := sha256.New()
		codeHash.Write([]byte(cfg.Code))
		codeHashBytes := codeHash.Sum(nil)
		codeHashDigest := codeHashBytes[0 : len(codeHashBytes)/2]
		codeHashString = base64.RawURLEncoding.EncodeToString(codeHashDigest)
	}
	cfg.CodeHash = codeHashString
	idToken, idTokenExpiresAt, err := p.CreateIDToken(cfg)
	if err != nil {
		return nil, err
	}

	res := &AuthToken{
		FingerPrint:           cfg.Nonce,
		FingerPrintHash:       fingerPrintHash,
		SessionTokenExpiresAt: sessionTokenExpiresAt,
		AccessToken:           &JWTToken{Token: accessToken, ExpiresAt: accessTokenExpiresAt},
		IDToken:               &JWTToken{Token: idToken, ExpiresAt: idTokenExpiresAt},
	}
	if utils.StringSliceContains(cfg.Scope, "offline_access") {
		refreshToken, refreshTokenExpiresAt, err := p.CreateRefreshToken(cfg)
		if err != nil {
			return nil, err
		}

		res.RefreshToken = &JWTToken{Token: refreshToken, ExpiresAt: refreshTokenExpiresAt}
	}

	return res, nil
}

// CreateSessionToken creates a new session token
func (p *provider) CreateSessionToken(cfg *AuthTokenConfig) (*SessionData, string, int64, error) {
	expiresAt := time.Now().AddDate(1, 0, 0).Unix()
	fingerPrintMap := &SessionData{
		Nonce:       cfg.Nonce,
		Roles:       cfg.Roles,
		Subject:     cfg.User.ID,
		Scope:       cfg.Scope,
		LoginMethod: cfg.LoginMethod,
		IssuedAt:    time.Now().Unix(),
		ExpiresAt:   expiresAt,
	}
	fingerPrintBytes, _ := json.Marshal(fingerPrintMap)
	fingerPrintHash, err := crypto.EncryptAES(p.config.ClientSecret, string(fingerPrintBytes))
	if err != nil {
		return nil, "", 0, err
	}

	return fingerPrintMap, fingerPrintHash, expiresAt, nil
}

// CreateRefreshToken util to create JWT token
func (p *provider) CreateRefreshToken(cfg *AuthTokenConfig) (string, int64, error) {
	// expires in 1 year
	expiryBound := time.Hour * 8760
	expiresAt := time.Now().Add(expiryBound).Unix()
	customClaims := jwt.MapClaims{
		"iss":           cfg.HostName,
		"aud":           p.config.ClientID,
		"sub":           cfg.User.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeRefreshToken,
		"roles":         cfg.Roles,
		"scope":         cfg.Scope,
		"nonce":         cfg.Nonce,
		"login_method":  cfg.Nonce,
		"allowed_roles": strings.Split(cfg.User.Roles, ","),
	}

	token, err := p.SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// CreateAccessToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
func (p *provider) CreateAccessToken(cfg *AuthTokenConfig) (string, int64, error) {
	expiryBound, err := utils.ParseDurationInSeconds(cfg.ExpireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	customClaims := jwt.MapClaims{
		"iss":           cfg.HostName,
		"aud":           p.config.ClientID,
		"nonce":         cfg.Nonce,
		"sub":           cfg.User.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeAccessToken,
		"scope":         cfg.Scope,
		"roles":         cfg.Roles,
		"login_method":  cfg.LoginMethod,
		"allowed_roles": strings.Split(cfg.User.Roles, ","),
	}
	// check for the extra access token script
	if p.config.CustomAccessTokenScript != "" {
		resUser := cfg.User.AsAPIUser()
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
		`, string(userBytes), string(claimBytes), p.config.CustomAccessTokenScript))

		val, err := vm.Get("functionRes")
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("error getting custom access token script")
		} else {
			extraPayload := make(map[string]interface{})
			err = json.Unmarshal([]byte(fmt.Sprintf("%v", val)), &extraPayload)
			if err != nil {
				p.dependencies.Log.Debug().Err(err).Msg("error converting accessTokenScript response to map")
			} else {
				for k, v := range extraPayload {
					customClaims[k] = v
				}
			}
		}
	}
	token, err := p.SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// GetAccessToken returns the access token from the request (either from header or cookie)
func (p *provider) GetAccessToken(gc *gin.Context) (string, error) {
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
func (p *provider) ValidateAccessToken(gc *gin.Context, accessToken string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	if accessToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	res, err := p.ParseJWTToken(accessToken)
	if err != nil {
		return res, err
	}

	userID := res["sub"].(string)
	nonce := res["nonce"].(string)

	// TODO: validate against existing Token
	// loginMethod := res["login_method"]
	// sessionKey := userID
	// if loginMethod != nil && loginMethod != "" {
	// 	sessionKey = loginMethod.(string) + ":" + userID
	// }

	// token, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce)
	// if nonce == "" || err != nil {
	// 	return res, fmt.Errorf(`unauthorized`)
	// }

	// if token != accessToken {
	// 	return res, fmt.Errorf(`unauthorized`)
	// }

	hostname := parsers.GetHost(gc)
	if ok, err := p.ValidateJWTClaims(res, &AuthTokenConfig{
		HostName: hostname,
		Nonce:    nonce,
		User:     &schemas.User{ID: userID},
	}); !ok || err != nil {
		return res, err
	}

	if res["token_type"] != constants.TokenTypeAccessToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}

// Function to validate refreshToken
func (p *provider) ValidateRefreshToken(gc *gin.Context, refreshToken string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	if refreshToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	res, err := p.ParseJWTToken(refreshToken)
	if err != nil {
		return res, err
	}

	userID := res["sub"].(string)
	nonce := res["nonce"].(string)

	// TODO: validate against existing token
	// loginMethod := res["login_method"]
	// sessionKey := userID
	// if loginMethod != nil && loginMethod != "" {
	// 	sessionKey = loginMethod.(string) + ":" + userID
	// }
	// token, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+nonce)
	// if nonce == "" || err != nil {
	// 	return res, fmt.Errorf(`unauthorized`)
	// }

	// if token != refreshToken {
	// 	return res, fmt.Errorf(`unauthorized`)
	// }

	hostname := parsers.GetHost(gc)
	if ok, err := p.ValidateJWTClaims(res, &AuthTokenConfig{
		HostName: hostname,
		Nonce:    nonce,
		User:     &schemas.User{ID: userID},
	}); !ok || err != nil {
		return res, err
	}

	if res["token_type"] != constants.TokenTypeRefreshToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}

func (p *provider) ValidateBrowserSession(gc *gin.Context, encryptedSession string) (*SessionData, error) {
	if encryptedSession == "" {
		return nil, fmt.Errorf(`unauthorized`)
	}

	decryptedFingerPrint, err := crypto.DecryptAES(p.config.ClientSecret, encryptedSession)
	if err != nil {
		return nil, err
	}

	var res SessionData
	err = json.Unmarshal([]byte(decryptedFingerPrint), &res)
	if err != nil {
		return nil, err
	}

	// TODO validate against saved token
	// sessionStoreKey := res.Subject
	// if res.LoginMethod != "" {
	// 	sessionStoreKey = res.LoginMethod + ":" + res.Subject
	// }
	// token, err := memorystore.Provider.GetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+res.Nonce)
	// if token == "" || err != nil {
	// 	log.Debugf("invalid browser session: %v, key: %s", err, sessionStoreKey+":"+constants.TokenTypeSessionToken+"_"+res.Nonce)
	// 	return nil, fmt.Errorf(`unauthorized`)
	// }

	// if encryptedSession != token {
	// 	return nil, fmt.Errorf(`unauthorized: invalid nonce`)
	// }

	if res.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf(`unauthorized: token expired`)
	}

	return &res, nil
}

// CreateIDToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
// For response_type (code) / authorization_code grant nonce should be empty
// for implicit flow it should be present to verify with actual state
func (p *provider) CreateIDToken(cfg *AuthTokenConfig) (string, int64, error) {
	expiryBound, err := utils.ParseDurationInSeconds(cfg.ExpireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	resUser := cfg.User.AsAPIUser()
	userBytes, _ := json.Marshal(&resUser)
	var userMap map[string]interface{}
	json.Unmarshal(userBytes, &userMap)

	customClaims := jwt.MapClaims{
		"iss":                 cfg.HostName,
		"aud":                 p.config.ClientID,
		"sub":                 cfg.User.ID,
		"exp":                 expiresAt,
		"iat":                 time.Now().Unix(),
		"token_type":          constants.TokenTypeIdentityToken,
		"allowed_roles":       strings.Split(cfg.User.Roles, ","),
		"login_method":        cfg.LoginMethod,
		p.config.JWTRoleClaim: cfg.Roles,
	}
	// split nonce to see if its authorization code grant method
	if cfg.CodeHash != "" {
		customClaims["at_hash"] = cfg.AtHash
		customClaims["c_hash"] = cfg.CodeHash
	} else {
		customClaims["nonce"] = cfg.Nonce
		customClaims["at_hash"] = cfg.Nonce
	}
	for k, v := range userMap {
		if k != "roles" {
			customClaims[k] = v
		}
	}
	// check for the extra access token script
	if p.config.CustomAccessTokenScript != "" {
		vm := otto.New()
		claimBytes, _ := json.Marshal(customClaims)
		vm.Run(fmt.Sprintf(`
			var user = %s;
			var tokenPayload = %s;
			var customFunction = %s;
			var functionRes = JSON.stringify(customFunction(user, tokenPayload));
		`, string(userBytes), string(claimBytes), p.config.CustomAccessTokenScript))

		val, err := vm.Get("functionRes")
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("error getting custom access token script")
		} else {
			extraPayload := make(map[string]interface{})
			err = json.Unmarshal([]byte(fmt.Sprintf("%v", val)), &extraPayload)
			if err != nil {
				p.dependencies.Log.Debug().Err(err).Msg("error converting accessTokenScript response to map")
			} else {
				for k, v := range extraPayload {
					customClaims[k] = v
				}
			}
		}
	}

	token, err := p.SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// GetIDToken returns the id token from the request header
func (p *provider) GetIDToken(gc *gin.Context) (string, error) {
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

// SessionOrAccessTokenData is a struct to hold session or access token data
type SessionOrAccessTokenData struct {
	UserID      string
	LoginMethod string
	Nonce       string
}

// GetUserIDFromSessionOrAccessToken returns the user id from the session or access token
func (p *provider) GetUserIDFromSessionOrAccessToken(gc *gin.Context) (*SessionOrAccessTokenData, error) {
	// First try to get the user id from the session
	isSession := true
	token, err := cookie.GetSession(gc)
	if err != nil || token == "" {
		p.dependencies.Log.Debug().Err(err).Msg("Failed to get session token")
		isSession = false
		token, err = p.GetAccessToken(gc)
		if err != nil || token == "" {
			p.dependencies.Log.Debug().Err(err).Msg("Failed to get access token")
			return nil, fmt.Errorf(`unauthorized`)
		}
	}
	if isSession {
		claims, err := p.ValidateBrowserSession(gc, token)
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Failed to validate session token")
			return nil, fmt.Errorf(`unauthorized`)
		}
		return &SessionOrAccessTokenData{
			UserID:      claims.Subject,
			LoginMethod: claims.LoginMethod,
			Nonce:       claims.Nonce,
		}, nil
	}
	// If not session, then validate the access token
	claims, err := p.ValidateAccessToken(gc, token)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Failed to validate access token")
		return nil, fmt.Errorf(`unauthorized`)
	}
	return &SessionOrAccessTokenData{
		UserID:      claims["sub"].(string),
		LoginMethod: claims["login_method"].(string),
		Nonce:       claims["nonce"].(string),
	}, nil
}
