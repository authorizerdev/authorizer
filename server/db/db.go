package db

import (
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
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
	UpdateVerificationTime(verifiedAt int64, id uint) error
	AddVerification(verification VerificationRequest) (VerificationRequest, error)
	GetVerificationByToken(token string) (VerificationRequest, error)
	DeleteToken(email string) error
	GetVerificationRequests() ([]VerificationRequest, error)
	GetVerificationByEmail(email string) (VerificationRequest, error)
	DeleteUser(email string) error
}

type manager struct {
	db *gorm.DB
}

var Mgr Manager

func InitDB() {
	var db *gorm.DB
	var err error
	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "authorizer_",
		},
	}
	if constants.DATABASE_TYPE == enum.Postgres.String() {
		db, err = gorm.Open(postgres.Open(constants.DATABASE_URL), ormConfig)
	}
	if constants.DATABASE_TYPE == enum.Mysql.String() {
		db, err = gorm.Open(mysql.Open(constants.DATABASE_URL), ormConfig)
	}
	if constants.DATABASE_TYPE == enum.Sqlite.String() {
		db, err = gorm.Open(sqlite.Open(constants.DATABASE_URL), ormConfig)
	}

	if err != nil {
		log.Fatal("Failed to init db:", err)
	} else {
		db.AutoMigrate(&User{}, &VerificationRequest{})
	}

	Mgr = &manager{db: db}
}
