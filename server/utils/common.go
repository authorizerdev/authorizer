package utils

import (
	"log"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
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
// Do not store token here as that could be security breach
func SaveSessionInDB(userId string, c *gin.Context) {
	sessionData := models.Session{
		UserID:    userId,
		UserAgent: GetUserAgent(c.Request),
		IP:        GetIP(c.Request),
	}

	err := db.Provider.AddSession(sessionData)
	if err != nil {
		log.Println("=> error saving session in db:", err)
	} else {
		log.Println("=> session saved in db:", sessionData)
	}
}
