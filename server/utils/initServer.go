package utils

import (
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
)

// any jobs that we want to run at start of server can be executed here

// 1. create roles table and add the roles list from env to table

func InitServer() {
	roles := []db.Role{}
	for _, val := range constants.ROLES {
		roles = append(roles, db.Role{
			Role: val,
		})
	}
	for _, val := range constants.PROTECTED_ROLES {
		roles = append(roles, db.Role{
			Role: val,
		})
	}
	err := db.Mgr.SaveRoles(roles)
	if err != nil {
		log.Println(`Error saving roles`, err)
	}
}
