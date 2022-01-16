package db

import (
	"log"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/constants"
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
	AddConfig(config Config) (Config, error)
	UpdateConfig(config Config) (Config, error)
	GetConfig() (Config, error)
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
	Config              string
}

var (
	IsORMSupported bool
	IsArangoDB     bool
	IsMongoDB      bool
	Mgr            Manager
	Prefix         = "authorizer_"
	Collections    = CollectionList{
		User:                Prefix + "users",
		VerificationRequest: Prefix + "verification_requests",
		Session:             Prefix + "sessions",
		Config:              Prefix + "config",
	}
)

func InitDB() {
	var sqlDB *gorm.DB
	var err error

	IsORMSupported = envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseType).(string) != constants.DbTypeArangodb && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseType).(string) != constants.DbTypeMongodb
	IsArangoDB = envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseType).(string) == constants.DbTypeArangodb
	IsMongoDB = envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseType).(string) == constants.DbTypeMongodb

	// sql db orm config
	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: Prefix,
		},
	}

	log.Println("db type:", envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseType).(string))

	switch envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseType).(string) {
	case constants.DbTypePostgres:
		sqlDB, err = gorm.Open(postgres.Open(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseURL).(string)), ormConfig)
		break
	case constants.DbTypeSqlite:
		sqlDB, err = gorm.Open(sqlite.Open(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseURL).(string)), ormConfig)
		break
	case constants.DbTypeMysql:
		sqlDB, err = gorm.Open(mysql.Open(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseURL).(string)), ormConfig)
		break
	case constants.DbTypeSqlserver:
		sqlDB, err = gorm.Open(sqlserver.Open(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDatabaseURL).(string)), ormConfig)
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
			sqlDB.AutoMigrate(&User{}, &VerificationRequest{}, &Session{}, &Config{})
		}
		Mgr = &manager{
			sqlDB:    sqlDB,
			arangodb: nil,
			mongodb:  nil,
		}
	}
}
