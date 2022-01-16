package main

import (
	"flag"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/routes"
	"github.com/authorizerdev/authorizer/server/session"
)

var VERSION string

func main() {
	env.ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	env.ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	env.ARG_ENV_FILE = flag.String("env_file", "", "Env file path")
	flag.Parse()

	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.EnvKeyVersion, VERSION)

	env.InitEnv()
	db.InitDB()
	env.PersistEnv()

	session.InitSession()
	oauth.InitOAuth()

	router := routes.InitRouter()

	router.Run(":" + envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyPort).(string))
}
