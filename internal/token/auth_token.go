package token

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/robertkrimen/otto"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ErrRefreshTokenReuse signals that a genuine, correctly-signed and
// client-bound refresh token was presented but no live session record backs
// it — i.e. it was already rotated away (or the session was logged out /
// expired) and is now being replayed. OAuth 2.1 §6.1 / RFC 9700 §4.14.2 treat
// this as a compromise signal: the caller MUST revoke the token family. It is
// returned instead of a plain unauthorized so the token endpoint can trigger
// that revocation. A malformed / foreign / wrong-type token yields a plain
// unauthorized (no side effect) — only a genuine reuse trips this.
var ErrRefreshTokenReuse = errors.New("refresh token reuse detected")

// reservedClaims are security-critical JWT claims that custom scripts must not override.
var reservedClaims = map[string]bool{
	"sub":           true,
	"iss":           true,
	"aud":           true,
	"exp":           true,
	"iat":           true,
	"token_type":    true,
	"roles":         true,
	"allowed_roles": true,
	"scope":         true,
	"nonce":         true,
	"login_method":  true,
	"at_hash":       true,
	"c_hash":        true,
	"auth_time":     true,
	"amr":           true,
	"acr":           true,
	// act (RFC 8693 §4.1) carries the delegation actor chain and MUST NOT be
	// forgeable by CustomAccessTokenScript — a script that could inject `act`
	// into a first-party user token would fabricate a delegation. client_id
	// (RFC 9068) identifies the actor and is likewise reserved.
	"act":       true,
	"client_id": true,
}

// AuthTokenConfig is the configuration for auth token
type AuthTokenConfig struct {
	LoginMethod string
	Nonce       string
	// OIDCNonce is the nonce value from the original OIDC /authorize
	// request. When set, CreateIDToken uses this for the id_token "nonce"
	// claim instead of Nonce. This separates the OIDC nonce (client-
	// provided, echoed back) from the internal session nonce (Nonce).
	OIDCNonce  string
	Code       string
	AtHash     string
	CodeHash   string
	ExpireTime string
	User       *schemas.User
	HostName   string
	Roles      []string
	Scope      []string
	// ServiceAccountID is set — and User is nil — for machine access tokens
	// issued via the client_credentials grant (RFC 6749 §4.4). When set,
	// CreateAccessToken builds a machine token whose `sub` is this id, whose
	// `scope` comes from Scope, and which carries NO roles/allowed_roles claim
	// (machines have no roles). LoginMethod must be set to
	// constants.AuthRecipeMethodServiceAccount so the token round-trips through
	// ValidateAccessToken on the existing stateful path.
	ServiceAccountID string
	// AuthTime is the Unix timestamp (seconds) at which the user
	// authenticated. OIDC Core §2 defines this as the `auth_time` ID
	// token claim. If zero, CreateIDToken falls back to time.Now() so
	// existing callers continue to work unchanged (backward compat).
	AuthTime int64
	// ClientID is the OAuth client the refresh token is being minted for —
	// the client that authenticated at /oauth/token when the token was
	// issued or last rotated. CreateRefreshToken embeds it as the
	// "client_id" claim so a later refresh_token redemption can be bound
	// to the same client (RFC 6749 §6). Empty is valid (client_credentials
	// / machine paths that never mint a refresh token).
	ClientID string
	// Resource is the RFC 8707 resource indicator the client bound to the
	// authorization request (the target MCP/API server). When non-empty it
	// becomes the ACCESS token's `aud` claim so the token cannot be replayed
	// at a different resource server (audience restriction). It intentionally
	// does NOT change the id_token or refresh_token audience: the id_token
	// audience is the client (OIDC), and the refresh_token audience is the
	// client too (RFC 6749 §6 client binding). Empty preserves existing
	// behavior — aud defaults to the requesting client / bootstrap client_id.
	Resource string
}

