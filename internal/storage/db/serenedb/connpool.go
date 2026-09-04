package serenedb

import (
	"context"
	stdsql "database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// maxWriteConflictRetries bounds the retry loop below. SereneDB resolves a
// conflict by aborting one writer immediately rather than making it wait, so
// the retries are cheap and a contended row converges in a few rounds.
const maxWriteConflictRetries = 8

// sqlState reports whether err carries the given SQLSTATE.
func sqlState(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// uniqueViolation reports whether err is a duplicate-key rejection from a
// unique index. SereneDB returns the standard SQLSTATE 23505 for these.
func uniqueViolation(err error) bool { return sqlState(err, "23505") }

// retryPool retries statements SereneDB aborts with a write conflict.
//
// Its MVCC is optimistic: two transactions touching the same row abort one of
// them (SQLSTATE 40001, "Conflict on tuple deletion") rather than blocking it,
// so retrying — not failing — is the resolution. Without this, concurrent
// writes to one row fail most of the time and leak the driver's message to API
// callers: 20 simultaneous profile updates for one user produced 17 errors.
//
// The retry sits at the pool so every write path gets it, not just the ones
// this package overrides. It applies only to autocommit statements: BeginTx
// hands back the raw *sql.Tx, because replaying one statement inside an already
// aborted transaction is wrong — the whole transaction has to be replayed, and
// GORM's Transaction() offers no hook for that. This is also why the provider
// sets SkipDefaultTransaction: GORM otherwise wraps every single Create/Update
// in a transaction, which would route all writes through BeginTx and past this
// retry. The four explicit Transaction() call sites in the SQL provider
// (user/client/organization/webhook cascade deletes) still surface 40001 under
// contention.
type retryPool struct {
	db *stdsql.DB
}

// newRetryPool adapts a database/sql handle for gorm.
func newRetryPool(db *stdsql.DB) gorm.ConnPool { return &retryPool{db: db} }

func (p *retryPool) retry(fn func() error) error {
	var err error
	for attempt := 0; attempt < maxWriteConflictRetries; attempt++ {
		if err = fn(); err == nil || !sqlState(err, "40001") {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return err
}

func (p *retryPool) PrepareContext(ctx context.Context, query string) (*stdsql.Stmt, error) {
	return p.db.PrepareContext(ctx, query)
}

func (p *retryPool) ExecContext(ctx context.Context, query string, args ...any) (stdsql.Result, error) {
	var res stdsql.Result
	err := p.retry(func() error {
		var err error
		res, err = p.db.ExecContext(ctx, query, args...)
		return err
	})
	return res, err
}

func (p *retryPool) QueryContext(ctx context.Context, query string, args ...any) (*stdsql.Rows, error) {
	var rows *stdsql.Rows
	err := p.retry(func() error {
		var err error
		rows, err = p.db.QueryContext(ctx, query, args...)
		return err
	})
	return rows, err
}

// QueryRowContext cannot retry: *sql.Row defers its error to Scan, so the
// conflict is not visible here. Single-row reads do not raise 40001 — it is a
// write conflict — so this is a read-path no-op rather than a gap.
func (p *retryPool) QueryRowContext(ctx context.Context, query string, args ...any) *stdsql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// BeginTx implements gorm.ConnPoolBeginner. The transaction is deliberately
// unwrapped — see the type comment.
func (p *retryPool) BeginTx(ctx context.Context, opts *stdsql.TxOptions) (gorm.ConnPool, error) {
	return p.db.BeginTx(ctx, opts)
}

// GetDBConn implements gorm.GetDBConnector so gorm.DB.DB() keeps working —
// HealthCheck and Close both go through it.
func (p *retryPool) GetDBConn() (*stdsql.DB, error) { return p.db, nil }
