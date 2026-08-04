package db

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// The DB-backed memory store exists so a multi-instance deployment gets SHARED
// state without requiring Redis. Its cache must therefore live in the database
// too, not in the process.
//
// A process-local cache silently breaks the security controls layered on these
// methods, because each replica keeps its own view:
//   - SAML assertion single-use (http_handlers/saml_sp.go) — an assertion
//     consumed on replica A is unseen by replica B, so a replay against a
//     different replica succeeds for the assertion's whole NotOnOrAfter window.
//   - RFC 7523 client-assertion jti replay (service/clientauth) — same.
//   - TOTP/OTP lockout counters (service/verify_otp.go) — counts never
//     aggregate, so N replicas grant N x the configured attempts.
//   - TOTP pending secrets and org-domain challenges — written on the replica
//     that starts the flow, missing on the one that finishes it.
//
// STORAGE: entries reuse the existing authorizer_session_tokens table, under a
// reserved user_id namespace (cacheNamespace), with the cache key as key_name
// and the value as token. That table is the right home for three concrete
// reasons, none of which the oauth_states table offers:
//
//   - A composite index on (user_id, key_name) — so every get/set/delete is an
//     indexed point lookup, not a scan.
//   - A real expires_at column, indexed. No expiry encoded into the value.
//   - An EXISTING reaper, CleanExpiredSessionTokens, already invoked on the
//     SetUserSession/GetUserSession paths, so expired cache rows are collected
//     without adding a sweeper. There is no equivalent for oauth_states — that
//     table has no reaper at all and grows unboundedly, which would have made
//     any scan over it progressively worse forever.
//
// Namespace safety: cacheNamespace can never collide with a real session's
// user_id, which is either a bare UUID or "<loginMethod>:<uuid>". It contains no
// ":" so DeleteSessionTokensByNamespace (prefix "<ns>:") cannot match it, and
// DeleteAllSessionTokensByUserID's substring match is driven by UUIDs, which are
// never a substring of this literal.
const (
	// cacheNamespace is the reserved user_id under which every cache row lives.
	cacheNamespace = "__authorizer_cache__"
)

// incrementMutex serializes this process's read-modify-write in IncrementCache.
// It does NOT make the increment atomic across instances — see IncrementCache.
var incrementMutex sync.Mutex

// SetCache stores a key-value pair with a TTL in seconds, shared across every
// instance backed by the same database.
//
// FAULT TOLERANCE — this is delete-then-insert, not an atomic upsert, because
// the storage layer exposes no upsert (SetUserSession uses the same two-step).
// So there is a window where the delete succeeds and the insert fails, leaving
// the key ABSENT rather than at either its old or new value. For a lockout
// counter that reads as a reset, which hands back attempts. Redis does not have
// this window (a single SET) and neither did the process-local map.
//
// It is survivable rather than fixed because the error is returned and every
// caller already treats a memory-store fault as fail-open by explicit design
// (verify_otp.go: "a memory-store fault must not be counted as a user failure
// or block a legitimate user"). A flapping database therefore loosens the
// lockout instead of locking out legitimate users — the deliberate direction,
// but worth knowing it is a real degradation and not merely theoretical. An
// atomic upsert in the storage layer closes it.
func (p *provider) SetCache(key string, value string, ttlSeconds int64) error {
	ctx := context.Background()
	now := time.Now().Unix()

	// Overwrite semantics: drop any existing row first. Whether one was actually
	// removed is irrelevant — this is a write, not a single-use claim.
	if err := p.deleteSessionTokenByUserIDAndKey(ctx, cacheNamespace, key); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error clearing existing cache entry")
		// Continue anyway — the add below is what must succeed.
	}

	entry := newCacheRow(key, value, now, ttlSeconds)
	if err := p.addSessionToken(ctx, entry); err != nil {
		return fmt.Errorf("error setting cache: %w", err)
	}
	return nil
}

// cacheRowNamespace seeds the deterministic cache-row UUIDs below. Arbitrary but
// FIXED — changing it re-points every cache row at a new primary key and voids
// the single-use guarantee for entries written by an older build.
var cacheRowNamespace = uuid.MustParse("6f9619ff-8b86-d011-b42d-00c04fc964ff")

