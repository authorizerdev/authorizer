package test

import (
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJwt(t *testing.T) {
	// persist older data till test is done and then reset it
	jwtType, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	assert.Nil(t, err)
	publicKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey)
	assert.Nil(t, err)
	privateKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey)
	assert.Nil(t, err)
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	assert.Nil(t, err)
	nonce := uuid.New().String()
	hostname := "localhost"
	subject := "test"
	claims := jwt.MapClaims{
		"exp":   time.Now().Add(time.Minute * 30).Unix(),
		"iat":   time.Now().Unix(),
		"email": "test@yopmail.com",
		"sub":   subject,
		"aud":   clientID,
		"nonce": nonce,
		"iss":   hostname,
	}

	t.Run("invalid jwt type", func(t *testing.T) {
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "invalid")
		token, err := token.SignJWTToken(claims)
		assert.Error(t, err, "unsupported signing method")
		assert.Empty(t, token)
	})
	t.Run("expired jwt token", func(t *testing.T) {
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "HS256")
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtSecret, "test")
		expiredClaims := jwt.MapClaims{
			"exp":   time.Now().Add(-time.Minute * 30).Unix(),
			"iat":   time.Now().Unix(),
			"email": "test@yopmail.com",
		}
		jwtToken, err := token.SignJWTToken(expiredClaims)
		assert.NoError(t, err)
		_, err = token.ParseJWTToken(jwtToken, hostname, nonce, subject)
		assert.Error(t, err, err.Error(), "Token is expired")
	})
	t.Run("HMAC algorithms", func(t *testing.T) {
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtSecret, "test")
		t.Run("HS256", func(t *testing.T) {
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "HS256")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("HS384", func(t *testing.T) {
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "HS384")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("HS512", func(t *testing.T) {
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "HS512")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	t.Run("RSA algorithms", func(t *testing.T) {
		t.Run("RS256", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewRSAKey("RS256", clientID)
			assert.NoError(t, err)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "RS256")
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("RS384", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewRSAKey("RS384", clientID)
			assert.NoError(t, err)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "RS384")
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("RS512", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewRSAKey("RS512", clientID)
			assert.NoError(t, err)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "RS512")
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	t.Run("ECDSA algorithms", func(t *testing.T) {
		t.Run("ES256", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewECDSAKey("ES256", clientID)
			assert.NoError(t, err)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "ES256")
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("ES384", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewECDSAKey("ES384", clientID)
			assert.NoError(t, err)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "ES384")
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("ES512", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewECDSAKey("ES512", clientID)
			assert.NoError(t, err)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, "ES512")
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken, hostname, nonce, subject)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtType, jwtType)
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPublicKey, publicKey)
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJwtPrivateKey, privateKey)
}
