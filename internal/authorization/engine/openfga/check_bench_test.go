package openfga

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// benchModel is the cheapest possible resolution shape (flat viewer grant,
// no `X from Y` indirection). Deeper production models cost more per Check —
// see perf/README.md before trusting these numbers against a real model.
const benchModel = `model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define can_view: viewer`

const benchWriteBatch = 100 // OpenFGA's own max tuple ops per Write call.

// benchTupleCount is the background tuple volume seeded before each
// benchmark, isolating in-process resolution cost from HTTP/DB round-trip
// cost (that's perf/k6/fga_check.js's job). Override:
//
//	FGA_BENCH_TUPLES=1000000 go test ./internal/authorization/engine/openfga/... -run '^$' -bench BenchmarkCheck -benchtime 5s
func benchTupleCount() int {
	if v := os.Getenv("FGA_BENCH_TUPLES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 10000
}

// setupBenchEngine builds an in-memory engine, writes benchModel, and seeds
// tupleCount background tuples plus one direct hit ("user:bench-hit" on
// "document:bench-hit"). Setup runs before the benchmark's timer starts.
func setupBenchEngine(b *testing.B, tupleCount int) engine.AuthorizationEngine {
	b.Helper()
	ctx := context.Background()
	log := zerolog.New(os.Stderr)

	eng, err := New(&Config{Store: StoreMemory}, &Dependencies{Log: &log})
	if err != nil {
		b.Fatalf("openfga.New: %v", err)
	}
	impl := eng.(*engineImpl)
	b.Cleanup(impl.Close)

	if _, err := eng.WriteModel(ctx, benchModel); err != nil {
		b.Fatalf("WriteModel: %v", err)
	}

	batch := make([]engine.TupleKey, 0, benchWriteBatch)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := eng.WriteTuples(ctx, batch); err != nil {
			b.Fatalf("WriteTuples: %v", err)
		}
		batch = batch[:0]
	}
	for i := 0; i < tupleCount; i++ {
		batch = append(batch, engine.TupleKey{
			User:     fmt.Sprintf("user:bench-user-%d", i),
			Relation: "viewer",
			Object:   fmt.Sprintf("document:bench-doc-%d", i),
		})
		if len(batch) == benchWriteBatch {
			flush()
		}
	}
	flush()

	if err := eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:bench-hit", Relation: "viewer", Object: "document:bench-hit"},
	}); err != nil {
		b.Fatalf("WriteTuples (hit tuple): %v", err)
	}

	return eng
}

// BenchmarkCheck isolates the in-process cost of a single Check against a
// store holding FGA_BENCH_TUPLES background tuples (default 10000).
func BenchmarkCheck(b *testing.B) {
	ctx := context.Background()
	eng := setupBenchEngine(b, benchTupleCount())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.Check(ctx, "user:bench-hit", "can_view", "document:bench-hit"); err != nil {
			b.Fatalf("Check: %v", err)
		}
	}
}

// BenchmarkBatchCheck isolates a 50-item BatchCheck call — OpenFGA's own
// hard cap per BatchCheck RPC (verified empirically; note this is BELOW the
// 100-check max our check_permissions API currently advertises — see
// perf/README.md's "Known issue" callout).
func BenchmarkBatchCheck(b *testing.B) {
	ctx := context.Background()
	eng := setupBenchEngine(b, benchTupleCount())

	requests := make([]engine.CheckRequest, 50)
	for i := range requests {
		requests[i] = engine.CheckRequest{User: "user:bench-hit", Relation: "can_view", Object: "document:bench-hit"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.BatchCheck(ctx, requests); err != nil {
			b.Fatalf("BatchCheck: %v", err)
		}
	}
}

// BenchmarkListObjects isolates ListObjects over the same seeded volume.
func BenchmarkListObjects(b *testing.B) {
	ctx := context.Background()
	eng := setupBenchEngine(b, benchTupleCount())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.ListObjects(ctx, "user:bench-hit", "can_view", "document"); err != nil {
			b.Fatalf("ListObjects: %v", err)
		}
	}
}
