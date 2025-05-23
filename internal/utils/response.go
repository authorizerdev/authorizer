package utils

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandleRedirectORJsonResponse handles the response based on redirectURL
func HandleRedirectORJsonResponse(c *gin.Context, httpResponse int, response map[string]interface{}, redirectURL string) {
	if strings.TrimSpace(redirectURL) == "" {
		c.JSON(httpResponse, response)
	} else {
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}
