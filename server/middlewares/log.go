package middlewares

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/utils"
)

// GinLogWriteFunc convert func to io.Writer.
type GinLogWriteFunc func([]byte) (int, error)

// GinLog Write function
func (fn GinLogWriteFunc) Write(data []byte) (int, error) {
	return fn(data)
}

// NewGinLogrusWrite logrus writer for gin
func NewGinLogrusWrite() io.Writer {
	return GinLogWriteFunc(func(data []byte) (int, error) {
		log.Info("%s", data)
		return 0, nil
	})
}

// JSONLogMiddleware logs a gin HTTP request in JSON format, with some additional custom key/values
func JSONLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process Request
		c.Next()

		// Stop timer
		duration := utils.GetDurationInMillseconds(start)

		entry := log.WithFields(log.Fields{
			"client_ip":          utils.GetIP(c.Request),
			"duration":           fmt.Sprintf("%.2f", duration),
			"method":             c.Request.Method,
			"path":               c.Request.RequestURI,
			"status":             c.Writer.Status(),
			"referrer":           c.Request.Referer(),
			"request_id":         c.Writer.Header().Get("Request-Id"),
			"authorizer_version": constants.VERSION,
		})

		if c.Writer.Status() >= 500 {
			entry.Error(c.Errors.String())
		} else {
			entry.Info("")
		}
	}
}
