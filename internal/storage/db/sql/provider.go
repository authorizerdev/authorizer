package sql

import (
	libsql "github.com/ekristen/gorm-libsql"
	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the sql data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *gorm.DB
}

/**
Required to address the impact of the following code block:
const (
	phoneNumberIndexName  = "UQ_phone_number"
	phoneNumberColumnName = "phone_number"
)

type indexInfo struct {
	IndexName  string `json:"index_name"`
	ColumnName string `json:"column_name"`
}
**/

// NewProvider returns a new SQL provider
func NewProvider(
	config *config.Config,
	deps *Dependencies,
) (*provider, error) {
	var sqlDB *gorm.DB
	var err error

	ormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: schemas.Prefix,
		},
		AllowGlobalUpdate: true,
	}

	dbType := config.DatabaseType
	dbURL := config.DatabaseURL

	switch dbType {
	case constants.DbTypePostgres, constants.DbTypeYugabyte, constants.DbTypeCockroachDB:
		sqlDB, err = gorm.Open(postgres.Open(dbURL), ormConfig)
	case constants.DbTypeSqlite:
		sqlDB, err = gorm.Open(sqlite.Open(dbURL+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), ormConfig)
	case constants.DbTypeLibSQL:
		sqlDB, err = gorm.Open(libsql.Open(dbURL), ormConfig)
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
	if sqlDB.Migrator().HasConstraint(&schemas.User{}, "authorizer_users_phone_number_key") {
		err = sqlDB.Migrator().DropConstraint(&schemas.User{}, "authorizer_users_phone_number_key")
		if err != nil {
			deps.Log.Debug().Err(err).Msg("failed to drop unique constraint on phone_number")
		}
	}

	err = sqlDB.AutoMigrate(&schemas.User{}, &schemas.VerificationRequest{}, &schemas.Session{}, &schemas.Env{}, &schemas.Webhook{}, &schemas.WebhookLog{}, &schemas.EmailTemplate{}, &schemas.OTP{}, &schemas.Authenticator{}, &schemas.SessionToken{}, &schemas.MFASession{}, &schemas.OAuthState{})
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
		config:       config,
		dependencies: deps,
		db:           sqlDB,
	}, nil
}
