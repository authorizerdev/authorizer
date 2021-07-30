package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO read query param = state which is base64 encoded
// make sure url is present in allowed origins
// set that in redirect_url
func AppHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Host:", c.Request.Host)
		// debug the request state
		if pusher := c.Writer.Pusher(); pusher != nil {
			// use pusher.Push() to do server push
			if err := pusher.Push("/app/build/bundle.js", nil); err != nil {
				log.Printf("Failed to push: %v", err)
			}
		}
		c.HTML(http.StatusOK, "app.tmpl", gin.H{
			"data": map[string]string{
				"domain":       c.Request.Host,
				"redirect_url": "http://localhost:8080/app",
			},
		})
	}
}
