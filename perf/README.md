# Performance testing

Load-test harness for Authorizer: token issuance, s2s (`client_credentials`),
JWT/session validation, and the embedded FGA (OpenFGA) layer. Goal is to find
where each layer's ceiling is on your hardware, not to hit a literal number of
concurrent users — see "Minimal → maximum resource tiers" below for why that's
the right framing.

## Known issue found while building this harness

`check_permissions` / `CheckPermissions` accepts up to 100 checks per request
(`maxPermissionChecks` in `internal/service/fga.go`, matches the proto
`max_items: 100`) and passes the whole slice straight to
`AuthzEngine.BatchCheck` in one call
(`internal/service/check_permissions.go:58`). The embedded OpenFGA server
enforces its own hard cap of **50** items per `BatchCheck` RPC — confirmed
empirically while writing `check_bench_test.go`:

```
BatchCheck: openfga.BatchCheck: rpc error: code = Code(2000)
desc = batchCheck received 100 checks, the maximum allowed is 50
```

Any real `check_permissions` call with 51-100 checks fails closed (denies the
whole batch) today. Not fixed here — out of scope for a perf-harness change
and needs its own fix (chunk internally at 50, or lower the public max and
update the proto/docs) plus a regression test. Flagging for a follow-up PR.

## Gotchas found running this end to end (M4 Pro, native Postgres)

Three things will silently wreck your numbers if you don't know about them —
found by actually running these scripts, not just reading the code:

1. **CSRF requires an `Origin` header on every state-changing request.**
   `/v1/signup`, `/v1/login`, `/v1/validate_jwt_token`, `/v1/check_permissions`,
   and all `/v1/admin/*` routes reject POSTs with no `Origin`/`Referer`
   matching `--allowed-origins` (`internal/http_handlers/csrf.go`). Only
   `/oauth/token`, `/oauth/revoke`, `/oauth/introspect`, SCIM, and the SAML ACS
   callback are exempt. The scripts here already send `Origin: <BASE_URL>` —
   if you write your own script or hit these routes with `curl`/`ghz`, do the
   same or every call 403s with `csrf_validation_failed`.

2. **The default per-IP rate limit (30 rps / burst 20) throttles any real load
   test run from one machine.** All of k6's traffic comes from one IP, so
   without raising it you're benchmarking the rate limiter, not the server —
   we saw 99.95% failures (mostly 429s) at 20 VUs until adding
   `--rate-limit-rps=1000000 --rate-limit-burst=1000000` to the server under
   test. This is worth remembering the other direction too: in production,
   traffic concentrated behind one NAT/proxy IP (a shared egress, a small pool
   of s2s callers) hits the same per-IP ceiling unless raised deliberately.

3. **macOS ephemeral port exhaustion on the load-generator side.** A sustained
   ~70k-request k6 run against `localhost` left 20k+ sockets in `TIME_WAIT`,
   which can exhaust the default ephemeral range (`net.inet.ip.portrange.first`
   / `.last`, ~16k ports) and make the *next* k6 run fail with `can't assign
   requested address` — a load-generator artifact, not a server problem. Give
   it 30-60s between successive high-RPS runs on macOS, or widen the ephemeral
   port range, before concluding the server itself has a ceiling.

## Tools (all OSS, all optional beyond k6)

