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

### Minimal-tier data point (M4 Pro, native Postgres 16, single node, loopback, 20 VUs)

Not publishable as-is per the checklist below (single laptop, loopback, no
cloud corroboration) — but a real, reproducible floor from actually running
this harness, default `--rate-limit-*` raised, `Origin` header set:

| Scenario | Throughput | p90 | p95 |
|---|---|---|---|
| `validate_jwt_token` | ~4,740 req/s | 10.8ms | 14.2ms |
| `check_permissions` (1 tuple, cold store) | ~4,670 req/s | 11.1ms | 13.7ms |
| FGA engine in-process `Check` (10k bg tuples, no HTTP/DB hop) | — | — | ~89µs/op |
| FGA engine in-process `BatchCheck` (50 items) | — | — | ~246µs/op |

The ~50x gap between the in-process Check (89µs) and the HTTP-path
check_permissions (~13.7ms p95) is the HTTP+CSRF+auth+DB round trip, not FGA
resolution — that's the layer to optimize first if the HTTP-path number needs
to move. Re-run at higher `VUS` and with a tuned DB pool to find where it
actually breaks; 20 VUs on a laptop is nowhere near this server's ceiling.

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
