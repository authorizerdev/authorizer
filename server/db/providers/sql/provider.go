package sql

import (
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type provider struct {
	db *gorm.DB
}

const (
	phoneNumberIndexName  = "UQ_phone_number"
	phoneNumberColumnName = "phone_number"
)

type indexInfo struct {
	IndexName  string `json:"index_name"`
	ColumnName string `json:"column_name"`
}

// NewProvider returns a new SQL provider
func NewProvider() (*provider, error) {
	var sqlDB *gorm.DB
	var err error
	customLogger := logger.New(
		logrus.StandardLogger(),
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)

	ormConfig := &gorm.Config{
		Logger: customLogger,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: models.Prefix,
		},
		AllowGlobalUpdate: true,
	}

	dbType := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseType
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL

	switch dbType {
	case constants.DbTypePostgres, constants.DbTypeYugabyte, constants.DbTypeCockroachDB:
		sqlDB, err = gorm.Open(postgres.Open(dbURL), ormConfig)
	case constants.DbTypeSqlite:
		sqlDB, err = gorm.Open(sqlite.Open(dbURL+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), ormConfig)
	case constants.DbTypeMysql, constants.DbTypeMariaDB, constants.DbTypePlanetScaleDB:
		sqlDB, err = gorm.Open(mysql.Open(dbURL), ormConfig)
	case constants.DbTypeSqlserver:
		sqlDB, err = gorm.Open(sqlserver.Open(dbURL), ormConfig)
	}

	if err != nil {
		return nil, err
	}

	// For sqlserver, handle uniqueness of phone_number manually via extra db call
	// during create and update mutation.
	if sqlDB.Migrator().HasConstraint(&models.User{}, "authorizer_users_phone_number_key") {
		err = sqlDB.Migrator().DropConstraint(&models.User{}, "authorizer_users_phone_number_key")
		logrus.Debug("Failed to drop phone number constraint:", err)
	}

	err = sqlDB.AutoMigrate(&models.User{}, &models.VerificationRequest{}, &models.Session{}, &models.Env{}, &models.Webhook{}, models.WebhookLog{}, models.EmailTemplate{}, &models.OTP{}, &models.SMSVerificationRequest{})
	if err != nil {
		return nil, err
	}

	// IMPACT: Request user to manually delete: UQ_phone_number constraint
	// unique constraint on phone number does not work with multiple null values for sqlserver
	// for more information check https://stackoverflow.com/a/767702
	// if dbType == constants.DbTypeSqlserver {
	// 	var indexInfos []indexInfo
	// 	// remove index on phone number if present with different name
	// 	res := sqlDB.Raw("SELECT i.name AS index_name, i.type_desc AS index_algorithm, CASE i.is_unique WHEN 1 THEN 'TRUE' ELSE 'FALSE' END AS is_unique, ac.Name AS column_name FROM sys.tables AS t INNER JOIN sys.indexes AS i ON t.object_id = i.object_id INNER JOIN sys.index_columns AS ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id INNER JOIN sys.all_columns AS ac ON ic.object_id = ac.object_id AND ic.column_id = ac.column_id WHERE t.name = 'authorizer_users' AND SCHEMA_NAME(t.schema_id) = 'dbo';").Scan(&indexInfos)
	// 	if res.Error != nil {
	// 		return nil, res.Error
	// 	}

	// 	for _, val := range indexInfos {
	// 		if val.ColumnName == phoneNumberColumnName && val.IndexName != phoneNumberIndexName {
	// 			// drop index & create new
	// 			if res := sqlDB.Exec(fmt.Sprintf(`ALTER TABLE authorizer_users DROP CONSTRAINT "%s";`, val.IndexName)); res.Error != nil {
	// 				return nil, res.Error
	// 			}

	// 			// create index
	// 			if res := sqlDB.Exec(fmt.Sprintf("CREATE UNIQUE NONCLUSTERED INDEX %s ON authorizer_users(phone_number) WHERE phone_number IS NOT NULL;", phoneNumberIndexName)); res.Error != nil {
	// 				return nil, res.Error
	// 			}
	// 		}
	// 	}
	// }

	return &provider{
		db: sqlDB,
	}, nil
}
