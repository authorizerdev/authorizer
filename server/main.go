package main

import (
	"flag"

	"github.com/authorizerdev/authorizer/server/cli"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/logs"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/routes"
)

// VERSION is used to define the version of authorizer from build tags
var VERSION string

func main() {
	cli.ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	cli.ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	cli.ARG_ENV_FILE = flag.String("env_file", "", "Env file path")
	cli.ARG_LOG_LEVEL = flag.String("log_level", "", "Log level, possible values are debug,info,warn,error,fatal,panic")
	cli.ARG_REDIS_URL = flag.String("redis_url", "", "Redis connection string")
	flag.Parse()

	log := logs.InitLog(refs.StringValue(cli.ARG_LOG_LEVEL))

	constants.VERSION = VERSION

	// initialize required envs (mainly db, env file path and redis)
	err := memorystore.InitRequiredEnv()
	if err != nil {
		log.Fatal("Error while initializing required envs: ", err)
	}

	// initialize memory store
	err = memorystore.InitMemStore()
	if err != nil {
		log.Fatal("Error while initializing memory store: ", err)
	}

	// initialize db provider
	err = db.InitDB()
	if err != nil {
		log.Fatalln("Error while initializing db: ", err)
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
		log.Fatalln("Error while persisting env: ", err)
	}

	// initialize oauth providers based on env
	err = oauth.InitOAuth()
	if err != nil {
		log.Fatalln("Error while initializing oauth: ", err)
	}

	router := routes.InitRouter(log)
	log.Info("Starting Authorizer: ", VERSION)
	port, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyPort)
	log.Info("Authorizer running at PORT: ", port)
	if err != nil {
		log.Info("Error while getting port from env using default port 8080: ", err)
		port = "8080"
	}

	router.Run(":" + port)
}
