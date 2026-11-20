package surrealdb

import (
	"context"
	"net"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	surrealdb "github.com/surrealdb/surrealdb.go/v2"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// testURL is the SurrealDB started by `make test-surrealdb`.
const testURL = "http://localhost:8002"

// requireSurrealDB skips unless TEST_DBS includes surrealdb and the server is
// reachable, so `make test` stays Docker-free.
func requireSurrealDB(t *testing.T) {
	t.Helper()
	if !slices.Contains(strings.Split(os.Getenv("TEST_DBS"), ","), constants.DbTypeSurrealDB) {
		t.Skip("set TEST_DBS=surrealdb (make test-surrealdb) to run SurrealDB tests")
	}
	conn, err := net.DialTimeout("tcp", "localhost:8002", 2*time.Second)
	if err != nil {
		t.Skipf("skipping SurrealDB tests: not reachable on localhost:8002: %v", err)
	}
	_ = conn.Close()
}

// freshNamespace returns a namespace no other test uses, so each test starts
// from an empty schema.
func freshNamespace() string { return "t_" + strings.ReplaceAll(uuid.NewString()[:13], "-", "") }

func newTestProvider(t *testing.T, ns string) *provider {
	t.Helper()
	p, err := newProviderAt(ns)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	return p
}

func newProviderAt(ns string) (*provider, error) {
	logger := zerolog.Nop()
	return NewProvider(&config.Config{
		DatabaseType:       constants.DbTypeSurrealDB,
		DatabaseURL:        testURL,
		DatabaseUsername:   "root",
		DatabasePassword:   "root",
		DatabaseName:       "authorizer",
		SurrealDBNamespace: ns,
	}, &Dependencies{Log: &logger})
}

func appliedVersions(t *testing.T, p *provider) []int {
	t.Helper()
	type row struct {
		V int `json:"v"`
	}
	rows, err := queryAll[row](context.Background(), p, "SELECT record::id(id) AS v FROM type::table($mt) ORDER BY v;", map[string]any{"mt": migrationsTable})
	require.NoError(t, err)
	var out []int
	for _, r := range rows {
		out = append(out, r.V)
	}
	return out
}

func allVersions() []int {
	var v []int
	for _, m := range migrations {
		v = append(v, m.version)
	}
	return v
}

// TestMigrateFreshAndRestart: a fresh database gets every table and index the
// schema declares, the version is recorded once, and later boots change nothing.
func TestMigrateFreshAndRestart(t *testing.T) {
	requireSurrealDB(t)
	ns := freshNamespace()
	p := newTestProvider(t, ns)
	assert.Equal(t, allVersions(), appliedVersions(t, p))

	_, indexes := parseSchema(t, migrations)
	for name, idx := range indexes {
		res, err := surrealdb.Query[map[string]any](context.Background(), p.db, "INFO FOR TABLE "+idx.table+";", nil)
		require.NoError(t, err, idx.table)
		assert.Contains(t, (*res)[0].Result["indexes"], name, "index %s on %s", name, idx.table)
	}

	user, err := p.AddUser(context.Background(), &schemas.User{Email: refs.NewStringRef("a@x.io"), SignupMethods: "basic_auth"})
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		again := newTestProvider(t, ns)
		got, err := again.GetUserByID(context.Background(), user.ID)
		require.NoError(t, err, "restart %d lost data", i+1)
		assert.Equal(t, "a@x.io", refs.StringValue(got.Email))
		assert.Equal(t, allVersions(), appliedVersions(t, again))
	}
}

// TestMigrateFailedVersionRollsBack: a version failing part-way leaves neither
// its earlier statements nor its record; the fixed version then applies.
func TestMigrateFailedVersionRollsBack(t *testing.T) {
	requireSurrealDB(t)
	p := newTestProvider(t, freshNamespace())
	ctx := context.Background()
	bad := append(slices.Clone(migrations), migration{
		version: len(migrations) + 1, name: "test: fails half-way",
		statements: []string{`DEFINE TABLE IF NOT EXISTS zz_half SCHEMALESS`, `THROW "deliberate failure"`},
	})
	require.Error(t, p.migrate(ctx, bad))
	assert.Equal(t, allVersions(), appliedVersions(t, p), "the failed version must not be recorded")
	res, err := surrealdb.Query[map[string]any](ctx, p.db, "INFO FOR DB;", nil)
	require.NoError(t, err)
	assert.NotContains(t, (*res)[0].Result["tables"], "zz_half", "the failed version's first statement must roll back")

	fixed := slices.Clone(bad)
	fixed[len(fixed)-1].statements = []string{`DEFINE TABLE IF NOT EXISTS zz_half SCHEMALESS`}
	require.NoError(t, p.migrate(ctx, fixed))
	assert.Equal(t, append(allVersions(), len(fixed)), appliedVersions(t, p))
}

// TestMigrateConcurrentBoot: instances booting at once against an empty
// database all come up, and each version is recorded once.
func TestMigrateConcurrentBoot(t *testing.T) {
	requireSurrealDB(t)
	ns := freshNamespace()
	const instances = 4
	errs := make([]error, instances)
	var wg sync.WaitGroup
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p, err := newProviderAt(ns)
			if err == nil {
				_ = p.Close()
			}
			errs[i] = err
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "instance %d", i)
	}
	assert.Equal(t, allVersions(), appliedVersions(t, newTestProvider(t, ns)))
}

