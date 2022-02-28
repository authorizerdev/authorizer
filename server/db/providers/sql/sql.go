package sql

import (
	"log"
	"os"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type provider struct {
	db *gorm.DB
}

// NewProvider returns a new SQL provider
func NewProvider() (*provider, error) {
	var sqlDB *gorm.DB
	var err error
	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,         // Disable color
		},
	)

	ormConfig := &gorm.Config{
		Logger: customLogger,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: models.Prefix,
		},
	}
	switch envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) {
	case constants.DbTypePostgres, constants.DbTypeYugabyte:
		sqlDB, err = gorm.Open(postgres.Open(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
	case constants.DbTypeSqlite:
		sqlDB, err = gorm.Open(sqlite.Open(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
	case constants.DbTypeMysql, constants.DbTypeMariaDB:
		sqlDB, err = gorm.Open(mysql.Open(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
	case constants.DbTypeSqlserver:
		sqlDB, err = gorm.Open(sqlserver.Open(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)), ormConfig)
	}

	if err != nil {
		return nil, err
	}

	sqlDB.AutoMigrate(&models.User{}, &models.VerificationRequest{}, &models.Session{}, &models.Env{})
	return &provider{
		db: sqlDB,
	}, nil
}