// cacheRowID derives the row's primary key from the cache key itself, so the
// SAME cache key always maps to the SAME row ID. Two properties follow:
//
//   - SetCache stays single-row per key (it deletes then inserts the same ID),
//     instead of the random-UUID scheme that could leave duplicate rows for one
//     key after a racing write, with GetCache then picking one arbitrarily.
//   - SetCacheNX gets its atomicity for free from the PRIMARY KEY constraint —
//     the second concurrent insert of the same cache key violates it and fails,
//     with no read-then-write window. See SetCacheNX for the per-backend caveat.
//
// UUIDv5 keeps the value inside the column's char(36).
func cacheRowID(key string) string {
	return uuid.NewSHA1(cacheRowNamespace, []byte(key)).String()
}

func newCacheRow(key, value string, now, ttlSeconds int64) *schemas.SessionToken {
	entry := &schemas.SessionToken{
		ID:        cacheRowID(key),
		UserID:    cacheNamespace,
		KeyName:   key,
		Token:     value,
		ExpiresAt: now + ttlSeconds,
		CreatedAt: now,
		UpdatedAt: now,
	}
	entry.Key = entry.ID
	return entry
}

// SetCacheNX claims a key by CREATING its row, letting the primary-key
// constraint decide the race rather than a read followed by a write. This is
// the replay-defence primitive behind SAML assertion IDs and RFC 7523 jti.
//
// Fails CLOSED: a duplicate key and a genuine storage fault both return
// "not claimed", so a database problem rejects the assertion rather than
// waving through a possible replay.
//
// ponytail: atomic on backends whose insert REJECTS a duplicate primary key
// (SQL, MongoDB, ArangoDB, Couchbase). DynamoDB PutItem and Cassandra/ScyllaDB
// INSERT are upserts by default, so there a concurrent second claim is not
// rejected and both callers can win — unchanged from the previous
// read-then-write behaviour, not a regression.
//
// Closing that gap is cheaper than it looks: both backends support a
// conditional write, and this repo already uses one — see
// storage/db/cassandradb/authenticator.go, which issues
// `INSERT ... IF NOT EXISTS`. The DynamoDB equivalent is PutItem with
// ConditionExpression attribute_not_exists(id). Doing it properly means adding
// a conditional-insert method to storage.Provider and implementing it in all
// six backends, which is why it is tracked rather than done inline here.
//
// Deployments needing an exact guarantee today should configure REDIS_URL,
// whose SET NX is atomic everywhere.
func (p *provider) SetCacheNX(key string, value string, ttlSeconds int64) (bool, error) {
	ctx := context.Background()
	now := time.Now().Unix()

	// An expired row that the reaper has not collected yet must not block a
	// Resolve any pre-existing row for this key BEFORE the insert.
	//
	// This is not merely an optimisation. The insert's atomicity comes from the
	// deterministic primary key (cacheRowID), so it only rejects a duplicate
	// when the existing row carries that same ID. A row written by SetCache on
	// a replica running a build older than this one has a RANDOM uuid instead,
	// so the insert would NOT collide, would succeed, and would report the key
	// as claimed — silently accepting a replay during a rolling upgrade, and
	// leaving two rows for one key.
	//
	// So: a LIVE row of any ID means the key is already held. An EXPIRED row is
	// cleared so it cannot block a legitimate fresh claim. Concurrency among
	// live claimants is still decided by the primary key on insert.
	if row, err := p.getSessionTokenByUserIDAndKey(ctx, cacheNamespace, key); err == nil && row != nil {
		if row.ExpiresAt > now {
			return false, nil
		}
		if err := p.deleteSessionTokenByUserIDAndKey(ctx, cacheNamespace, key); err != nil {
			p.dependencies.Log.Debug().Err(err).Str("cache_key", key).
				Msg("failed to clear expired cache row before claim")
		}
	}

	if err := p.addSessionToken(ctx, newCacheRow(key, value, now, ttlSeconds)); err != nil {
		// Duplicate primary key (already claimed) or a storage fault — both
		// mean "this caller did not take the key". Debug, not Warn: a losing
		// claim is the EXPECTED outcome on every replay attempt.
		p.dependencies.Log.Debug().Err(err).Str("cache_key", key).
			Msg("cache key not claimed (already held or storage fault)")
		return false, nil
	}
	return true, nil
}