| Tool | Layer | Install |
|---|---|---|
| [k6](https://github.com/grafana/k6) | REST/GraphQL (login, token, validate, check_permissions) | `brew install k6` |
| [ghz](https://github.com/bojand/ghz) | gRPC direct (bypasses the REST gateway) | `brew install ghz` |
| `pgbench` | Postgres ceiling, isolated from the app | ships with `brew install postgresql@16` |
| `go test -bench` | Embedded FGA engine, isolated from HTTP/DB | already in the module |

## 0. Build and run the server under test

Two ways to do this. Use container mode unless you specifically need the raw
hardware ceiling with zero virtualization overhead — most people deploy
Authorizer in a container, so it's the more representative default, and it's
also just less setup.

### Container mode (recommended)

```bash
CPUS=2 MEM=2g ./perf/run_container.sh   # prints ADMIN_SECRET and next steps
# ...run scenarios (section 2)...
./perf/stop_container.sh                # when done
```

Builds `authorizer:perf-local` from the repo's own `Dockerfile` — same image
a real deployment runs — pinned to your host's own CPU architecture (checked
via `uname -m`) so it never silently builds `linux/amd64` and runs under
emulation on Apple Silicon, which would badly skew every number. Starts
Postgres 16 + Redis 7 as containers on a dedicated Docker network, and the
app container with the CPU/memory limits you pass (`CPUS`/`MEM`) — that's
your resource tier. Re-run with a higher `CPUS` to test the next tier; see
"Vertical scaling" below for what that actually buys you.

Rate limiting and CSRF are still live — see "Gotchas" below for the `Origin`
header and `--rate-limit-*` overrides you need for a real load test.

### Native mode (raw hardware ceiling, more setup)

Use the built binary, not `go run` — and run Postgres/Redis natively
(Homebrew) rather than via Docker Desktop if the numbers will be published;
Docker Desktop's Linux VM adds a virtualization tax on disk/network that
skews results (see "Publishing numbers" below).

```bash
make build
brew services start postgresql@16
brew services start redis
createdb authorizer_perf

./build/darwin/arm64/authorizer \
  --admin-secret "$(openssl rand -hex 16)" \
  --database-type postgres \
  --database-url "postgres://localhost:5432/authorizer_perf?sslmode=disable" \
  --jwt-type RS256 \
  --jwt-private-key "$(cat perf/dev-jwt-private.pem)" \
  --jwt-public-key "$(cat perf/dev-jwt-public.pem)" \
  --client-id kbyuFDidLLm280LIwVFiazOqjO3ty8KH \
  --client-secret 60Op4HFM0I8ajz0WdiStAbziZ-VFQttXuxixHHs2R7r7-CW8GR79l-mmLqMhc-Sa \
  --allowed-origins localhost:8080 \
  --http-port 8080
```

`--admin-secret`, `--jwt-type`/`--jwt-private-key`/`--jwt-public-key`, and
`--client-id`/`--client-secret` are all required — the server exits with
`client ID missing in rootArgs` (or similar) without them. `--fga-store` can
be omitted: it auto-reuses `--database-url` when the main DB is SQL-compatible
(postgres/mysql/sqlite) — see `internal/config/fga.go`.

## 1. Seed data

`check_permissions` numbers against an empty FGA store are meaningless — model
depth and tuple count both change resolution cost. Seed first:

```bash
ADMIN_SECRET=... BASE_URL=http://localhost:8080 TUPLES=1000000 \
  k6 run perf/k6/seed_fga.js
```

Writes a small test model (`user -> viewer -> document`) plus N tuples in
batches via `_fga_write_tuples`. Also creates one password user (for
login/validate) and one client_credentials client (for s2s) — printed at the
end, export them for the scripts below.

## 2. Run scenarios

`make perf-seed` / `make perf-k6-login` / `make perf-k6-s2s` /
`make perf-k6-validate` / `make perf-k6-check` / `make perf-fga-bench` wrap
the commands below with defaults — export env vars first to override.

Each script takes `BASE_URL`, `VUS`, `DURATION` (defaults: 10 VUs / 30s —
that's the "minimal" tier; crank these up for higher tiers):

```bash
k6 run -e BASE_URL=http://localhost:8080 -e VUS=50  -e DURATION=1m perf/k6/login.js
k6 run -e BASE_URL=http://localhost:8080 -e VUS=200 -e DURATION=1m -e CLIENT_ID=... -e CLIENT_SECRET=... perf/k6/s2s_client_credentials.js
k6 run -e BASE_URL=http://localhost:8080 -e VUS=500 -e DURATION=1m -e TOKEN=... perf/k6/validate_jwt.js
k6 run -e BASE_URL=http://localhost:8080 -e VUS=200 -e DURATION=1m -e TOKEN=... perf/k6/fga_check.js
```

gRPC direct (bypasses the REST gateway, isolates transport overhead):

```bash
ghz --insecure --proto proto/authorizer/v1/authorizer.proto \
    --call authorizer.v1.AuthorizerService.CheckPermissions \
    -d '{"checks":[{"relation":"can_view","object":"document:1"}]}' \
    -m '{"authorization":"Bearer '"$TOKEN"'"}' \
    -c 50 -z 60s localhost:9091
```

Postgres ceiling, isolated from the app (run once, tells you whether the app
or the DB is the limiter):

```bash
pgbench -i -s 50 authorizer
pgbench -c 50 -j 4 -T 60 authorizer
```

FGA engine in-process ceiling (no HTTP/DB round trip — see
`internal/authorization/engine/openfga/check_bench_test.go`):

```bash
go test ./internal/authorization/engine/openfga/... \
  -run '^$' -bench BenchmarkCheck -benchmem -cpuprofile cpu.out
```

## Minimal → maximum resource tiers

Don't chase "millions of concurrent users" literally — chase the QPS ceiling
millions of users would produce, at increasing resource levels, and confirm
the curve is linear:

| Tier | What changes | What it tells you |
|---|---|---|
| Minimal | 1 replica, default DB pool, `VUS=10` | Correctness under load, baseline latency |
| Medium | 1 replica, tuned DB pool, `VUS=200`+ | Per-replica ceiling, where it breaks first (DB pool / FGA resolution / CPU) |
| Maximum (single box) | N replicas behind a local LB, Redis for rate-limit/session | Whether scaling is linear per replica |
| Maximum (cloud) | Same as above on target deploy hardware (x86 or Graviton) | The number you can actually publish |

### Data points (M4 Pro host, single node)

Not publishable as-is per the checklist below (one laptop, no cloud
corroboration) — but real, reproducible numbers from actually running this
harness, default `--rate-limit-*` raised, `Origin` header set:

| Tier | Scenario | Input | Throughput | p50 | p95 |
|---|---|---|---|---|---|
| A. Native, unconstrained cores | `validate_jwt_token` | 20 VUs | 4,743 req/s | 0.58ms | 14.2ms |
| A. Native, unconstrained cores | `check_permissions` | 20 VUs, 1 tuple | 4,668 req/s | 0.69ms | 13.7ms |
| B. Container, 2 vCPU / 2GB | `login` | 20 VUs | 27.8 req/s | 690ms | 1.00s |
| B. Container, 2 vCPU / 2GB | `client_credentials` s2s | 20 VUs | 7.9 req/s | 2.50s | 2.86s |
| B. Container, 2 vCPU / 2GB | `validate_jwt_token` | 20 VUs | 5,670 req/s | 3.15ms | 6.29ms |
| B. Container, 2 vCPU / 2GB | `check_permissions` | 20 VUs, 200k tuples | 5,001 req/s | 3.52ms | 6.86ms |
| C. Container, 8 vCPU / 4GB | `login` | 80 VUs | 103.1 req/s | 738ms | 1.14s |
| C. Container, 8 vCPU / 4GB | `client_credentials` s2s | 80 VUs | 31.2 req/s | 2.49s | 2.99s |
| C. Container, 8 vCPU / 4GB | `validate_jwt_token` | 100 VUs | 15,721 req/s | 5.71ms | 11.6ms |
| C. Container, 8 vCPU / 4GB | `check_permissions` | 100 VUs, 200k tuples | 12,388 req/s | 7.11ms | 14.3ms |
| D. In-process, memory store (`go test -bench`) | `Check` | 10k bg tuples, no HTTP/DB hop | — | — | 89µs/op |
| D. In-process, memory store (`go test -bench`) | `Check` | 200k bg tuples, no HTTP/DB hop | — | — | 2.25ms/op |

**Correction from an earlier draft of this table:** we originally claimed FGA
resolution cost was flat regardless of store size, based on comparing a
10k-tuple in-process number against a 200k-tuple HTTP number — an
apples-to-oranges comparison on two axes at once (different tuple volume,
*and* the in-process benchmark uses OpenFGA's in-memory datastore while the
HTTP path's FGA store auto-reuses `--database-type=postgres`, a different
backend). Once both were measured at the same 200k volume, the in-memory
datastore's Check cost is **~25x higher at 200k tuples than at 10k**
(89µs → 2.25ms — worth knowing if you rely on OpenFGA's in-memory store
past dev/test scale). At 200k tuples, that 2.25ms in-process cost is roughly
**two-thirds of the entire HTTP check_permissions p50** (3.52ms) — FGA
resolution is a real, dominant cost at this volume, not noise. The
Postgres-backed FGA store (what the HTTP path actually uses) wasn't
benchmarked in-process at 200k tuples here — a real gap, not a claim.
`validate_jwt_token` has no FGA involvement and stays cheap regardless.

`login` and `client_credentials` are slow **by design**: both verify a bcrypt
hash, which is deliberately CPU-expensive to resist offline brute-forcing.
Login uses `bcrypt.DefaultCost` (10, `internal/service/login.go:288`);
service-account client secrets use cost **12** — 4x more expensive, a
decision already on record: *"the schema doc comment on ClientSecret commits
to cost 12 — this MUST stay 12"* (`internal/service/admin_clients.go:24-26`).
Throughput on a CPU-bound op is `cores ÷ time-per-op`: 2 cores ÷ ~70ms (cost
10) ≈ 28 req/s, 2 cores ÷ ~280ms (cost 12) ≈ 7 req/s — both match measured
numbers almost exactly.

**Vertical scaling, proven, not assumed** — B → C is a 4x CPU increase
(2→8 vCPU) at roughly proportional VU increase:

| Scenario | 2 vCPU | 8 vCPU | Scale factor |
|---|---|---|---|
| `login` | 27.8 req/s | 103.1 req/s | 3.71x |
| `client_credentials` s2s | 7.9 req/s | 31.2 req/s | 3.93x |

Near-linear. The fix for slow login/s2s throughput is capacity (more cores),
not weakening bcrypt — that would be a real security regression for a
cosmetic throughput win. `validate_jwt_token`/`check_permissions` scaled less
cleanly (2.5-2.8x) only because they were never core-saturated to begin with
at 20 VUs; the comparison isn't apples-to-apples for those two since VUs rose
5x alongside cores, not 4x.

One anomaly worth flagging: `login` at 8 vCPU/80 VUs had a 2.05% error rate
(0% everywhere else). Likely DB connection pool saturation under 80
concurrent bcrypt-heavy requests — Authorizer has no CLI flag today to tune
`MaxOpenConns`/`MaxIdleConns` (checked `internal/storage/db/sql/client.go`
and `cmd/root.go`; GORM defaults apply). Not chased further here; worth a
look if you hit it at a higher tier.

Isolate the load generator from the server under test at every tier — on a
laptop especially, k6/ghz and the server competing for the same cores measures
contention, not the server's ceiling.

## Publishing numbers worldwide — checklist

Any number that leaves this repo needs its methodology attached, or it's not
defensible against scrutiny (this is a self-hosted auth server competing on
"lighter than Keycloak" — the number needs to survive a skeptical read):

- [ ] Hardware: exact chip, core count, RAM, arch (x86_64 vs arm64)
- [ ] Native vs Docker/VM (Docker Desktop on macOS adds virtualization overhead — disclose it)
- [ ] DB engine + version + pool size, Redis on/off
- [ ] Dataset size seeded (user count, FGA tuple count, model depth)
- [ ] Network topology: loopback vs LAN vs real network (loopback numbers are a floor, not realistic)
- [ ] Warm-up period excluded from reported numbers
- [ ] p50/p95/p99 reported, not just average
- [ ] Single-node vs N-replica, and whether the curve was confirmed linear
- [ ] At least one run corroborated on cloud hardware matching a real deploy target — a laptop-only number is a starting point, not a publishable one
- [ ] Scripts in this directory are what produced the number (reproducibility)
