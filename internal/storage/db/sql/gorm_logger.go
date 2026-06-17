package sql

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm/logger"
)

// zerologGORMLogger adapts zerolog to GORM's logger.Interface so that all GORM
// diagnostics (errors, slow queries, statement traces) are emitted as structured
// JSON on the same logger as the rest of the application — never on os.Stdout.
//
// This is critical for the MCP stdio server, which uses stdout as its JSON-RPC
// transport: any stray plain-text line from GORM's default os.Stdout logger
// would corrupt the stream.
type zerologGORMLogger struct {
	log                  zerolog.Logger
	level                logger.LogLevel
	slowThreshold        time.Duration
	ignoreRecordNotFound bool
}

func newZerologGORMLogger(log *zerolog.Logger) logger.Interface {
	return &zerologGORMLogger{
		log:                  log.With().Str("component", "gorm").Logger(),
		level:                logger.Warn,
		slowThreshold:        200 * time.Millisecond,
		ignoreRecordNotFound: true,
	}
}

func (l *zerologGORMLogger) LogMode(level logger.LogLevel) logger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *zerologGORMLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Info {
		l.log.Info().Msgf(msg, args...)
	}
}

func (l *zerologGORMLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Warn {
		l.log.Warn().Msgf(msg, args...)
	}
}

func (l *zerologGORMLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Error {
		l.log.Error().Msgf(msg, args...)
	}
}

// Trace logs SQL statements. Errors surface at error level, slow queries at
// warn level, and everything else at debug level (only when LogLevel >= Info).
func (l *zerologGORMLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.level >= logger.Error &&
		(!errors.Is(err, logger.ErrRecordNotFound) || !l.ignoreRecordNotFound):
		l.log.Error().
			Err(err).
			Str("sql", sql).
			Int64("rows", rows).
			Dur("duration", elapsed).
			Msg("gorm error")

	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.level >= logger.Warn:
		l.log.Warn().
			Str("sql", sql).
			Int64("rows", rows).
			Dur("duration", elapsed).
			Str("slow_threshold", l.slowThreshold.String()).
			Msg("slow sql")

	case l.level >= logger.Info:
		l.log.Debug().
			Str("sql", sql).
			Int64("rows", rows).
			Dur("duration", elapsed).
			Msg("sql")
	}
}

// compile-time assertion that zerologGORMLogger satisfies the interface.
var _ logger.Interface = (*zerologGORMLogger)(nil)
