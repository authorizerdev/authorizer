package cassandradb

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestBuildAuditLogWhereAllowFilteringDecision pins the rule that the scan hint
// is derived from auditLogIndexedColumns, not from the clause count alone.
//
// Getting this wrong is not a performance nit in either direction: a hint-less
// equality against a column with no index is REJECTED by Cassandra, so an index
// dropped without updating the set turns a working filter into an error. That
// coupling is the whole reason the set exists (see #805).
func TestBuildAuditLogWhereAllowFilteringDecision(t *testing.T) {
	tests := []struct {
		name      string
		filter    map[string]interface{}
		wantWhere string
		wantHint  bool
		wantVals  int
	}{
		{
			name:      "no filters needs no clause and no hint",
			filter:    map[string]interface{}{},
			wantWhere: "",
			wantHint:  false,
		},
		{
			name:      "single indexed equality stays index-served",
			filter:    map[string]interface{}{"action": "user.login_success"},
			wantWhere: " WHERE action=?",
			wantHint:  false,
			wantVals:  1,
		},
		{
			name:      "two indexed equalities need the hint: one query, one index",
			filter:    map[string]interface{}{"action": "user.login_success", "actor_id": "u1"},
			wantWhere: " WHERE action=? AND actor_id=?",
			wantHint:  true,
			wantVals:  2,
		},
		{
			name:      "a range on created_at is never index-served",
			filter:    map[string]interface{}{"from_timestamp": int64(1)},
			wantWhere: " WHERE created_at>=?",
			wantHint:  true,
			wantVals:  1,
		},
		{
			name:      "all six filters apply together",
			filter:    map[string]interface{}{"action": "a", "actor_id": "b", "resource_type": "c", "resource_id": "d", "from_timestamp": int64(1), "to_timestamp": int64(2)},
			wantWhere: " WHERE action=? AND actor_id=? AND resource_type=? AND resource_id=? AND created_at>=? AND created_at<=?",
			wantHint:  true,
			wantVals:  6,
		},
		{
			name:      "empty string filters are ignored, not bound as empty",
			filter:    map[string]interface{}{"action": "", "actor_id": "u1"},
			wantWhere: " WHERE actor_id=?",
			wantHint:  false,
			wantVals:  1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			where, vals, hint := buildAuditLogWhere(tc.filter)
			if where != tc.wantWhere {
				t.Errorf("where\n got: %q\nwant: %q", where, tc.wantWhere)
			}
			if hint != tc.wantHint {
				t.Errorf("allowFiltering = %v, want %v (for %q)", hint, tc.wantHint, where)
			}
			if len(vals) != tc.wantVals {
				t.Errorf("bound %d values, want %d", len(vals), tc.wantVals)
			}
			// The hint is returned as a flag, never baked into the clause: CQL
			// grammar puts LIMIT before ALLOW FILTERING.
			if strings.Contains(where, "ALLOW FILTERING") {
				t.Errorf("where clause must not carry the hint inline: %q", where)
			}
		})
	}
}

// TestAuditLogIndexedColumnsMatchCreatedIndexes keeps the set in step with the
// indexes NewProvider actually creates, by reading the provider source.
//
// This is the coupling #805 names: a column listed in the set with no index
// behind it produces a hint-less equality that Cassandra REJECTS, so dropping
// an index without updating the set turns a working filter into an error. A
// static check catches that without needing a live cluster.
func TestAuditLogIndexedColumnsMatchCreatedIndexes(t *testing.T) {
	src, err := os.ReadFile("provider.go")
	if err != nil {
		t.Fatalf("read provider.go: %v", err)
	}
	created := map[string]bool{}
	re := regexp.MustCompile(`CREATE INDEX IF NOT EXISTS authorizer_audit_log_\w+ ON %s\.%s \((\w+)\)`)
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		created[m[1]] = true
	}
	if len(created) == 0 {
		t.Fatal("found no audit_log CREATE INDEX statements — did the statement format change?")
	}
	for col := range auditLogIndexedColumns {
		if !created[col] {
			t.Errorf("auditLogIndexedColumns lists %q but NewProvider creates no index for it: "+
				"a hint-less equality on it would be rejected by Cassandra", col)
		}
	}
	for col := range created {
		if !auditLogIndexedColumns[col] {
			t.Errorf("NewProvider indexes %q but the set omits it: queries on it take a needless scan", col)
		}
	}
	if created["created_at"] {
		t.Error("created_at index is back: a 2i serves equality only, and every query on this column is a range")
	}
}
