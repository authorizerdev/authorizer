# Optional fields and NULL semantics across storage providers

This document explains how **nullable** fields (especially `*int64` timestamps such as `email_verified_at`, `phone_number_verified_at`, `revoked_timestamp`) are **stored**, **updated**, and why we **do not** add broad `json`/`bson` **`omitempty`** tags to those struct fields.

## Goal

Application code uses Go **nil pointers** to mean “unset / not verified / not revoked.” Updates must **clear** a previously set value when the pointer is **nil**, matching SQL **NULL** semantics.

## Behaviour by provider

| Provider | Typical update path | Nil pointer on update |
|----------|---------------------|------------------------|
| **SQL (GORM)** | `Save(&user)` | Written as **SQL NULL**. |
| **Cassandra** | JSON → map → `UPDATE` | Nil map values become **`= null`** in CQL. |
| **MongoDB** | `UpdateOne` with `$set` and the `User` struct | Driver marshals nil pointers as **BSON Null** when the field is **not** `omitempty`, so the field is cleared in the document. |
| **Couchbase** | `Upsert` full document | `encoding/json` encodes nil pointers as JSON **`null`** unless the field uses `json:",omitempty"`, in which case the key is **omitted** and old values can persist. |
| **ArangoDB** | `UpdateDocument` with struct | Encoding follows JSON-style rules; nil pointers become **`null`** when not omitted by tags. |
| **DynamoDB** | `UpdateItem` with **SET** from marshalled attributes | Nil pointers are **omitted from SET** (see `internal/storage/db/dynamodb/marshal.go`). Attributes are **not** removed automatically, so **explicit REMOVE** is required to clear a previously stored attribute. Implemented for users in `internal/storage/db/dynamodb/user.go` (`updateByHashKeyWithRemoves`, `userDynamoRemoveAttrsIfNil`). Reads may normalize `0` → unset via `normalizeUserOptionalPtrs`. |

## Why not use `omitempty` on `json` / `bson` for nullable auth fields?

For **document** databases, **`omitempty`** means: *if this pointer is nil, do not include this key in the encoded payload.*

During an **update**, omitting a key usually means **“do not change this field”**, not **“set to null.”** That reproduces the DynamoDB-class bug: the old value remains.

Therefore, for fields where **nil must clear** stored state, keep **`json` / `bson` tags without `omitempty`** (as in `internal/storage/schemas/user.go`) unless every call site is proven to do a **full document replace** and you have verified the driver behaviour end-to-end.

MongoDB’s own guidance aligns with this: `omitempty` skips marshaling empty values, which is wrong when you need to persist **null** to clear a field in `$set`.

## DynamoDB specifics

- **PutItem**: Omitting nil pointers keeps items small; optional attributes may be absent (same idea as “omitempty” on write, but implemented in custom `marshalStruct` by skipping nil pointers).
- **UpdateItem**: Only **SET** attributes present in the marshalled map. Clearing requires **`REMOVE`** for the corresponding attribute names when the Go field is nil.
- Do **not** rely on adding `dynamo:",omitempty"` alone to “fix” updates: the custom marshaller already skips nil pointers; the gap was **REMOVE** on update, not tag-based omission.

## Related code

- Schema: `internal/storage/schemas/user.go`
- DynamoDB user update + REMOVE list: `internal/storage/db/dynamodb/user.go`
- DynamoDB update helper: `internal/storage/db/dynamodb/ops.go` (`updateByHashKeyWithRemoves`)
- DynamoDB marshal/unmarshal: `internal/storage/db/dynamodb/marshal.go`

## TEST_DBS and memory_store tests

- **`internal/memory_store/db`**: Runs one subtest per entry in `TEST_DBS` (URLs aligned with `internal/integration_tests/test_helper.go` — keep `test_config_test.go` in sync when adding backends).
- **`internal/memory_store` (Redis / in-memory)**: Not driven by `TEST_DBS`. In-memory tests always run; **Redis** subtests run only when **`TEST_ENABLE_REDIS=1`** (or `true`) and Redis is reachable (e.g. `localhost:6380` in `provider_test.go`). See `redisMemoryStoreTestsEnabled` in `provider_test.go`.
