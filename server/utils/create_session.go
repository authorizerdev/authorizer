package utils

import (
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/gin-gonic/gin"
)

func CreateSession(userId string, c *gin.Context) {
	sessionData := db.Session{
		UserID:    userId,
		UserAgent: GetUserAgent(c.Request),
		IP:        GetIP(c.Request),
	}

	db.Mgr.AddSession(sessionData)
}
