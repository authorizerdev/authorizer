# Release runbook: keeping the server and SDKs in sync

Authorizer ships as one server repo plus five satellite repos that must stay
in lockstep with it: three hand-maintained client SDKs (`authorizer-go`,
`authorizer-py`, `authorizer-js`) and two generated-only proto packages
(`authorizer-proto-go`, `authorizer-proto-python`) that the Go/Python SDKs
depend on. Nearly every cross-repo bug found in this project's history has
been the same shape: **the server's actual wire contract changed, and some
downstream artifact — a vendored stub, a hand-written type, a doc page —
didn't**. This runbook exists to make that class of bug structurally harder,
not just something a reviewer has to remember to check for.

## The dependency graph

```
proto/*.proto (source of truth)
  │
  ├─▶ gen/go, gen/openapi           (server-side; make proto-gen / proto-check)
  │
  ├─▶ buf.build/authorizerdev/authorizer (BSR module, pushed on every merge to main)
  │     │
  │     ├─▶ authorizer-proto-go     (own repo, regenerates from BSR on its own schedule)
  │     │     └─▶ authorizer-go depends on it as a real go.mod dependency
  │     │
  │     └─▶ authorizer-proto-python (own repo, regenerates from BSR on its own schedule)
  │           └─▶ authorizer-py depends on it as a real PyPI dependency
  │
  └─▶ internal/graph/schema.graphqls (GraphQL — independently maintained, NOT
        generated from proto; can drift from it silently, see "Known drift
        sources" below)
        └─▶ authorizer-js hand-types its GraphQL/REST surface directly
            (cannot use gen/ts's message types — see "Why authorizer-js
            stays hand-typed" below)
```

Two facts fall out of this that drive the whole runbook:

1. **`authorizer-go`/`authorizer-py` are no longer built by copying generated
   code.** They depend on `authorizer-proto-go`/`authorizer-proto` as real,
   versioned dependencies — the same shape as `prometheus/client_golang`
   depending on `prometheus/client_model`. There is no manual "regenerate,
   then copy into 3 repos" step left for those two.
2. **`authorizer-js`'s GraphQL/REST surface and the GraphQL schema itself are
   both hand-maintained and can drift independently of proto.** This is the
   one place in the whole graph without a generated source of truth, so it
   needs the most manual vigilance.

## Release order

Always release in this order — each step's version needs to exist before the
next step can depend on it correctly:

