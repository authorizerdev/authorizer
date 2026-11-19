package cassandradb

import (
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestIndexProbeQueryOmitsAllowFiltering is the regression guard for the bug where
// the readiness probe carried ALLOW FILTERING. That hint is what lets Cassandra
// answer a query on a non-indexed column, so the probe succeeded on its first
// attempt whether or not the index existed — the wait never waited, and all 17
// call sites kept the startup race it was written to close.
//
// A probe is only a readiness probe if it FAILS until the index is ready.
func TestIndexProbeQueryOmitsAllowFiltering(t *testing.T) {
	q := indexProbeQuery("authorizer", indexProbe{"audit_log", "actor_id"})
	if strings.Contains(strings.ToUpper(q), "ALLOW FILTERING") {
		t.Fatalf("probe must not carry ALLOW FILTERING, it defeats its own purpose: %s", q)
	}
	if want := "SELECT id FROM authorizer.audit_log WHERE actor_id='' LIMIT 1"; q != want {
		t.Fatalf("probe shape changed\n got: %s\nwant: %s", q, want)
	}
}

// TestIndexProbesCoverAuditLogFilters keeps the wait list in step with the columns
// ListAuditLogs filters on. An audit filter whose index is not waited for returns
// an index-unavailable error on the first query against a fresh keyspace.
func TestIndexProbesCoverAuditLogFilters(t *testing.T) {
	probed := map[indexProbe]bool{}
	for _, p := range cassandraIndexProbes {
		if probed[p] {
			t.Errorf("duplicate probe %v — it would consume two shares of the budget", p)
		}
		probed[p] = true
	}
	for _, col := range []string{"actor_id", "action", "resource_type", "resource_id"} {
		if !probed[indexProbe{schemas.Collections.AuditLog, col}] {
			t.Errorf("audit_log.%s is filterable but never probed", col)
		}
	}
}
