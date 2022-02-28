package test

import (
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

func TestJwt(t *testing.T) {
	// persist older data till test is done and then reset it
	jwtType := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	publicKey := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey)
	privateKey := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey)
	clientID := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	claims := jwt.MapClaims{
		"exp":   time.Now().Add(time.Minute * 30).Unix(),
		"iat":   time.Now().Unix(),
		"email": "test@yopmail.com",
		"sub":   "test",
		"aud":   clientID,
	}

	t.Run("invalid jwt type", func(t *testing.T) {
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "invalid")
		token, err := token.SignJWTToken(claims)
		assert.Error(t, err, "unsupported signing method")
		assert.Empty(t, token)
	})
	t.Run("expired jwt token", func(t *testing.T) {
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS256")
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtSecret, "test")
		expiredClaims := jwt.MapClaims{
			"exp":   time.Now().Add(-time.Minute * 30).Unix(),
			"iat":   time.Now().Unix(),
			"email": "test@yopmail.com",
		}
		jwtToken, err := token.SignJWTToken(expiredClaims)
		assert.NoError(t, err)
		_, err = token.ParseJWTToken(jwtToken)
		assert.Error(t, err, err.Error(), "Token is expired")
	})
	t.Run("HMAC algorithms", func(t *testing.T) {
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtSecret, "test")
		t.Run("HS256", func(t *testing.T) {
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS256")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("HS384", func(t *testing.T) {
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS384")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("HS512", func(t *testing.T) {
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS512")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	t.Run("RSA algorithms", func(t *testing.T) {
		t.Run("RS256", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewRSAKey("RS256", clientID)
			assert.NoError(t, err)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "RS256")
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("RS384", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewRSAKey("RS384", clientID)
			assert.NoError(t, err)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "RS384")
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("RS512", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewRSAKey("RS512", clientID)
			assert.NoError(t, err)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "RS512")
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	t.Run("ECDSA algorithms", func(t *testing.T) {
		t.Run("ES256", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewECDSAKey("ES256", clientID)
			assert.NoError(t, err)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "ES256")
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("ES384", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewECDSAKey("ES384", clientID)
			assert.NoError(t, err)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "ES384")
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("ES512", func(t *testing.T) {
			_, privateKey, publickKey, _, err := crypto.NewECDSAKey("ES512", clientID)
			assert.NoError(t, err)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "ES512")
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
			envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publickKey)
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, jwtType)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, publicKey)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, privateKey)
}
