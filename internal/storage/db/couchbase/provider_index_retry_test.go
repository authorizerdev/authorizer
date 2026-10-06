package couchbase

import (
	"errors"
	"fmt"
	"testing"

	"github.com/couchbase/gocb/v2"
	"github.com/stretchr/testify/assert"
)

// A CREATE INDEX that outruns the client is the AMBIGUOUS timeout case — the
// server may have accepted it. If that is not treated as transient, the retry
// that would see "already exists" never runs, and a non-retried index error
// aborts provider construction: the server does not boot.
//
// The original check substring-matched "unambiguous timeout", which does not
// match gocbcore's "ambiguous timeout". This pins both.
func TestIsTransientQueryErr_CoversBothTimeoutKinds(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"ambiguous", gocb.ErrAmbiguousTimeout},
		{"unambiguous", gocb.ErrUnambiguousTimeout},
		{"wrapped ambiguous", fmt.Errorf("create index: %w", gocb.ErrAmbiguousTimeout)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, isTransientQueryErr(tc.err),
				"%v must be retryable, else index creation aborts startup", tc.err)
		})
	}
}

// Guard the premise of the fix: the two sentinels really do share ErrTimeout,
// and their messages really do differ in a way substring matching gets wrong.
func TestGocbTimeoutSentinels_ShareParentButNotMessage(t *testing.T) {
	assert.True(t, errors.Is(gocb.ErrAmbiguousTimeout, gocb.ErrTimeout))
	assert.True(t, errors.Is(gocb.ErrUnambiguousTimeout, gocb.ErrTimeout))
	assert.NotContains(t, gocb.ErrAmbiguousTimeout.Error(), "unambiguous timeout",
		"if this ever contains the longer string, the old substring check was fine")
}

func TestIsTransientQueryErr_NonTransientStaysFatal(t *testing.T) {
	assert.False(t, isTransientQueryErr(nil))
	assert.False(t, isTransientQueryErr(errors.New("syntax error in CREATE INDEX")))
}

// "already exists" short-circuits before the transient check, so a re-run
// against a database that already has the index is a no-op rather than a retry
// loop. This is what makes repeated boots safe.
func TestIsIndexExistsErr_TolerantOnRestart(t *testing.T) {
	assert.True(t, isIndexExistsErr("Index AuditLogResourceIdIndex already exists"))
	assert.True(t, isIndexExistsErr("The index #primary already exists"))
	assert.False(t, isIndexExistsErr("ambiguous timeout"))
}