// loginMethodToAMR maps an internal LoginMethod value to the OIDC Core §2
// Authentication Methods Reference array. Returns nil (omit the claim)
// for unknown or empty methods.
func loginMethodToAMR(method string) []string {
	switch strings.ToLower(method) {
	case constants.AuthRecipeMethodBasicAuth, constants.AuthRecipeMethodMobileBasicAuth:
		return []string{"pwd"}
	case constants.AuthRecipeMethodMagicLinkLogin, constants.AuthRecipeMethodMobileOTP:
		return []string{"otp"}
	case constants.AuthRecipeMethodGoogle,
		constants.AuthRecipeMethodGithub,
		constants.AuthRecipeMethodFacebook,
		constants.AuthRecipeMethodLinkedIn,
		constants.AuthRecipeMethodApple,
		constants.AuthRecipeMethodDiscord,
		constants.AuthRecipeMethodTwitter,
		constants.AuthRecipeMethodTwitch,
		constants.AuthRecipeMethodRoblox,
		constants.AuthRecipeMethodMicrosoft:
		return []string{"fed"}
	}
	return nil
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

// SessionData holds the session claims persisted for a user session.
//
// IssuedAt is stamped fresh on every CreateSessionToken call, including
// silent rollovers of an already-authenticated session. AuthTime is the
// timestamp of the End-User's actual last authentication (OIDC Core §2's
// auth_time) and MUST survive rollovers unchanged — conflating the two
// previously caused auth_time and max_age staleness checks to reset on
// every silent /authorize call, defeating both.
type SessionData struct {
	Subject     string   `json:"sub"`
	Roles       []string `json:"roles"`
	Scope       []string `json:"scope"`
	Nonce       string   `json:"nonce"`
	IssuedAt    int64    `json:"iat"`
	AuthTime    int64    `json:"auth_time"`
	ExpiresAt   int64    `json:"exp"`
	LoginMethod string   `json:"login_method"`
}

// EffectiveAuthTime returns AuthTime, falling back to IssuedAt for session
// cookies minted before AuthTime existed (unmarshal leaves it at the zero
// value with no error).
func (sd *SessionData) EffectiveAuthTime() int64 {
	if sd.AuthTime != 0 {
		return sd.AuthTime
	}
	return sd.IssuedAt
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
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	authTime := cfg.AuthTime
	if authTime == 0 {
		authTime = time.Now().Unix()
	}
	fingerPrintMap := &SessionData{
		Nonce:       cfg.Nonce,
		Roles:       cfg.Roles,
		Subject:     cfg.User.ID,
		Scope:       cfg.Scope,
		LoginMethod: cfg.LoginMethod,
		IssuedAt:    time.Now().Unix(),
		AuthTime:    authTime,
		ExpiresAt:   expiresAt,
	}
	fingerPrintBytes, _ := json.Marshal(fingerPrintMap)
	fingerPrintHash, err := crypto.EncryptAES(p.config.ClientSecret, string(fingerPrintBytes))
	if err != nil {
		return nil, "", 0, err
	}

	return fingerPrintMap, fingerPrintHash, expiresAt, nil
}

// audience returns the OIDC "aud" claim value: the OAuth client the token is
// being issued to. cfg.ClientID carries the actual requesting client for the
// /authorize and /oauth/token flows; callers that mint a token for
// Authorizer's own hosted app (GraphQL login/signup, social/SAML/SSO
// callbacks establishing the browser session) leave it unset, and the
// reserved bootstrap client_id remains the audience — unchanged behavior.
func (p *provider) audience(cfg *AuthTokenConfig) string {
	if cfg.ClientID != "" {
		return cfg.ClientID
	}
	return p.config.ClientID
}

// accessTokenAudience returns the "aud" claim for an ACCESS token. When the
// client supplied an RFC 8707 resource indicator it is the audience — the
// token is bound to that resource server and is not replayable elsewhere.
// Otherwise it falls back to audience() (the requesting client), so callers
// that don't use resource indicators are unaffected. Only the access token
// uses this; id_token / refresh_token audiences remain the client.
func (p *provider) accessTokenAudience(cfg *AuthTokenConfig) string {
	if cfg.Resource != "" {
		return cfg.Resource
	}
	return p.audience(cfg)
}

// CreateRefreshToken util to create JWT token
func (p *provider) CreateRefreshToken(cfg *AuthTokenConfig) (string, int64, error) {
	// Lifetime is configurable via --refresh-token-expires-in (seconds).
	// Default 30 days when unset or non-positive.
	expirySeconds := p.config.RefreshTokenExpiresIn
	if expirySeconds <= 0 {
		expirySeconds = 60 * 60 * 24 * 30
	}
	expiryBound := time.Duration(expirySeconds) * time.Second
	expiresAt := time.Now().Add(expiryBound).Unix()
	authTime := cfg.AuthTime
	if authTime == 0 {
		authTime = time.Now().Unix()
	}
	customClaims := jwt.MapClaims{
		"iss":           cfg.HostName,
		"aud":           p.audience(cfg),
		"sub":           cfg.User.ID,
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"auth_time":     authTime,
		"token_type":    constants.TokenTypeRefreshToken,
		"roles":         cfg.Roles,
		"scope":         cfg.Scope,
		"nonce":         cfg.Nonce,
		"login_method":  cfg.LoginMethod,
		"allowed_roles": strings.Split(cfg.User.Roles, ","),
		"client_id":     cfg.ClientID,
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
	// Machine identity (client_credentials, RFC 6749 §4.4): there is no
	// resource owner, so cfg.User is nil. Building the human token below would
	// nil-deref on cfg.User.ID/Roles — route to the machine builder instead.
	// The human path below is left completely unchanged.
	if cfg.User == nil && cfg.ServiceAccountID != "" {
		return p.createMachineAccessToken(cfg)
	}
	expiryBound, err := utils.ParseDurationInSeconds(cfg.ExpireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	customClaims := jwt.MapClaims{
		"iss":           cfg.HostName,
		"aud":           p.accessTokenAudience(cfg),
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
		_ = json.Unmarshal(userBytes, &userMap)
		p.runCustomAccessTokenScript(userBytes, customClaims)
	}
	token, err := p.SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// createMachineAccessToken builds the OAuth2 access token JWT for a service
// account (client_credentials, RFC 6749 §4.4). It is the machine counterpart
// to the human path in CreateAccessToken: identical iss/aud/exp/iat/
// token_type/nonce shape, but `sub` is the service account id, `scope` carries
// the granted scopes, and there are NO roles/allowed_roles claims — machines
// have no roles. The CUSTOM_ACCESS_TOKEN_SCRIPT is intentionally not run: its
// contract is customFunction(user, tokenPayload) and there is no user.
// login_method is set (to constants.AuthRecipeMethodServiceAccount by the
// caller) so ValidateAccessToken derives the same memory-store session key the
// token endpoint registered this token under.
func (p *provider) createMachineAccessToken(cfg *AuthTokenConfig) (string, int64, error) {
	expiryBound, err := utils.ParseDurationInSeconds(cfg.ExpireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	customClaims := jwt.MapClaims{
		"iss":          cfg.HostName,
		"aud":          p.config.ClientID,
		"nonce":        cfg.Nonce,
		"sub":          cfg.ServiceAccountID,
		"exp":          expiresAt,
		"iat":          time.Now().Unix(),
		"token_type":   constants.TokenTypeAccessToken,
		"scope":        cfg.Scope,
		"login_method": cfg.LoginMethod,
	}
	token, err := p.SignJWTToken(customClaims)
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt, nil
}

// CreateMachineAuthToken issues a stateful OAuth2 access token for a service
// account (client_credentials, RFC 6749 §4.4). It returns ONLY an access token
// — no id_token, no refresh_token, no browser/session token — because machines
// have no OIDC identity and re-authenticate on expiry. The caller MUST register
// the returned token in the memory store exactly as human access tokens are
// (see the /oauth/token handler), or ValidateAccessToken will reject it.
func (p *provider) CreateMachineAuthToken(cfg *AuthTokenConfig) (*JWTToken, error) {
	accessToken, expiresAt, err := p.CreateAccessToken(cfg)
	if err != nil {
		return nil, err
	}
	return &JWTToken{Token: accessToken, ExpiresAt: expiresAt}, nil
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

	return authSplit[1], nil
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

	userID, ok := res["sub"].(string)
	if !ok || userID == "" {
		return res, fmt.Errorf(`unauthorized: missing sub claim`)
	}
	nonce, _ := res["nonce"].(string)

	loginMethod, _ := res["login_method"].(string)
	sessionKey := userID
	if loginMethod != "" {
		sessionKey = loginMethod + ":" + userID
	}

	token, err := p.dependencies.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce)
	if nonce == "" || err != nil {
		p.dependencies.Log.Debug().Err(err).Msgf("invalid access token: %v, key: %s", err, sessionKey+":"+constants.TokenTypeAccessToken+"_"+nonce)
		return res, fmt.Errorf(`unauthorized`)
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(accessToken)) != 1 {
		p.dependencies.Log.Debug().Msgf("invalid access token: %s, key: %s", err, sessionKey+":"+constants.TokenTypeAccessToken+"_"+nonce)
		return res, fmt.Errorf(`unauthorized`)
	}

	if p.userIsRevoked(gc, userID) {
		p.dependencies.Log.Debug().Str("user_id", userID).Msg("access token rejected: user revoked")
		return res, fmt.Errorf(`unauthorized: user revoked`)
	}

	// /userinfo and the generic session-or-access-token resolver present no
	// client credentials — a bearer access token is accepted from whichever
	// client it was issued to, there being no request-time client identity to
	// bind it to. Trust the token's own "aud" (set at issuance, protected by
	// the JWT signature) as the expected value; the checks above already
	// establish the token is a genuine, unexpired, unrevoked Authorizer token.
	aud, _ := res["aud"].(string)
	hostname := parsers.GetHost(gc)
	if ok, err := p.ValidateJWTClaims(res, &AuthTokenConfig{
		HostName: hostname,
		Nonce:    nonce,
		User:     &schemas.User{ID: userID},
		ClientID: aud,
	}); !ok || err != nil {
		return res, err
	}

	if res["token_type"] != constants.TokenTypeAccessToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}

// Function to validate refreshToken. expectedClientID is the OAuth client
// presenting the token at the token endpoint (RFC 6749 §6 client binding) —
// it must match the "aud" claim the token was issued with.
func (p *provider) ValidateRefreshToken(gc *gin.Context, refreshToken string, expectedClientID string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	if refreshToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	res, err := p.ParseJWTToken(refreshToken)
	if err != nil {
		return res, err
	}

	userID, ok := res["sub"].(string)
	if !ok || userID == "" {
		return res, fmt.Errorf(`unauthorized: missing sub claim`)
	}
	nonce, _ := res["nonce"].(string)

	loginMethod, _ := res["login_method"].(string)
	sessionKey := userID
	if loginMethod != "" {
		sessionKey = loginMethod + ":" + userID
	}
	token, err := p.dependencies.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+nonce)
	if nonce == "" || err != nil {
		p.dependencies.Log.Debug().Err(err).Msgf("invalid refresh token: %v, key: %s", err, sessionKey+":"+constants.TokenTypeRefreshToken+"_"+nonce)
		// Reuse detection (OAuth 2.1 §6.1 / RFC 9700 §4.14.2): ParseJWTToken
		// above already verified this token's signature, so it is genuinely
		// one we issued. If it carries a nonce and is a refresh token bound to
		// the presenting client, yet no live session record backs the nonce,
		// it was already rotated away and is being replayed — a compromise
		// signal. Surface it distinctly so the token endpoint revokes the
		// family. Anything else (missing nonce, wrong token_type, or a token
		// bound to a different client) is an ordinary unauthorized with no
		// side effect, so a forged/foreign token cannot force a revocation.
		if nonce != "" {
			tokenType, _ := res["token_type"].(string)
			aud, _ := res["aud"].(string)
			if tokenType == constants.TokenTypeRefreshToken && aud == expectedClientID {
				return res, ErrRefreshTokenReuse
			}
		}
		return res, fmt.Errorf(`unauthorized`)
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(refreshToken)) != 1 {
		p.dependencies.Log.Debug().Msgf("invalid refresh token: %s, key: %s", err, sessionKey+":"+constants.TokenTypeRefreshToken+"_"+nonce)
		return res, fmt.Errorf(`unauthorized`)
	}

	hostname := parsers.GetHost(gc)
	if ok, err := p.ValidateJWTClaims(res, &AuthTokenConfig{
		HostName: hostname,
		Nonce:    nonce,
		User:     &schemas.User{ID: userID},
		ClientID: expectedClientID,
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

	sessionStoreKey := res.Subject
	if res.LoginMethod != "" {
		sessionStoreKey = res.LoginMethod + ":" + res.Subject
	}
	token, err := p.dependencies.MemoryStoreProvider.GetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+res.Nonce)
	if token == "" || err != nil {
		p.dependencies.Log.Debug().Err(err).Msgf("invalid session token: %v, key: %s", err, sessionStoreKey+":"+constants.TokenTypeSessionToken+"_"+res.Nonce)
		return nil, fmt.Errorf(`unauthorized`)
	}

	if subtle.ConstantTimeCompare([]byte(encryptedSession), []byte(token)) != 1 {
		return nil, fmt.Errorf(`unauthorized: invalid nonce`)
	}

	if res.ExpiresAt <= time.Now().Unix() {
		return nil, fmt.Errorf(`unauthorized: token expired`)
	}

	if p.userIsRevoked(gc, res.Subject) {
		p.dependencies.Log.Debug().Str("user_id", res.Subject).Msg("browser session rejected: user revoked")
		return nil, fmt.Errorf(`unauthorized: user revoked`)
	}

	return &res, nil
}

// userIsRevoked re-checks the DB RevokedTimestamp for a user resolved from an
// already-issued access token or browser session. This is defense-in-depth:
// the session-store deletion SCIM deactivate() (and account deactivation)
// perform is the primary revocation mechanism for these stateful tokens, but
// if that delete was missed or failed on this instance, a held token would
// otherwise keep authenticating requests until its natural exp. Mirrors the
// same demote-only pattern used by introspect.go/token.go/login.go: a lookup
// failure never blocks a request that otherwise validated (fail open on DB
// errors so a transient storage blip can't take down every authenticated
// request), only a confirmed RevokedTimestamp does.
func (p *provider) userIsRevoked(gc *gin.Context, userID string) bool {
	if p.dependencies.StorageProvider == nil || userID == "" {
		return false
	}
	user, err := p.dependencies.StorageProvider.GetUserByID(gc, userID)
	if err != nil || user == nil {
		return false
	}
	return user.RevokedTimestamp != nil
}

// CreateIDToken util to create the OIDC ID token JWT, based on user
// information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT.
// See the in-function block comment for the at_hash / c_hash / nonce
// emission rules per OIDC Core §3.1.3.6 / §3.2.2.10.
func (p *provider) CreateIDToken(cfg *AuthTokenConfig) (string, int64, error) {
	expiryBound, err := utils.ParseDurationInSeconds(cfg.ExpireTime)
	if err != nil {
		expiryBound = time.Minute * 30
	}
	expiresAt := time.Now().Add(expiryBound).Unix()
	resUser := cfg.User.AsAPIUser()
	userBytes, _ := json.Marshal(&resUser)
	var userMap map[string]interface{}
	_ = json.Unmarshal(userBytes, &userMap)

	customClaims := jwt.MapClaims{
		"iss":                 cfg.HostName,
		"aud":                 p.audience(cfg),
		"sub":                 cfg.User.ID,
		"exp":                 expiresAt,
		"iat":                 time.Now().Unix(),
		"token_type":          constants.TokenTypeIdentityToken,
		"allowed_roles":       strings.Split(cfg.User.Roles, ","),
		"login_method":        cfg.LoginMethod,
		p.config.JWTRoleClaim: cfg.Roles,
	}
	// OIDC Core §3.1.3.6 / §3.2.2.10:
	//   at_hash REQUIRED whenever the response includes an access_token
	//           in the same flow. CreateAuthToken always issues an
	//           access_token, so cfg.AtHash is always populated.
	//   c_hash  REQUIRED only in hybrid flows that return both code
	//           and id_token. Set by the /authorize hybrid dispatch
	//           when cfg.Code is populated.
	//   nonce   MUST be echoed whenever the auth request supplied one,
	//           regardless of flow.
	if cfg.AtHash != "" {
		customClaims["at_hash"] = cfg.AtHash
	}
	if cfg.CodeHash != "" {
		customClaims["c_hash"] = cfg.CodeHash
	}
	// OIDC Core §3.1.3.3: the nonce claim MUST echo the value from the
	// original authorize request. OIDCNonce carries that value when the
	// token is issued via the token endpoint (code flow). For implicit
	// flows the caller sets Nonce directly.
	idTokenNonce := cfg.OIDCNonce
	if idTokenNonce == "" {
		idTokenNonce = cfg.Nonce
	}
	if idTokenNonce != "" {
		customClaims["nonce"] = idTokenNonce
	}
	// OIDC Core §2: auth_time — Unix seconds. Default to now if caller
	// did not supply a session-level auth timestamp (backward compat).
	authTime := cfg.AuthTime
	if authTime == 0 {
		authTime = time.Now().Unix()
	}
	customClaims["auth_time"] = authTime

	// OIDC Core §2: amr — Authentication Methods Reference array. Omit
	// the claim for unknown login methods rather than emit an empty array.
	if amr := loginMethodToAMR(cfg.LoginMethod); len(amr) > 0 {
		customClaims["amr"] = amr
	}

	// OIDC Core §2: acr — Authentication Context Class Reference.
	// Hardcoded "0" (no-op baseline per OIDC Core §2). MFA-aware ACR
	// alongside acr_values request support is a future enhancement;
	// for now returning "0" is safer than omitting the claim for
	// clients that require its presence.
	customClaims["acr"] = "0"
	for k, v := range userMap {
		if k != "roles" {
			customClaims[k] = v
		}
	}
	// check for the extra access token script
	if p.config.CustomAccessTokenScript != "" {
		p.runCustomAccessTokenScript(userBytes, customClaims)
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

	return authSplit[1], nil
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
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf(`unauthorized: missing sub claim`)
	}
	loginMethod, _ := claims["login_method"].(string)
	nonce, _ := claims["nonce"].(string)
	return &SessionOrAccessTokenData{
		UserID:      userID,
		LoginMethod: loginMethod,
		Nonce:       nonce,
	}, nil
}

const scriptTimeout = 5 * time.Second
const scriptTimeoutMsg = "script execution timeout: exceeded 5 seconds"

// runCustomAccessTokenScript executes the custom access token script in an Otto JS VM
// with a 5-second execution timeout to prevent CPU exhaustion from infinite loops.
func (p *provider) runCustomAccessTokenScript(userBytes []byte, customClaims jwt.MapClaims) {
	vm := otto.New()
	vm.Interrupt = make(chan func(), 1)

	// Start a goroutine that will interrupt the VM after the timeout
	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(scriptTimeout):
			vm.Interrupt <- func() {
				panic(scriptTimeoutMsg)
			}
		case <-done:
			// Script finished before timeout; goroutine exits cleanly
			return
		}
	}()

	defer func() {
		close(done)
		if caught := recover(); caught != nil {
			if msg, ok := caught.(string); ok && msg == scriptTimeoutMsg {
				p.dependencies.Log.Error().Msg("custom access token script timed out after 5 seconds")
			} else {
				panic(caught)
			}
		}
	}()

	claimBytes, _ := json.Marshal(customClaims)
	_, _ = vm.Run(fmt.Sprintf(`
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
				if !reservedClaims[k] {
					customClaims[k] = v
				}
			}
		}
	}
}
