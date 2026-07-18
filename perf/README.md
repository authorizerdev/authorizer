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

## Tools (all OSS, all optional beyond k6)

| Tool | Layer | Install |
|---|---|---|
| [k6](https://github.com/grafana/k6) | REST/GraphQL (login, token, validate, check_permissions) | `brew install k6` |
| [ghz](https://github.com/bojand/ghz) | gRPC direct (bypasses the REST gateway) | `brew install ghz` |
| `pgbench` | Postgres ceiling, isolated from the app | ships with `brew install postgresql@16` |
| `go test -bench` | Embedded FGA engine, isolated from HTTP/DB | already in the module |

## 0. Build and run the server under test

Use the built binary, not `go run` — and run Postgres/Redis natively
(Homebrew) rather than via Docker Desktop if you're on macOS and the numbers
will be published; Docker Desktop's Linux VM adds a virtualization tax on
disk/network that skews results (see "Publishing numbers" below).

```bash
make build
brew services start postgresql@16
brew services start redis

./build/darwin/arm64/authorizer \
  --admin-secret "$(openssl rand -hex 16)" \
  --database-type postgres \
  --database-url "postgres://localhost:5432/authorizer?sslmode=disable" \
  --fga-store postgres \
  --port 8080
```

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
    -c 50 -z 60s localhost:8081
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
