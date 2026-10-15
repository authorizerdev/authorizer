package cassandradb

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// knownMixedTables are tables that still mix timestamp domains. Empty, and the
// test below fails if a table is listed here without actually mixing — so this
// map can only shrink.
var knownMixedTables = map[string]string{}

// TestEveryTableUsesOneTimestampDomain pins the invariant that all writes to a
// given table take their mutation timestamp from the same clock.
//
// Cassandra and Scylla resolve conflicts by last-write-wins on that timestamp,
// and there are two sources:
//
//   - SERVER. A lightweight transaction (`IF NOT EXISTS`, `IF EXISTS`, `IF
//     <col> = ?`) is stamped from the Paxos ballot. So is any write sent with
//     DefaultTimestamp(false), which tells the driver to send no timestamp and
//     let the coordinator assign one.
//   - CLIENT. Every other write carries gocql's own time.Now() value, because
//     DefaultTimestamp defaults to true and this provider sets no explicit
//     USING TIMESTAMP.
//
// Mix the two on one partition and the later write can carry the earlier
// timestamp, in which case it is discarded with no error at all. Measured on
// scylladb/scylla, 50 pairs per cell:
//
//	LWT insert, then plain row DELETE          46-48/50 deletes dropped
//	LWT insert, then DELETE ... IF EXISTS            0/50
//	LWT insert, then server-stamped DELETE           0/50
//
// Both columns of the fix work; which one a table uses depends on whether its
// LWT is load-bearing. org_domain keeps its LWT and converges onto it (the
// domain is the partition key, first-writer-wins is the atomicity guarantee);
// session_tokens and oauth_states keep a plain insert and server-stamp it,
// because converging the other way would add a Paxos round-trip to every
// login.
//
// A static check, because the live failure depends on sub-millisecond clock
// lead between two processes and cannot be asserted on reliably in CI. #813 is
// the report that led here.
func TestEveryTableUsesOneTimestampDomain(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	// Write statements are built with fmt.Sprintf and name their table as
	// schemas.Collections.X, so both the verb and the target are recoverable
	// from source without a live cluster.
	//
	// Both quote forms, because this provider uses each: the earlier version of
	// this guard matched only "..." and was therefore blind to every raw-string
	// statement — including AddSessionToken's insert, one of the two writes it
	// was written to watch.
	stmt := regexp.MustCompile("[\"`](INSERT INTO|UPDATE|DELETE FROM) %s[^\"`]*[\"`]")
	coll := regexp.MustCompile(`schemas\.Collections\.(\w+)`)
	assigned := regexp.MustCompile(`(\w+)\s*:?=\s*fmt\.Sprintf\(`)

	type seen struct{ server, client []string }
	byTable := map[string]*seen{}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, rErr := os.ReadFile(f)
		if rErr != nil {
			t.Fatalf("read %s: %v", f, rErr)
		}
		lines := strings.Split(string(src), "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			m := stmt.FindString(line)
			if m == "" {
				continue
			}
			tm := coll.FindStringSubmatch(line)
			if tm == nil {
				// Table named on a continuation line; read ahead a little
				// rather than guess.
				for j := i + 1; j < len(lines) && j <= i+3 && tm == nil; j++ {
					tm = coll.FindStringSubmatch(lines[j])
				}
			}
			if tm == nil {
				// The statement names its table through a variable, so which
				// table it hits is only known at run time and this scan cannot
				// classify it. DeleteUser's cascade over
				// schemas.UserOwnedCollections is the one such write today, and
				// it spans both domains — three of those tables are
				// server-domain, four are not.
				//
				// A write like that MUST decide its domain from the table it is
				// about to hit. Requiring it to consult serverDomainTables is
				// what stops the cascade from silently client-stamping a
				// server-domain table, which is invisible everywhere else.
				region := strings.Join(lines[i:min(i+25, len(lines))], "\n")
				if !strings.Contains(region, "serverDomainTables[") {
					t.Errorf("%s:%d writes to a table named by a variable without consulting "+
						"serverDomainTables, so it cannot be stamped for the table it hits:\n  %s",
						f, i+1, strings.TrimSpace(line))
				}
				continue
			}
			table := tm[1]
			if byTable[table] == nil {
				byTable[table] = &seen{}
			}

			// An LWT is stamped from the ballot, so it is already server-domain.
			if strings.Contains(m, "IF NOT EXISTS") || strings.Contains(m, "IF EXISTS") ||
				strings.Contains(m, " IF ") {
				byTable[table].server = append(byTable[table].server, m)
				continue
			}

			// Otherwise the domain is decided at the Query() call that executes
			// this statement: DefaultTimestamp(false) makes it server-domain.
			// Find that call -- either inline on this line, or further down
			// referring to the variable this statement was assigned to.
			region := line
			if am := assigned.FindStringSubmatch(line); am != nil {
				needle := "Query(" + am[1]
				for j := i; j < len(lines) && j <= i+30; j++ {
					if strings.Contains(lines[j], needle) {
						region = strings.Join(lines[j:min(j+3, len(lines))], "\n")
						break
					}
				}
			} else {
				region = strings.Join(lines[i:min(i+3, len(lines))], "\n")
			}
			if strings.Contains(region, "DefaultTimestamp(false)") {
				byTable[table].server = append(byTable[table].server, m)
			} else {
				byTable[table].client = append(byTable[table].client, m)
			}
		}
	}

	if len(byTable) == 0 {
		t.Fatal("found no write statements — did the fmt.Sprintf construction change?")
	}

	for table, s := range byTable {
		mixed := len(s.server) > 0 && len(s.client) > 0
		_, allowed := knownMixedTables[table]

		switch {
		case mixed && !allowed:
			t.Errorf("%s mixes timestamp domains on one partition — the later write "+
				"can carry the earlier timestamp and be dropped silently.\n  server: %v\n  client: %v",
				table, s.server, s.client)
		case !mixed && allowed:
			t.Errorf("%s no longer mixes timestamp domains — remove it from "+
				"knownMixedTables so the guard starts protecting it", table)
		}
	}

	// serverDomainTables must agree with what was just detected. DeleteUser's
	// cascade writes to every table in schemas.UserOwnedCollections through one
	// statement whose target is a loop variable, so the scan above cannot see
	// it; the cascade consults that map instead. If the two disagree, a table
	// is being written in the wrong domain by the cascade and nothing else
	// would say so.
	fieldToTable := map[string]string{}
	cv := reflect.Indirect(reflect.ValueOf(schemas.Collections))
	for i := 0; i < cv.NumField(); i++ {
		fieldToTable[cv.Type().Field(i).Name] = cv.Field(i).String()
	}
	userOwned := map[string]bool{}
	for _, tbl := range schemas.UserOwnedCollections {
		userOwned[tbl] = true
	}
	for field, s := range byTable {
		tbl, ok := fieldToTable[field]
		if !ok {
			t.Errorf("schemas.Collections has no field %s — the reverse lookup drifted", field)
			continue
		}
		isServerDomain := len(s.server) > 0 && len(s.client) == 0
		switch {
		case isServerDomain && userOwned[tbl] && !serverDomainTables[tbl]:
			t.Errorf("%s is server-domain and cascaded by DeleteUser, but missing from "+
				"serverDomainTables — the cascade will client-stamp it and re-mix the table", field)
		case !isServerDomain && serverDomainTables[tbl]:
			t.Errorf("%s is listed in serverDomainTables but its writes are not all "+
				"server-domain\n  server: %v\n  client: %v", field, s.server, s.client)
		}
	}
	for tbl := range serverDomainTables {
		if !userOwned[tbl] {
			continue
		}
		found := false
		for field := range byTable {
			if fieldToTable[field] == tbl {
				found = true
			}
		}
		if !found {
			t.Errorf("serverDomainTables names %q but no write statements were attributed to it", tbl)
		}
	}

	// The tables this guard was written for, named explicitly so a drift in the
	// detection regexes shows up as a failure rather than as silent coverage
	// loss.
	for _, table := range []string{"User", "OrgDomain", "SessionToken", "OAuthState", "Authenticators"} {
		s := byTable[table]
		if s == nil {
			t.Errorf("no write statements attributed to %s — the detection regex likely drifted", table)
			continue
		}
		if len(s.server) > 0 && len(s.client) > 0 {
			t.Errorf("%s regressed to mixed timestamp domains", table)
		}
	}
}
