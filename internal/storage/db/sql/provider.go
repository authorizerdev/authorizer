package sql

import (
	libsql "github.com/ekristen/gorm-libsql"
	"github.com/rs/zerolog"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	sqlite "github.com/authorizerdev/authorizer/internal/storage/db/sql/sqlitedialect"
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
		AllowGlobalUpdate: false,
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

	// v1 declared the email and phone_number columns of authorizer_users and
	// authorizer_otps as UNIQUE. v2 keeps plain (non-unique) indexes and enforces
	// uniqueness in the application layer (see AddUser/UpdateUser) so behaviour is
	// identical across all supported databases and these fields can stay optional
	// (email-only or phone-only signups).
	//
	// On an upgraded SQL database those columns are still unique, so GORM
	// AutoMigrate tries to DROP CONSTRAINT "uni_<table>_<col>" — a name that does
	// not match what the database actually created — failing with SQLSTATE 42704
	// ("constraint does not exist") and aborting the whole migration. Depending on
	// the v1 release, the old uniqueness exists either as a CONSTRAINT
	// (<table>_<col>_key) or as a standalone UNIQUE INDEX (idx_<table>_<col>), so
	// we clear both forms before AutoMigrate. Non-fatal: fresh installs have
	// nothing to drop.
	staleUniqueColumns := []struct {
		model      any
		table, col string
	}{
		{&schemas.User{}, "authorizer_users", "email"},
		{&schemas.User{}, "authorizer_users", "phone_number"},
		{&schemas.OTP{}, "authorizer_otps", "email"},
		{&schemas.OTP{}, "authorizer_otps", "phone_number"},
	}

	// 1) Drop stale unique CONSTRAINTs by their well-known names. A unique
	//    constraint's backing index can't be removed with DROP INDEX, so it has
	//    to go via DROP CONSTRAINT — and GORM's GetIndexes does not surface
	//    constraint-backed indexes, so step 2 below would miss these. The three
	//    names cover every form seen in the field:
	//      "<table>_<col>_key"   Postgres/MySQL default for a gorm:"unique" tag
	//      "uni_<table>_<col>"   GORM's default unique-constraint name
	//      "idx_<table>_<col>"   a v1 gorm:"uniqueIndex" that the DB promoted to
	//                            a UNIQUE constraint of the same name
	for _, c := range staleUniqueColumns {
		for _, name := range []string{
			c.table + "_" + c.col + "_key",
			"uni_" + c.table + "_" + c.col,
			"idx_" + c.table + "_" + c.col,
		} {
			if sqlDB.Migrator().HasConstraint(c.model, name) {
				if dropErr := sqlDB.Migrator().DropConstraint(c.model, name); dropErr != nil {
					deps.Log.Debug().Err(dropErr).Str("constraint", name).Msg("failed to drop stale unique constraint")
				}
			}
		}
	}

	// 2) Drop any remaining stale UNIQUE index on the same columns, name-agnostic.
	//    Only single-column UNIQUE indexes on email / phone_number are removed,
	//    so the current schema's non-unique indexes are left intact and
	//    AutoMigrate recreates anything it needs as non-unique. The discovered
	//    object may be either a standalone unique index (a v1 gorm:"uniqueIndex"
	//    stored as `CREATE UNIQUE INDEX idx_<table>_<col>`) or a UNIQUE
	//    constraint whose backing index shares that name — the latter cannot be
	//    dropped with DROP INDEX ("constraint ... requires it"), so it must go
	//    via DROP CONSTRAINT. Pick the right one.
	uniqueColTargets := map[string]bool{"email": true, "phone_number": true}
	for _, model := range []any{&schemas.User{}, &schemas.OTP{}} {
		indexes, idxErr := sqlDB.Migrator().GetIndexes(model)
		if idxErr != nil {
			deps.Log.Debug().Err(idxErr).Msg("failed to list indexes while clearing stale unique indexes")
			continue
		}
		for _, idx := range indexes {
			unique, ok := idx.Unique()
			if !ok || !unique {
				continue
			}
			cols := idx.Columns()
			if len(cols) != 1 || !uniqueColTargets[cols[0]] {
				continue
			}
			name := idx.Name()
			if sqlDB.Migrator().HasConstraint(model, name) {
				if dropErr := sqlDB.Migrator().DropConstraint(model, name); dropErr != nil {
					deps.Log.Debug().Err(dropErr).Str("constraint", name).Msg("failed to drop stale unique constraint")
				}
			} else if dropErr := sqlDB.Migrator().DropIndex(model, name); dropErr != nil {
				deps.Log.Debug().Err(dropErr).Str("index", name).Msg("failed to drop stale unique index")
			}
		}
	}

	err = sqlDB.AutoMigrate(&schemas.User{}, &schemas.VerificationRequest{}, &schemas.Session{}, &schemas.Env{}, &schemas.Webhook{}, &schemas.WebhookLog{}, &schemas.EmailTemplate{}, &schemas.OTP{}, &schemas.Authenticator{}, &schemas.SessionToken{}, &schemas.MFASession{}, &schemas.OAuthState{}, &schemas.AuditLog{})
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

// Close closes the underlying SQL connection pool.
func (p *provider) Close() error {
	if p.db == nil {
		return nil
	}
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