// TestErrorClassifiersMatchServer pins the messages errors.go recognises
// against the real server, so a SurrealDB or SDK upgrade that changes them
// fails here instead of silently turning upserts and retries off.
func TestErrorClassifiersMatchServer(t *testing.T) {
	requireSurrealDB(t)
	p := newTestProvider(t, freshNamespace())
	ctx := context.Background()

	auth := &schemas.Authenticator{UserID: "u", Method: "totp"}
	require.NoError(t, p.create(ctx, schemas.Collections.Authenticators, "a1", auth))
	err := p.create(ctx, schemas.Collections.Authenticators, "a2", auth)
	assert.True(t, isDuplicate(err), "unique-index violation not recognised: %v", err)
	assert.False(t, isRecordExists(err))

	err = p.create(ctx, schemas.Collections.Authenticators, "a1", &schemas.Authenticator{UserID: "u2", Method: "totp"})
	assert.True(t, isRecordExists(err), "duplicate record id not recognised: %v", err)
	assert.False(t, isDuplicate(err))

	// Contended transactions: replayed without the retry, some must conflict.
	require.NoError(t, p.exec(ctx, "CREATE ctr:c SET n = 0;", nil))
	var conflicts, others int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := p.exec(ctx, "BEGIN; UPDATE ctr:c SET n += 1; UPDATE ctr:c SET n += 1; COMMIT;", nil)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				return
			}
			if isConflict(err) {
				conflicts++
			} else {
				others++
				t.Logf("unclassified error: %v", err)
			}
		}()
	}
	wg.Wait()
	assert.Positive(t, conflicts, "20 contended transactions produced no recognised conflict")
	assert.Zero(t, others, "a transaction failed with an error isConflict does not recognise")
}

// TestCascadeDeleteSurvivesContention: concurrent DeleteUser calls on one
// user — each a multi-statement transaction over the same rows — all succeed
// thanks to the whole-transaction replay, and leave nothing behind.
func TestCascadeDeleteSurvivesContention(t *testing.T) {
	requireSurrealDB(t)
	p := newTestProvider(t, freshNamespace())
	ctx := context.Background()
	for round := 0; round < 5; round++ {
		user, err := p.AddUser(ctx, &schemas.User{Email: refs.NewStringRef(uuid.NewString() + "@x.io"), SignupMethods: "basic_auth"})
		require.NoError(t, err)
		for i := 0; i < 5; i++ {
			require.NoError(t, p.AddSession(ctx, &schemas.Session{UserID: user.ID}))
		}
		const workers = 8
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) { defer wg.Done(); errs[i] = p.DeleteUser(ctx, user) }(i)
		}
		wg.Wait()
		for i, err := range errs {
			require.NoError(t, err, "round %d worker %d", round, i)
		}
		_, err = p.GetUserByID(ctx, user.ID)
		assert.True(t, IsNotFound(err), "user must be gone: %v", err)
		left, err := queryAll[map[string]any](ctx, p, "SELECT * FROM type::table($tb) WHERE user_id = $u;", map[string]any{"tb": schemas.Collections.Session, "u": user.ID})
		require.NoError(t, err)
		assert.Empty(t, left)
	}
}

// TestConcurrentEnrollmentLeavesOneAuthenticator: racing AddAuthenticator
// calls for one (user, method) all succeed and leave exactly one record.
func TestConcurrentEnrollmentLeavesOneAuthenticator(t *testing.T) {
	requireSurrealDB(t)
	p := newTestProvider(t, freshNamespace())
	ctx := context.Background()
	userID := uuid.NewString()
	errs := make([]error, 8)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = p.AddAuthenticator(ctx, &schemas.Authenticator{UserID: userID, Method: "totp", Secret: uuid.NewString()})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "worker %d", i)
	}
	rows, err := queryAll[map[string]any](ctx, p, "SELECT * FROM type::table($tb) WHERE user_id = $u;", map[string]any{"tb": schemas.Collections.Authenticators, "u": userID})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

// TestConsumeOnce: concurrent consumers of one OAuth state or one recovery
// code — exactly one wins.
func TestConsumeOnce(t *testing.T) {
	requireSurrealDB(t)
	p := newTestProvider(t, freshNamespace())
	ctx := context.Background()

	require.NoError(t, p.AddOAuthState(ctx, &schemas.OAuthState{StateKey: "st", State: "v"}))
	auth, err := p.AddAuthenticator(ctx, &schemas.Authenticator{UserID: "u", Method: "totp", RecoveryCodes: refs.NewStringRef("old")})
	require.NoError(t, err)

	var states, codes int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			won, err := p.DeleteOAuthStateByKey(ctx, "st")
			assert.NoError(t, err)
			swapped, err := p.ConsumeAuthenticatorRecoveryCode(ctx, auth.ID, "old", "new-"+uuid.NewString())
			assert.NoError(t, err)
			mu.Lock()
			defer mu.Unlock()
			if won {
				states++
			}
			if swapped {
				codes++
			}
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 1, states, "exactly one caller may consume an OAuth state")
	assert.Equal(t, 1, codes, "exactly one caller may consume a recovery code")
}
