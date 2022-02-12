package test

import (
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

func TestJwt(t *testing.T) {
	claims := jwt.MapClaims{
		"exp":   time.Now().Add(time.Minute * 30).Unix(),
		"iat":   time.Now().Unix(),
		"email": "test@yopmail.com",
	}

	// persist older data till test is done and then reset it
	jwtType := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	jwtSecret := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)

	t.Run("invalid jwt type", func(t *testing.T) {
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "invalid")
		token, err := token.SignJWTToken(claims)
		assert.Error(t, err, "unsupported signing method")
		assert.Empty(t, token)
	})
	t.Run("expired jwt token", func(t *testing.T) {
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS256")
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
		t.Run("HS256", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS256")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("HS384", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS384")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("HS512", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "HS512")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	t.Run("RSA algorithms", func(t *testing.T) {
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, "-----BEGIN RSA PRIVATE KEY-----\nMIICWgIBAAKBgHUQac/v0f3c8m4L9BMWfxBiEzkdV5CoaqfxhO5IwAX/1cs0WceN\njM7g/qzC7YmEOSiYqupiRtsyn6riz0xT/VUg4uv1uZ/muC6EVfOjR5Ack3Brquql\nD+oMxN4CeA0Wzp2dEV4N3Gv7wWHdhg9ZSc4g6+ZUdlkhIPfeO9RNK9pPAgMBAAEC\ngYBqLrIbp0dNQn0vbm48ZhppDNys4L2NfAYKQZs23Aw5JN6Si/CnffBrsk+u+ryl\nEKcb+KaHJQ9qQdfsFAC+FizhMQy0Dq9yw6shnqHX+paB6E6z2/vX8ToPzJRwxBY3\nyuaetCEpSXR7pQEd5YWDTUH7qYnb9FObD+umhVvmlsTHCQJBALagPmexu0DvMXKZ\nWdplik6eXg9lptiuj5MYqitEUyzU9E9HNeHKlZM7szGeWG3jNduoKcyo4M0Flvt9\ncP+soVUCQQCkGOQ5Y3/GoZmclKWMVwqGdmL6wEjhNfg4PRfgUalHBif9Q1KnM8FP\nAvIqIH8bttRfyT185WmaM2gml0ApwF0TAkBVil9QoK4t7xvBKtUsd809n+481gc9\njR4Q70edtoYjBKhejeNOHF7NNPRtNFcFOZybg3v4sc2CGrEqoQoRp+F1AkBeLmMe\nhPrbF/jAI5h4WaSS0/OvExlBGOaj8Hx5pKTRPLlK5I7VpCC4pmoyv3/0ehSd/TQr\nMMhRVlvaeki7Lcq9AkBravJUadVCAIsB6oh03mo8gUFFFqXDyEl6BiJYqrjCQ5wd\nAQYJGbqQvgjPxN9+PTPldDNi6KVXntSg5gF/dA+Z\n-----END RSA PRIVATE KEY-----")
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, "-----BEGIN PUBLIC KEY-----\nMIGeMA0GCSqGSIb3DQEBAQUAA4GMADCBiAKBgHUQac/v0f3c8m4L9BMWfxBiEzkd\nV5CoaqfxhO5IwAX/1cs0WceNjM7g/qzC7YmEOSiYqupiRtsyn6riz0xT/VUg4uv1\nuZ/muC6EVfOjR5Ack3BrquqlD+oMxN4CeA0Wzp2dEV4N3Gv7wWHdhg9ZSc4g6+ZU\ndlkhIPfeO9RNK9pPAgMBAAE=\n-----END PUBLIC KEY-----")
		t.Run("RS256", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "RS256")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("RS384", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "RS384")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("RS512", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "RS512")
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
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgevZzL1gdAFr88hb2\nOF/2NxApJCzGCEDdfSp6VQO30hyhRANCAAQRWz+jn65BtOMvdyHKcvjBeBSDZH2r\n1RTwjmYSi9R/zpBnuQ4EiMnCqfMPWiZqB4QdbAd0E7oH50VpuZ1P087G\n-----END PRIVATE KEY-----")
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEEVs/o5+uQbTjL3chynL4wXgUg2R9\nq9UU8I5mEovUf86QZ7kOBIjJwqnzD1omageEHWwHdBO6B+dFabmdT9POxg==\n-----END PUBLIC KEY-----")
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "ES256")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("ES384", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, "-----BEGIN PRIVATE KEY-----\nMIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDCAHpFQ62QnGCEvYh/p\nE9QmR1C9aLcDItRbslbmhen/h1tt8AyMhskeenT+rAyyPhGhZANiAAQLW5ZJePZz\nMIPAxMtZXkEWbDF0zo9f2n4+T1h/2sh/fviblc/VTyrv10GEtIi5qiOy85Pf1RRw\n8lE5IPUWpgu553SteKigiKLUPeNpbqmYZUkWGh3MLfVzLmx85ii2vMU=\n-----END PRIVATE KEY-----")
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, "-----BEGIN PUBLIC KEY-----\nMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEC1uWSXj2czCDwMTLWV5BFmwxdM6PX9p+\nPk9Yf9rIf374m5XP1U8q79dBhLSIuaojsvOT39UUcPJROSD1FqYLued0rXiooIii\n1D3jaW6pmGVJFhodzC31cy5sfOYotrzF\n-----END PUBLIC KEY-----")
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "ES384")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
		t.Run("ES512", func(t *testing.T) {
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPrivateKey, "-----BEGIN PRIVATE KEY-----\nMIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIBiyAa7aRHFDCh2qga\n9sTUGINE5jHAFnmM8xWeT/uni5I4tNqhV5Xx0pDrmCV9mbroFtfEa0XVfKuMAxxf\nZ6LM/yKhgYkDgYYABAGBzgdnP798FsLuWYTDDQA7c0r3BVk8NnRUSexpQUsRilPN\nv3SchO0lRw9Ru86x1khnVDx+duq4BiDFcvlSAcyjLACJvjvoyTLJiA+TQFdmrear\njMiZNE25pT2yWP1NUndJxPcvVtfBW48kPOmvkY4WlqP5bAwCXwbsKrCgk6xbsp12\new==\n-----END PRIVATE KEY-----")
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtPublicKey, "-----BEGIN PUBLIC KEY-----\nMIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBgc4HZz+/fBbC7lmEww0AO3NK9wVZ\nPDZ0VEnsaUFLEYpTzb90nITtJUcPUbvOsdZIZ1Q8fnbquAYgxXL5UgHMoywAib47\n6MkyyYgPk0BXZq3mq4zImTRNuaU9slj9TVJ3ScT3L1bXwVuPJDzpr5GOFpaj+WwM\nAl8G7CqwoJOsW7Kddns=\n-----END PUBLIC KEY-----")
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, "ES512")
			jwtToken, err := token.SignJWTToken(claims)
			assert.NoError(t, err)
			assert.NotEmpty(t, jwtToken)
			c, err := token.ParseJWTToken(jwtToken)
			assert.NoError(t, err)
			assert.Equal(t, c["email"].(string), claims["email"])
		})
	})

	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtType, jwtType)
	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJwtSecret, jwtSecret)
}
