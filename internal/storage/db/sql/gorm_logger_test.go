package sql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

// newTestLogger returns a zerologGORMLogger that writes to buf and the
// underlying zerolog.Logger so callers can inspect emitted JSON.
func newTestLogger(buf *bytes.Buffer) logger.Interface {
	zl := zerolog.New(buf).With().Timestamp().Logger()
	return newZerologGORMLogger(&zl)
}

func loggedFields(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m), "invalid JSON from logger: %s", buf.String())
	return m
}

func TestZerologGORMLogger_Trace(t *testing.T) {
	ctx := context.Background()

	t.Run("normal query at Info level emits debug message", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf).LogMode(logger.Info)

		l.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 1 }, nil)

		fields := loggedFields(t, buf)
		assert.Equal(t, "debug", fields["level"])
		assert.Equal(t, "sql", fields["message"])
		assert.Equal(t, "SELECT 1", fields["sql"])
	})

	t.Run("normal query at Warn level emits nothing", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf) // default level = Warn

		l.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 1 }, nil)

		assert.Empty(t, buf.Bytes(), "no output expected at Warn level for a fast non-error query")
	})

	t.Run("slow query emits warn message", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf) // default slowThreshold = 200ms

		begin := time.Now().Add(-500 * time.Millisecond)
		l.Trace(ctx, begin, func() (string, int64) { return "SELECT slow", 10 }, nil)

		fields := loggedFields(t, buf)
		assert.Equal(t, "warn", fields["level"])
		assert.Equal(t, "slow sql", fields["message"])
		assert.Equal(t, "SELECT slow", fields["sql"])
		assert.NotNil(t, fields["slow_threshold"])
	})

	t.Run("real error emits error message", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf)

		l.Trace(ctx, time.Now(), func() (string, int64) { return "INSERT bad", 0 }, errors.New("constraint violation"))

		fields := loggedFields(t, buf)
		assert.Equal(t, "error", fields["level"])
		assert.Equal(t, "gorm error", fields["message"])
		assert.Equal(t, "INSERT bad", fields["sql"])
		assert.Contains(t, fields["error"], "constraint violation")
	})

	t.Run("ErrRecordNotFound is suppressed by default", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf)

		l.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT missing", 0 }, logger.ErrRecordNotFound)

		assert.Empty(t, buf.Bytes(), "ErrRecordNotFound should be silenced when ignoreRecordNotFound=true")
	})

	t.Run("ErrRecordNotFound surfaces when ignoreRecordNotFound=false", func(t *testing.T) {
		buf := &bytes.Buffer{}
		zl := zerolog.New(buf)
		l := &zerologGORMLogger{
			log:                  zl.With().Str("component", "gorm").Logger(),
			level:                logger.Error,
			slowThreshold:        200 * time.Millisecond,
			ignoreRecordNotFound: false,
		}

		l.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT missing", 0 }, logger.ErrRecordNotFound)

		fields := loggedFields(t, buf)
		assert.Equal(t, "error", fields["level"])
		assert.Equal(t, "gorm error", fields["message"])
	})

	t.Run("Silent level suppresses all output", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf).LogMode(logger.Silent)

		l.Trace(ctx, time.Now().Add(-time.Second), func() (string, int64) { return "SELECT 1", 1 }, errors.New("boom"))

		assert.Empty(t, buf.Bytes())
	})
}

func TestZerologGORMLogger_InfoWarnError(t *testing.T) {
	ctx := context.Background()

	t.Run("Info logs at info level when level>=Info", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf).LogMode(logger.Info)
		l.Info(ctx, "test info %s", "msg")
		fields := loggedFields(t, buf)
		assert.Equal(t, "info", fields["level"])
	})

	t.Run("Info suppressed when level<Info", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf).LogMode(logger.Warn)
		l.Info(ctx, "should not appear")
		assert.Empty(t, buf.Bytes())
	})

	t.Run("Warn logs at warn level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf).LogMode(logger.Warn)
		l.Warn(ctx, "slow thing %d", 42)
		fields := loggedFields(t, buf)
		assert.Equal(t, "warn", fields["level"])
	})

	t.Run("Error logs at error level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := newTestLogger(buf).LogMode(logger.Error)
		l.Error(ctx, "something broke")
		fields := loggedFields(t, buf)
		assert.Equal(t, "error", fields["level"])
	})
}

func TestZerologGORMLogger_LogMode(t *testing.T) {
	buf := &bytes.Buffer{}
	base := newTestLogger(buf)
	clone := base.LogMode(logger.Info)

	// clone must be independent of base
	assert.NotSame(t, base, clone)
}

// TestZerologGORMLogger_StdoutSafety verifies that no bytes land on os.Stdout
// when the logger emits errors or slow-query warnings — the invariant that
// protects the MCP stdio JSON-RPC stream from corruption.
func TestZerologGORMLogger_StdoutSafety(t *testing.T) {
	ctx := context.Background()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	buf := &bytes.Buffer{}
	l := newTestLogger(buf).LogMode(logger.Info)

	// error path
	l.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 0 }, errors.New("boom"))
	// slow-query path
	l.Trace(ctx, time.Now().Add(-time.Second), func() (string, int64) { return "SELECT slow", 1 }, nil)
	// info path
	l.Info(ctx, "hello")

	w.Close()
	os.Stdout = origStdout

	captured := &bytes.Buffer{}
	captured.ReadFrom(r)
	assert.Empty(t, captured.Bytes(), "zerologGORMLogger must never write to os.Stdout")
}