1. **`authorizer` (server)** — merge to `main`, cut a release (`gh release
   create <tag> --prerelease` for an RC). This pushes the updated schema to
   BSR automatically (`.github/workflows/buf.yml`'s `push to main -> publish
   (buf push)` job — verify it actually ran and succeeded, don't assume).
2. **Wait for the published Docker image to actually exist** before running
   any SDK CI that pulls it. `release.yaml` triggers on `release: created`,
   not on merge — the image build+push takes real time (observed: ~10-12
   minutes end to end). Check `docker manifest inspect
   quay.io/authorizer/authorizer:<tag>` succeeds before re-running SDK CI
   against it — CI that runs too early gets a 404/timeout, not a useful
   signal that something's actually broken.
3. **`authorizer-proto-go` / `authorizer-proto-python`** — if the proto
   change affects generated messages, trigger `regenerate.yml`
   (`workflow_dispatch`, don't wait for the weekly cron if you need it now),
   review the auto-opened PR, merge, tag a new version, and for
   `authorizer-proto-python` confirm the PyPI publish actually succeeded
   (`release.yml` is tag-triggered — the workflow that runs is the one
   **at that tag's commit**, not whatever's on `main` right now; if you
   retag after fixing the release workflow itself, you must move the tag to
   the new commit or the old workflow content still runs).
4. **`authorizer-go` / `authorizer-py`** — bump their dependency on the
   proto package(s), update any code touching removed/changed fields, run
   the FULL live-integration suite against a locally built server image
   (not just unit tests — see "Verification bar" below), then release.
5. **`authorizer-js`** — only if the GraphQL/REST surface actually changed
   (see "Known drift sources"); it never touches the proto packages.
6. **Examples repo, docs repo** — update pinned SDK versions and any
   documented method signatures last, once the SDKs they reference are
   actually published and installable.

Skipping the order (e.g. releasing an SDK before its proto dependency is
published) produces exactly the failure this session hit: CI that resolves
a version that doesn't exist yet, or worse, silently resolves an *older*
cached version and passes for the wrong reason.

## Pre-release checklist

Run all of these before cutting **any** release, server or SDK:

- [ ] **`make proto-check`** (server-side gen/go, gen/openapi) is clean.
- [ ] **`make proto-check-clients`** (gen/ts) is clean. Note this does
      **not** cover `authorizer-proto-go`/`authorizer-proto-python` — those
      regenerate from BSR independently on their own repos, not from this
      repo's `gen/` at all anymore.
- [ ] **GraphQL schema vs. proto schema**: if you touched a field/message
      that exists conceptually in both (pagination, request envelopes,
      anything with a GraphQL input type mirroring a proto message), grep
      the other one and confirm they still agree. There is **no automated
      check for this** — it's the one drift class this whole graph doesn't
      structurally prevent. When in doubt, diff the GraphQL input type
      against the proto message field-by-field by hand.
- [ ] **Full test suite green** — `go build ./...`, `go vet ./...`, `make
      test` (SQLite), `make lint`. For storage-layer changes, at least one
      non-SQL backend too.
- [ ] **`make smoke`** before a server release specifically (build tag
      `smoke`, boots the real binary, exercises GraphQL/REST/gRPC/MCP).
- [ ] **Docs**: if a public method's signature, a request/response shape, or
      a config flag changed, grep `docs/sdks/*/functions.md` and
      `docs/sdks/*/admin.md` in the `authorizerdev/docs` repo for the old
      shape. These are hand-written and have gone stale before (46 of 81 Go
      admin methods were once undocumented — caught by an explicit audit,
      not any automated process).

## Verification bar for SDK releases

"The unit tests pass" is not sufficient — the bugs this project has actually
shipped were wire-shape mismatches that only a real round trip catches.
Before tagging an SDK release:

1. Build a local server image from the exact commit you're releasing
   against: `docker build -t authorizer:localmain .` from the server repo.
2. Run it with the flags the SDK's own test suite expects (check
   `test/authorizer_test.go` / `tests/integration/test_live.py` for the
   exact `client-id`/`admin-secret`/ports/`--disable-mfa` defaults — they're
   documented at the top of those files specifically so this doesn't need
   re-discovering every time).
3. Run the **full** live-integration suite (not `-m "not live"`, not just
   unit tests) across every transport the SDK supports (graphql, rest,
   grpc) — a bug can be transport-specific (the `_users` admin query's stale
   `PaginatedRequest` type broke GraphQL only; REST/gRPC use proto messages
   directly and were unaffected, which is exactly the kind of thing a
   single-transport check would miss).
4. For a **published-package** change (not just a git branch), do an actual
   clean install from the registry (`go get module@version` in a throwaway
   module, `pip install package==version` in a fresh venv) and confirm it
   resolves and imports — `go.sum`/PyPI propagation lag and packaging
   mistakes (a missing `/v2` module suffix, a broken sdist) don't show up
   from testing a local checkout.

## Known drift sources (from this project's actual incident history)

Keep this section updated — every entry here is a real bug that shipped,
not a hypothetical:

- **GraphQL `PaginatedRequest` double-wrapper vs. proto's flat
  `PaginationRequest`.** The two schemas were never generated from the same
  source for this type; they silently diverged. Fixed by standardizing on
  the proto shape everywhere and removing the wrapper — but nothing
  prevents a *new* divergence like this from being introduced again. Grep
  both schemas by hand for anything that looks like a shared concept.
- **A field removed from the server (`is_multi_factor_auth_enabled` on
  `SignupRequest`, a security fix) kept getting sent by SDKs that hadn't
  been updated**, in some cases for weeks. The removal was correct and
  reviewed; the *propagation* to dependents wasn't tracked anywhere.
- **`gen/go-client`/`gen/python` (before they were replaced by dedicated
  proto packages) went stale for the same reason** — the regen step was a
  manual "clone the server repo, run buf generate, copy the output in"
  dance that nobody was reminded to do. This is why those directories don't
  exist anymore; the dedicated proto-package repos regenerate themselves
  from BSR instead of depending on a human remembering.
- **`authorizer-go`'s `go.mod` never got the `/v2` module-path suffix** Go
  requires once a module is tagged v2+. Every v2.x release before this was
  found was **completely unresolvable** via `go get` — not a subtle bug, a
  release that silently could never be consumed, for an unknown amount of
  time before anyone tried a clean install and noticed. This is exactly why
  step 4 of the verification bar above (a real clean install, not a local
  checkout test) exists.
- **A CI-config secret dependency wasn't set until the first real release
  attempt.** A repo's `release.yml` referencing `secrets.PYPI_API_TOKEN`
  looks complete in code review; it doesn't fail until someone actually
  tries to cut a release and the token doesn't exist. Prefer PyPI's Trusted
  Publishing (OIDC via `pypa/gh-action-pypi-publish`) for new packages —
  there's no secret to forget to set in the first place.

## Why `authorizer-js` stays hand-typed

Documented here because the temptation to "just use the generated `gen/ts`
types" comes up naturally once Go/Python are on generated packages, and the
answer is a real architectural mismatch, not an oversight:

- `gen/ts` (protobuf-es) generates **camelCase** message shapes
  (`phoneNumber`, `confirmPassword`) — the standard JS/TS convention for
  generated protobuf code.
- This server's REST transport is **deliberately configured to snake_case**
  (`internal/gateway/mount.go`, `UseProtoNames: true`) specifically to keep
  REST payloads aligned with the GraphQL surface, which is also snake_case.
- `authorizer-js` only speaks GraphQL and REST (browsers can't speak raw
  gRPC, so it never needed the generated message types for that reason
  either) — both of its real transports are snake_case, so generated
  camelCase types would require a rename/mapping layer to actually match
  the wire format, at which point the "just use the generated types"
  shortcut isn't actually a shortcut.

If this server's REST config ever changes to emit camelCase by default,
this tradeoff should be re-evaluated — but as long as REST intentionally
mirrors GraphQL's snake_case, generated TS types would be actively wrong
for both of `authorizer-js`'s transports, not just unnecessary.

## SDK-level e2e coverage

`e2e-playground/` (this repo's live-playground Playwright suite) exercises
the full feature surface — OIDC, SAML, SCIM, WebAuthn, TOTP/SMS-OTP/WebOTP,
10 social OAuth providers, MFA enforcement, OTP lockout — but does so via
raw GraphQL/REST calls and browser automation, never through the actual
published SDKs. `e2e-playground/sdk-tests/{go,python}/` exercises the same
live stack through the real `authorizer-go`/`authorizer-py` packages
specifically to catch the wire-shape-mismatch class of bug described above
before it reaches a real consumer. When adding a new feature to the server,
consider whether it needs coverage in both places — the Playwright suite
proves the feature works; the SDK-driven suite proves the SDKs correctly
expose it.
