package db

import (
	"log"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/providers"
	"github.com/authorizerdev/authorizer/server/db/providers/arangodb"
	"github.com/authorizerdev/authorizer/server/db/providers/mongodb"
	"github.com/authorizerdev/authorizer/server/db/providers/sql"
	"github.com/authorizerdev/authorizer/server/envstore"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Manager interface {
	AddUser(user User) (User, error)
	UpdateUser(user User) (User, error)
	DeleteUser(user User) error
	GetUsers() ([]User, error)
	GetUserByEmail(email string) (User, error)
	GetUserByID(email string) (User, error)
	AddVerification(verification VerificationRequest) (VerificationRequest, error)
	GetVerificationByToken(token string) (VerificationRequest, error)
	DeleteVerificationRequest(verificationRequest VerificationRequest) error
	GetVerificationRequests() ([]VerificationRequest, error)
	GetVerificationByEmail(email string, identifier string) (VerificationRequest, error)
	AddSession(session Session) error
	DeleteUserSession(userId string) error
	AddEnv(env Env) (Env, error)
	UpdateEnv(env Env) (Env, error)
	GetEnv() (Env, error)
}

type manager struct {
	sqlDB    *gorm.DB
	arangodb arangoDriver.Database
	mongodb  *mongo.Database
}

// mainly used by nosql dbs
type CollectionList struct {
	User                string
	VerificationRequest string
	Session             string
	Env                 string
}

var (
	IsORMSupported bool
	IsArangoDB     bool
	IsMongoDB      bool
	Mgr            Manager
	Provider       providers.Provider
	Prefix         = "authorizer_"
	Collections    = CollectionList{
		User:                Prefix + "users",
		VerificationRequest: Prefix + "verification_requests",
		Session:             Prefix + "sessions",
		Env:                 Prefix + "env",
	}
)

func InitDB() {
	var sqlDB *gorm.DB
	var err error

	IsORMSupported = envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeArangodb && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeMongodb
	IsArangoDB = envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeArangodb
	IsMongoDB = envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeMongodb

	// sql db orm config
	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: Prefix,
		},
	}

	if IsORMSupported {
		Provider, err = sql.NewProvider()
		if err != nil {
			log.Println("=> error setting sql provider:", err)
		}
	}

	if IsArangoDB {
		Provider, err = arangodb.NewProvider()
		if err != nil {
			log.Println("=> error setting arangodb provider:", err)
		}
	}

	if IsMongoDB {
		Provider, err = mongodb.NewProvider()
		if err != nil {
			log.Println("=> error setting arangodb provider:", err)
		}
	}

	log.Println("db type:", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType))

	switch envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) {
	case constants.DbTypePostgres:
		sqlDB, err = gorm.Open(postgres.Open(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
		break
	case constants.DbTypeSqlite:
		sqlDB, err = gorm.Open(sqlite.Open(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
		break
	case constants.DbTypeMysql:
		sqlDB, err = gorm.Open(mysql.Open(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
		break
	case constants.DbTypeSqlserver:
		sqlDB, err = gorm.Open(sqlserver.Open(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
		break
	case constants.DbTypeArangodb:
		arangodb, err := initArangodb()
		if err != nil {
			log.Fatal("error initializing arangodb:", err)
		}

		Mgr = &manager{
			sqlDB:    nil,
			arangodb: arangodb,
			mongodb:  nil,
		}

		break
	case constants.DbTypeMongodb:
		mongodb, err := initMongodb()
		if err != nil {
			log.Fatal("error initializing mongodb connection:", err)
		}

		Mgr = &manager{
			sqlDB:    nil,
			arangodb: nil,
			mongodb:  mongodb,
		}
	}

	// common for all sql dbs that are configured via go-orm
	if IsORMSupported {
		if err != nil {
			log.Fatal("Failed to init sqlDB:", err)
		} else {
			sqlDB.AutoMigrate(&User{}, &VerificationRequest{}, &Session{}, &Env{})
		}
		Mgr = &manager{
			sqlDB:    sqlDB,
			arangodb: nil,
			mongodb:  nil,
		}
	}
}
