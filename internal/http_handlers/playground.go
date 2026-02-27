package http_handlers

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// PlaygroundHandler is the handler for the /playground route
func (h *httpProvider) PlaygroundHandler() gin.HandlerFunc {
	// log := h.Log.With().Str("func", "PlaygroundHandler").Logger()
	return func(c *gin.Context) {
		var handlerFunc http.HandlerFunc

		enablePlayground := h.Config.EnablePlayground

		// if env set to true, then check if logged in as super admin, if logged in then return graphql else 401 error
		// if env set to false, then disabled the playground with 404 error
		if enablePlayground {
			if h.TokenProvider.IsSuperAdmin(c) {
				handlerFunc = playground.Handler("GraphQL", "/graphql")
			} else {
				log.Debug().Msg("not logged in as super admin")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in as super admin"})
				return
			}
		} else {
			log.Debug().Msg("playground is disabled")
			c.JSON(http.StatusNotFound, gin.H{"error": "playground is disabled"})
			return
		}
		handlerFunc.ServeHTTP(c.Writer, c.Request)
	}
}
