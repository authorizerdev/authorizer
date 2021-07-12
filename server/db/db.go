package db

import (
	"log"

	"github.com/yauthdev/yauth/server/constants"
	"github.com/yauthdev/yauth/server/enum"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Manager interface {
	AddUser(user User) (User, error)
	GetUsers() ([]User, error)
	GetUserByEmail(email string) (User, error)
	AddVerification(verification Verification) (Verification, error)
}

type manager struct {
	db *gorm.DB
}

var Mgr Manager

func init() {
	var db *gorm.DB
	var err error
	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "yauth_",
		},
	}
	if constants.DB_TYPE == enum.Postgres {
		db, err = gorm.Open(postgres.Open(constants.DB_URL), ormConfig)
	}
	if constants.DB_TYPE == enum.Mysql {
		db, err = gorm.Open(mysql.Open(constants.DB_URL), ormConfig)
	}
	if constants.DB_TYPE == enum.Sqlite {
		db, err = gorm.Open(sqlite.Open(constants.DB_URL), ormConfig)
	}

	if err != nil {
		log.Fatal("Failed to init db:", err)
	} else {
		db.AutoMigrate(&User{}, &Verification{})
	}

	Mgr = &manager{db: db}
}
