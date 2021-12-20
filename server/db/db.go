package db

import (
	"log"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
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
	GetVerificationByEmail(email string) (VerificationRequest, error)
	AddSession(session Session) error
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
	}
)

func InitDB() {
	var sqlDB *gorm.DB
	var err error

	IsORMSupported = constants.DATABASE_TYPE != enum.Arangodb.String() && constants.DATABASE_TYPE != enum.Mongodb.String()
	IsArangoDB = constants.DATABASE_TYPE == enum.Arangodb.String()
	IsMongoDB = constants.DATABASE_TYPE == enum.Mongodb.String()

	// sql db orm config
	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: Prefix,
		},
	}

	log.Println("db type:", constants.DATABASE_TYPE)

	switch constants.DATABASE_TYPE {
	case enum.Postgres.String():
		sqlDB, err = gorm.Open(postgres.Open(constants.DATABASE_URL), ormConfig)
		break
	case enum.Sqlite.String():
		sqlDB, err = gorm.Open(sqlite.Open(constants.DATABASE_URL), ormConfig)
		break
	case enum.Mysql.String():
		sqlDB, err = gorm.Open(mysql.Open(constants.DATABASE_URL), ormConfig)
		break
	case enum.SQLServer.String():
		sqlDB, err = gorm.Open(sqlserver.Open(constants.DATABASE_URL), ormConfig)
		break
	case enum.Arangodb.String():
		arangodb, err := initArangodb()
		if err != nil {
			log.Fatal("error initializing arangodb:", err)
		}

		Mgr = &manager{
			sqlDB:    nil,
			mongodb:  nil,
			arangodb: arangodb,
		}

		break
	case enum.Mongodb.String():
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
			sqlDB.AutoMigrate(&User{}, &VerificationRequest{}, &Session{})
		}
		Mgr = &manager{
			sqlDB:    sqlDB,
			arangodb: nil,
			mongodb:  nil,
		}
	}
}
