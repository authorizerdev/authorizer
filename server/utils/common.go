package utils

import (
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/gin-gonic/gin"
)

// StringSliceContains checks if a string slice contains a particular string
func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// SaveSessionInDB saves sessions generated for a given user with meta information
// Not store token here as that could be security breach
func SaveSessionInDB(userId string, c *gin.Context) {
	sessionData := db.Session{
		UserID:    userId,
		UserAgent: GetUserAgent(c.Request),
		IP:        GetIP(c.Request),
	}

	db.Mgr.AddSession(sessionData)
}
