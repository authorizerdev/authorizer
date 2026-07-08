package sql

import (
	libsql "github.com/ekristen/gorm-libsql"
	"github.com/rs/zerolog"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		// Use zerolog so GORM diagnostics are structured JSON on the app logger,
		// never on os.Stdout. The MCP stdio server uses stdout as its JSON-RPC
		// transport; any plain-text GORM line there corrupts the stream.
		Logger: newZerologGORMLogger(deps.Log),
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

	// Older Authorizer releases (< 2.3.0) declared the email and phone_number
	// columns of authorizer_users and authorizer_otps as UNIQUE. v2 keeps plain
	// (non-unique) indexes and enforces uniqueness in the application layer (see
	// AddUser/UpdateUser), so behaviour is identical across all supported
	// databases and these fields can stay optional (email-only / phone-only
	// signups).
	//
	// On an upgraded SQL database those columns are still unique, so GORM 1.25.x
	// AutoMigrate reconciles them by issuing DROP CONSTRAINT "uni_<table>_<col>"
	// — a fixed name (NamingStrategy.UniqueName) that rarely matches what the old
	// DB actually created (authorizer_users_email_key, idx_authorizer_otps_phone_number,
	// or any custom name) — failing with "constraint does not exist" (Postgres
	// SQLSTATE 42704) and aborting startup. Clear the legacy uniqueness up front,
	// name-agnostically, before AutoMigrate runs.
	clearLegacyColumnUniqueness(sqlDB, deps.Log)

	err = sqlDB.AutoMigrate(&schemas.User{}, &schemas.VerificationRequest{}, &schemas.Session{}, &schemas.Env{}, &schemas.Webhook{}, &schemas.WebhookLog{}, &schemas.EmailTemplate{}, &schemas.OTP{}, &schemas.Authenticator{}, &schemas.SessionToken{}, &schemas.MFASession{}, &schemas.OAuthState{}, &schemas.AuditLog{}, &schemas.Client{}, &schemas.TrustedIssuer{}, &schemas.Organization{}, &schemas.OrgMembership{}, &schemas.FederatedIdentity{}, &schemas.ScimEndpoint{})
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

// clearLegacyColumnUniqueness removes any single-column UNIQUE constraint or
// index on the email / phone_number columns of authorizer_users and
// authorizer_otps, regardless of its name. See the call site in NewProvider for
// why this must run before AutoMigrate. Best-effort and non-fatal: fresh
// installs, and catalogs without information_schema (sqlite), simply find
// nothing to drop.
func clearLegacyColumnUniqueness(db *gorm.DB, log *zerolog.Logger) {
	// Run through a discarding GORM logger. The information_schema probe below
	// intentionally errors on databases without that catalog (sqlite); GORM's
	// default logger writes such errors to os.Stdout, which corrupts the MCP
	// server's stdio JSON-RPC stream. We already surface anything actionable via
	// the zerolog `log` (stderr), so GORM's own logging here is redundant.
	db = db.Session(&gorm.Session{Logger: logger.Discard})

	targets := map[string]bool{"email": true, "phone_number": true}
	tables := []struct {
		model any
		name  string
	}{
		{&schemas.User{}, "authorizer_users"},
		{&schemas.OTP{}, "authorizer_otps"},
	}

	for _, t := range tables {
		// 1) Drop single-column UNIQUE *constraints* by their real names, read
		//    from the same catalog GORM uses (information_schema). This matches
		//    ANY name — ..._key, uni_..., idx_..., or custom — which is what
		//    actually prevents the DROP CONSTRAINT "uni_..." 42704 abort. A
		//    constraint's backing index cannot be removed with DROP INDEX, so it
		//    must go via DropConstraint.
		var rows []struct {
			ConstraintName string `gorm:"column:constraint_name"`
			ColumnName     string `gorm:"column:column_name"`
		}
		const q = `SELECT tc.constraint_name AS constraint_name, kcu.column_name AS column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON tc.constraint_name = kcu.constraint_name
			 AND tc.table_schema = kcu.table_schema
			 AND tc.table_name = kcu.table_name
			WHERE tc.constraint_type = 'UNIQUE' AND tc.table_name = ?`
		if err := db.Raw(q, t.name).Scan(&rows).Error; err != nil {
			// sqlite and other catalogs without information_schema land here; they
			// are not affected by the GORM unique-reconciliation behaviour.
			log.Debug().Err(err).Str("table", t.name).Msg("skip legacy unique-constraint scan")
		} else {
			columnsByConstraint := map[string][]string{}
			for _, r := range rows {
				columnsByConstraint[r.ConstraintName] = append(columnsByConstraint[r.ConstraintName], r.ColumnName)
			}
			for name, cols := range columnsByConstraint {
				if len(cols) == 1 && targets[cols[0]] {
					if err := db.Migrator().DropConstraint(t.model, name); err != nil {
						log.Debug().Err(err).Str("constraint", name).Msg("failed to drop legacy unique constraint")
					}
				}
			}
		}

		// 2) Drop any standalone single-column UNIQUE *index* (CREATE UNIQUE
		//    INDEX, not a constraint) on the same columns. These do not cause the
		//    abort — GORM reads column-uniqueness from table_constraints only —
		//    but removing them makes the column genuinely non-unique, matching the
		//    v2 application-layer model. GetIndexes already excludes
		//    constraint-backed indexes, so this only sees the standalone form;
		//    AutoMigrate then recreates the plain non-unique search index.
		indexes, err := db.Migrator().GetIndexes(t.model)
		if err != nil {
			log.Debug().Err(err).Str("table", t.name).Msg("skip legacy unique-index scan")
			continue
		}
		for _, idx := range indexes {
			if unique, ok := idx.Unique(); !ok || !unique {
				continue
			}
			if cols := idx.Columns(); len(cols) == 1 && targets[cols[0]] {
				if err := db.Migrator().DropIndex(t.model, idx.Name()); err != nil {
					log.Debug().Err(err).Str("index", idx.Name()).Msg("failed to drop legacy unique index")
				}
			}
		}
	}
}
