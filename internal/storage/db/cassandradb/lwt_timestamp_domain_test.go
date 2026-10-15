package cassandradb

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// knownMixedTables are tables that still mix lightweight and plain writes.
//
// Each is a real instance of the defect this test guards against, left for its
// own change because the fix direction is not obvious and both touch paths
// where a wrong call costs more than the bug:
//
//   - authenticators: an LWT insert and an LWT update sit alongside two plain
//     updates and a plain delete. Which way to converge needs deciding, not
//     guessing.
//   - session_tokens / oauth_states: the insert is plain while two deletes are
//     LWT — and those deletes must STAY LWT, because the applied flag is the
//     exactly-once arbiter for token consumption. Converging means lifting the
//     inserts, on the hottest write path in the system.
//
// Shrinking this map is the point. Adding to it is not.
var knownMixedTables = map[string]string{
	"Authenticators": "LWT insert + LWT update alongside plain updates and a plain delete",
	"SessionToken":   "plain insert + LWT deletes (applied flag is the consumption arbiter)",
	"OAuthState":     "plain insert + LWT delete (same arbiter role)",
}

// TestNoTableMixesLWTAndPlainWrites pins the invariant that every write to a
// given table is either a lightweight transaction or none of them are.
//
// Cassandra and Scylla resolve conflicts by last-write-wins on the mutation
// timestamp, and an LWT does not take that timestamp from the same clock a
// plain write does. An LWT insert's row marker is derived from the Paxos ballot
// (server clock); a plain write carries gocql's client-side timestamp
// (DefaultTimestamp defaults to true). A row tombstone only shadows a row whose
// marker it exceeds, so a plain DELETE issued after an LWT insert is discarded
// silently when the server clock runs ahead — no error, the row just survives.
// Mixing the two on one partition is the documented anti-pattern, and this is
// why.
//
// Measured, 50 insert-then-delete pairs per cell:
//
//	scylladb/scylla 3.0.8   plain DELETE 46/50 dropped   IF EXISTS 0/50
//	cassandra 4.1.12        plain DELETE  0/50 dropped   IF EXISTS 0/50
//
// Cassandra escapes it only because its ballot timestamp is truncated to whole
// milliseconds, which floors the marker behind the later client timestamp. That
// is a margin, not a guarantee, and it does not hold on Scylla.
//
// A static check, because the live failure depends on sub-millisecond clock lead
// between two processes and cannot be asserted on reliably in CI. #813 is the
// report that led here; its Org_Domain_Operations failure is this bug.
func TestNoTableMixesLWTAndPlainWrites(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	// Write statements are built with fmt.Sprintf and name their table as
	// schemas.Collections.X, so both the verb and the target are recoverable
	// from source without a live cluster.
	stmt := regexp.MustCompile(`"(INSERT INTO|UPDATE|DELETE FROM) %s[^"]*"`)
	coll := regexp.MustCompile(`schemas\.Collections\.(\w+)`)

	type seen struct{ lwt, plain []string }
	byTable := map[string]*seen{}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, rErr := os.ReadFile(f)
		if rErr != nil {
			t.Fatalf("read %s: %v", f, rErr)
		}
		for _, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			m := stmt.FindString(line)
			if m == "" {
				continue
			}
			tm := coll.FindStringSubmatch(line)
			if tm == nil {
				// Table named on a continuation line; the statement literal
				// alone cannot be attributed, so skip rather than guess.
				continue
			}
			table := tm[1]
			if byTable[table] == nil {
				byTable[table] = &seen{}
			}
			isLWT := strings.Contains(m, "IF NOT EXISTS") || strings.Contains(m, "IF EXISTS")
			if isLWT {
				byTable[table].lwt = append(byTable[table].lwt, m)
			} else {
				byTable[table].plain = append(byTable[table].plain, m)
			}
		}
	}

	if len(byTable) == 0 {
		t.Fatal("found no write statements — did the fmt.Sprintf construction change?")
	}

	for table, s := range byTable {
		mixed := len(s.lwt) > 0 && len(s.plain) > 0
		_, allowed := knownMixedTables[table]

		switch {
		case mixed && !allowed:
			t.Errorf("%s mixes LWT and plain writes on one partition — a plain write "+
				"issued after an LWT can carry an earlier timestamp and be dropped silently.\n  LWT:   %v\n  plain: %v",
				table, s.lwt, s.plain)
		case !mixed && allowed:
			t.Errorf("%s no longer mixes LWT and plain writes — remove it from "+
				"knownMixedTables so the guard starts protecting it", table)
		}
	}

	// The two tables this test was written for.
	for _, table := range []string{"User", "OrgDomain"} {
		s := byTable[table]
		if s == nil {
			t.Errorf("no write statements attributed to %s — the detection regex likely drifted", table)
			continue
		}
		if len(s.lwt) > 0 && len(s.plain) > 0 {
			t.Errorf("%s regressed to mixed writes", table)
		}
	}
}