// GetCache retrieves a cached value by key. Returns an empty string and a nil
// error when the key is absent or expired (a miss is not an error), matching the
// Redis and in-memory providers.
//
// FAULT TOLERANCE — this returns a MISS, not an error, when the database read
// fails, and that is a fail-OPEN for the replay defences built on it: a store
// outage makes an already-consumed SAML assertion or client-assertion jti look
// unseen, so it is accepted again. Two constraints force it:
//
//   - The storage layer cannot distinguish "no such row" from "read failed" —
//     GetSessionTokenByUserIDAndKey returns a plain error for both, and each of
//     the 7 providers returns a different one (gorm.ErrRecordNotFound,
//     gocql.ErrNotFound, a bare errors.New, a gocb error...). Propagating the
//     error would turn every ordinary cache miss into a failure.
//   - Both replay callers already discard it: saml_sp.go only reports a replay
//     when err == nil, and clientauth drops the error entirely. So returning it
//     would not currently change their behaviour.
//
// The fault is logged at DEBUG, deliberately not Warn: because a miss and a
// failure are the same error here, and a miss is the COMMON case for replay
// defence (most assertions are new), Warn would emit a line on every normal
// lookup and flood the log. Debug keeps it retrievable when investigating
// without drowning the signal.
//
// Closing the fail-open properly means giving the storage layer a miss/error
// distinction (a (*SessionToken, bool, error) signature or a sentinel) and then
// making those callers fail closed — a cross-provider contract change, tracked
// rather than bodged here.
func (p *provider) GetCache(key string) (string, error) {
	ctx := context.Background()
	row, err := p.getSessionTokenByUserIDAndKey(ctx, cacheNamespace, key)
	if err != nil {
		// Indistinguishable from a miss (see above). Debug, not Warn — see the
		// log-level note in the doc comment.
		p.dependencies.Log.Debug().Err(err).Str("cache_key", key).
			Msg("cache read miss or failure (indistinguishable at this layer); treated as a miss")
		return "", nil
	}
	if row == nil {
		return "", nil
	}
	// Inclusive expiry, matching GetUserSession and the in-memory store: an entry
	// is invalid AT its expiry second, not one second later.
	//
	// No eviction here on purpose: this is a read path (every replay check hits
	// it), and deleting from it would add a write per expired lookup. The row is
	// already treated as absent, SetCache deletes before it rewrites, and
	// CleanExpiredSessionTokens — invoked on the SetUserSession/GetUserSession
	// paths — collects it.
	if row.ExpiresAt <= time.Now().Unix() {
		return "", nil
	}
	return row.Token, nil
}

// IncrementCache increments the counter at key, creating it at 1 when absent or
// expired, and refreshes its TTL.
//
// ponytail: read-modify-write against the shared table, serialized only within
// this process. Two replicas incrementing in the same instant can read the same
// prior value and one increment is lost, so a counter can UNDERCOUNT under exact
// concurrency. That is deliberate and strictly bounded: the previous
// process-local map made counts not aggregate at all, so N replicas granted N x
// the attempts before an OTP lockout — this makes the count shared and
// approximately right instead of reliably wrong. For an exact distributed
// counter configure REDIS_URL (the Redis provider uses a native atomic INCR);
// the upgrade path here is a compare-and-set or native increment in the storage
// layer, which is a per-provider change across all 7 backends.
func (p *provider) IncrementCache(key string, ttlSeconds int64) (int64, error) {
	incrementMutex.Lock()
	defer incrementMutex.Unlock()

	current, err := p.GetCache(key)
	if err != nil {
		return 0, err
	}
	var next int64 = 1
	if current != "" {
		if parsed, pErr := strconv.ParseInt(current, 10, 64); pErr == nil {
			next = parsed + 1
		}
	}
	if err := p.SetCache(key, strconv.FormatInt(next, 10), ttlSeconds); err != nil {
		return 0, err
	}
	return next, nil
}

// DeleteCacheByPrefix removes the cache entry for the given key.
//
// IMPORTANT — this performs an EXACT-KEY delete, not prefix expansion, and so
// diverges from the Redis and in-memory providers, which do expand a prefix.
// That divergence is deliberate and is safe only because every caller today
// passes a COMPLETE key, never a partial one:
//
//	authenticators/totp/totp.go   DeleteCacheByPrefix(totpPendingSecretKey(userID))
//	service/verify_otp.go         DeleteCacheByPrefix(lockKey)
//	service/verify_otp.go         DeleteCacheByPrefix(otpLockKey)
//
// The alternative — scanning every row and filtering in Go — would put an
// O(table) query on a user-facing auth path (it runs on every successful OTP or
// TOTP verification), and the storage interface exposes no prefix query to do it
// indexed. An exact-key delete rides the (user_id, key_name) index instead.
//
// If a caller ever genuinely needs prefix expansion here it will silently no-op,
// so: add a prefix-delete to storage.Provider (all 7 backends) and implement it
// properly rather than reaching for GetAllSessionTokens.
func (p *provider) DeleteCacheByPrefix(prefix string) error {
	ctx := context.Background()
	return p.deleteSessionTokenByUserIDAndKey(ctx, cacheNamespace, prefix)
}
