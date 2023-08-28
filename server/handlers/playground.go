package handlers

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
)

// PlaygroundHandler is the handler for the /playground route
func PlaygroundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var h http.HandlerFunc

		disablePlayground, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePlayGround)
		if err != nil {
			log.Debug("error while getting disable playground value")
			return
		}

		// if env set to false, then check if logged in as super admin, if logged in then return graphql else 401 error
		// if env set to true, then disabled the playground with 404 error
		if !disablePlayground {
			if token.IsSuperAdmin(c) {
				h = playground.Handler("GraphQL", "/graphql")
			} else {
				log.Debug("not logged in as super admin")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in as super admin"})
				return
			}
		} else {
			log.Debug("playground is disabled")
			c.JSON(http.StatusNotFound, gin.H{"error": "playground is disabled"})
			return
		}
		h.ServeHTTP(c.Writer, c.Request)
	}
}
