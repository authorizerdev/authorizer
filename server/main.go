package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/routes"
	"github.com/authorizerdev/authorizer/server/sessionstore"
)

var VERSION string

type LogUTCFormatter struct {
	log.Formatter
}

func (u LogUTCFormatter) Format(e *log.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

func main() {
	envstore.ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	envstore.ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	envstore.ARG_ENV_FILE = flag.String("env_file", "", "Env file path")
	envstore.ARG_LOG_LEVEL = flag.String("log_level", "info", "Log level, possible values are debug,info,warn,error,fatal,panic")
	flag.Parse()

	// global log level
	logrus.SetFormatter(LogUTCFormatter{&logrus.JSONFormatter{}})
	logrus.SetReportCaller(true)

	// log instance for gin server
	log := logrus.New()
	log.SetFormatter(LogUTCFormatter{&logrus.JSONFormatter{}})
	log.SetReportCaller(true)

	var logLevel logrus.Level
	switch *envstore.ARG_LOG_LEVEL {
	case "debug":
		logLevel = logrus.DebugLevel
	case "info":
		logLevel = logrus.InfoLevel
	case "warn":
		logLevel = logrus.WarnLevel
	case "error":
		logLevel = logrus.ErrorLevel
	case "fatal":
		logLevel = logrus.FatalLevel
	case "panic":
		logLevel = logrus.PanicLevel
	default:
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
	log.SetLevel(logLevel)

	constants.VERSION = VERSION

	// initialize required envs (mainly db & env file path)
	err := env.InitRequiredEnv()
	if err != nil {
		log.Fatal("Error while initializing required envs: ", err)
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

	// initialize session store (redis or in-memory based on env)
	err = sessionstore.InitSession()
	if err != nil {
		log.Fatalln("Error while initializing session store: ", err)
	}

	// initialize oauth providers based on env
	err = oauth.InitOAuth()
	if err != nil {
		log.Fatalln("Error while initializing oauth: ", err)
	}

	router := routes.InitRouter(log)
	log.Info("Starting Authorizer: ", VERSION)
	router.Run(":" + envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyPort))
}
