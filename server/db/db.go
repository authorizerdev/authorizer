package db

import (
	"log"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Manager interface {
	SaveUser(user User) (User, error)
	UpdateUser(user User) (User, error)
	GetUsers() ([]User, error)
	GetUserByEmail(email string) (User, error)
	GetUserByID(email string) (User, error)
	UpdateVerificationTime(verifiedAt int64, id uuid.UUID) error
	AddVerification(verification VerificationRequest) (VerificationRequest, error)
	GetVerificationByToken(token string) (VerificationRequest, error)
	DeleteToken(email string) error
	GetVerificationRequests() ([]VerificationRequest, error)
	GetVerificationByEmail(email string) (VerificationRequest, error)
	DeleteUser(email string) error
	SaveRoles(roles []Role) error
	SaveSession(session Session) error
}

type manager struct {
	sqlDB    *gorm.DB
	arangodb *arangoDriver.Database
}

// mainly used by nosql dbs
type CollectionList struct {
	User                string
	VerificationRequest string
	Role                string
	Session             string
}

var (
	Mgr         Manager
	Prefix      = "authorizer_"
	Collections = CollectionList{
		User:                Prefix + "users",
		VerificationRequest: Prefix + "verification_requests",
		Role:                Prefix + "roles",
		Session:             Prefix + "sessions",
	}
)

func isSQL() bool {
	return constants.DATABASE_TYPE != enum.Arangodb.String()
}

func InitDB() {
	var sqlDB *gorm.DB
	var err error

	// sql db orm config
	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: Prefix,
		},
	}

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
	case enum.Arangodb.String():
		arangodb, err := initArangodb()
		if err != nil {
			log.Fatal("error initing arangodb:", err)
		}
		Mgr = &manager{
			sqlDB:    nil,
			arangodb: arangodb,
		}

		// check if collections exists

		break
	}

	// common for all sql dbs that are configured via gorm
	if isSQL() {
		if err != nil {
			log.Fatal("Failed to init sqlDB:", err)
		} else {
			sqlDB.AutoMigrate(&User{}, &VerificationRequest{}, &Role{}, &Session{})
		}
		Mgr = &manager{
			sqlDB:    sqlDB,
			arangodb: nil,
		}
	}
}
