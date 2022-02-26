package main

import (
	"flag"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/routes"
	"github.com/authorizerdev/authorizer/server/sessionstore"
)

var VERSION string

func main() {
	envstore.ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	envstore.ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	envstore.ARG_ENV_FILE = flag.String("env_file", "", "Env file path")
	flag.Parse()

	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyVersion, VERSION)

	// initialize required envs (mainly db & env file path)
	err := env.InitRequiredEnv()
	if err != nil {
		log.Fatal("Error while initializing required envs:", err)
	}

	// initialize db provider
	err = db.InitDB()
	if err != nil {
		log.Fatalln("Error while initializing db:", err)
	}

	// initialize all envs
	// (get if present from db else construct from os env + defaults)
	err = env.InitAllEnv()
	if err != nil {
		log.Fatalln("Error while initializing env: ", err)
	}

	// persist all envs
	err = env.PersistEnv()
	if err != nil {
		log.Fatalln("Error while persisting env:", err)
	}

	// initialize session store (redis or in-memory based on env)
	err = sessionstore.InitSession()
	if err != nil {
		log.Fatalln("Error while initializing session store:", err)
	}

	// initialize oauth providers based on env
	err = oauth.InitOAuth()
	if err != nil {
		log.Fatalln("Error while initializing oauth:", err)
	}

	router := routes.InitRouter()
	router.Run(":" + envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyPort))
}
