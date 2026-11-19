package sql

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// maxTransactionRetries bounds how often transaction replays a block that
// keeps losing serialization races.
const maxTransactionRetries = 8

// transaction runs fn in a database transaction, the same as
// p.db.WithContext(ctx).Transaction(fn).
//
// With Options.RetryTransactions set, a block that aborts with a
// serialization failure (SQLSTATE 40001) is rolled back and run again from the
// start, with a short jittered backoff, up to maxTransactionRetries times. fn must
// therefore be safe to run more than once: it may only write through tx and
// must have no side effects outside the transaction. Without the option fn
// runs once and its error is returned unchanged.
func (p *provider) transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	for attempt := 0; ; attempt++ {
		err := p.db.WithContext(ctx).Transaction(fn)
		if !p.retryTransactions || attempt == maxTransactionRetries || !isSerializationFailure(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(txBackoff(attempt)):
		}
	}
}

// txBackoff is the wait before replay number attempt+1: linear, plus up to as
// much again in jitter, so transactions that collided once do not collide
// again on every replay.
func txBackoff(attempt int) time.Duration {
	base := time.Duration(attempt+1) * 5 * time.Millisecond
	return base + rand.N(base)
}

// isSerializationFailure reports whether err is SQLSTATE 40001.
func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}
