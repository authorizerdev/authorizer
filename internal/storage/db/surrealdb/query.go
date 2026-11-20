package surrealdb

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"time"

	surrealdb "github.com/surrealdb/surrealdb.go/v2"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// selectFields is the projection every read uses: the record's own id is
// returned as the plain string `_id`, which is the json tag of the ID field
// on every struct in internal/storage/schemas, so rows decode straight into
// them.
const selectFields = "SELECT *, record::id(id) AS _id OMIT id FROM "

// run executes sql. If the server refuses the session's credentials — the
// sign-in token expired — it signs in again and runs sql once more. Rejected
// requests never execute, so the retry cannot apply a statement twice.
func run[T any](ctx context.Context, p *provider, sql string, vars map[string]any) (*[]surrealdb.QueryResult[T], error) {
	res, err := surrealdb.Query[T](ctx, p.db, sql, vars)
	if !isAuthExpired(err) || p.config.DatabaseUsername == "" {
		return res, err
	}
	if rerr := p.reauthenticate(ctx); rerr != nil {
		return res, fmt.Errorf("%w (re-authentication failed: %v)", err, rerr)
	}
	return surrealdb.Query[T](ctx, p.db, sql, vars)
}

// queryAll runs sql and decodes the result of its last statement into []T.
//
// Rows are fetched as maps and decoded through decodeRow, so fields tagged
// `json:"-"` (password hashes, client secrets, encrypted keys — kept out of
// API JSON) are read back from their stored names too.
func queryAll[T any](ctx context.Context, p *provider, sql string, vars map[string]any) ([]T, error) {
	res, err := run[[]map[string]any](ctx, p, sql, vars)
	if err != nil {
		return nil, err
	}
	if res == nil || len(*res) == 0 {
		return nil, nil
	}
	last := (*res)[len(*res)-1]
	if last.Error != nil {
		return nil, last.Error
	}
	out := make([]T, 0, len(last.Result))
	for _, row := range last.Result {
		var v T
		if err := decodeRow(row, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// queryOne runs sql and returns its first row, or ErrNotFound (wrapped with
// what) when there is none.
func queryOne[T any](ctx context.Context, p *provider, what, sql string, vars map[string]any) (*T, error) {
	rows, err := queryAll[T](ctx, p, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: %w", what, ErrNotFound)
	}
	return &rows[0], nil
}

// exec runs sql for its side effects.
//
// When a transaction fails, SurrealDB reports a placeholder error on every
// statement ("The query was not executed due to a failed transaction") and the
// real cause — a conflict, a duplicate, a bad statement — on just one of them,
// often the COMMIT. exec returns that root cause, so callers can classify it.
func (p *provider) exec(ctx context.Context, sql string, vars map[string]any) error {
	res, err := run[any](ctx, p, sql, vars)
	if res == nil || len(*res) == 0 {
		return err
	}
	var first error
	for _, r := range *res {
		if r.Error == nil {
			continue
		}
		if first == nil {
			first = r.Error
		}
		if !isPlaceholder(r.Error.Message) {
			return r.Error
		}
	}
	if first != nil {
		return first
	}
	return err
}

// isPlaceholder reports whether msg is the stand-in error SurrealDB attaches
// to statements skipped because their transaction already failed.
func isPlaceholder(msg string) bool {
	return strings.Contains(msg, "not executed due to a failed transaction") ||
		strings.Contains(msg, "not executed due to a cancelled transaction") ||
		strings.Contains(msg, "transaction was aborted due to a prior error")
}

// maxTransactionAttempts bounds how often a conflicting transaction is
// replayed.
const maxTransactionAttempts = 8

// transaction runs statements as one BEGIN ... COMMIT and replays the whole
// block when SurrealDB aborts it with a retryable conflict.
//
// Single statements outside a transaction are not retried: SurrealDB
// serialises them itself. 1,000 concurrent single-statement writes to one row
// (full updates, MERGE updates, consume-once deletes, compare-and-set) raised
// no conflict on SurrealDB 4.0.1, while multi-statement transactions over the
// same rows conflict routinely (TestErrorClassifiersMatchServer). Statements must be
// safe to run again: every caller here only deletes or writes absolute values.
func (p *provider) transaction(ctx context.Context, statements string, vars map[string]any) error {
	sql := "BEGIN TRANSACTION;\n" + statements + "\nCOMMIT TRANSACTION;"
	var err error
	for attempt := 0; attempt < maxTransactionAttempts; attempt++ {
		if err = p.exec(ctx, sql, vars); err == nil || !isConflict(err) {
			return err
		}
		base := time.Duration(attempt+1) * 5 * time.Millisecond
		select {
		case <-ctx.Done():
			return err
		case <-time.After(base + rand.N(base)):
		}
	}
	return err
}

// storedName is the field name a struct field is stored under: its json tag,
// or for a `json:"-"` field (kept out of API JSON, e.g. password hashes) its
// bson tag — the same rule the ArangoDB provider uses. "" means not stored.
func storedName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "-" {
		name, _, _ = strings.Cut(f.Tag.Get("bson"), ",")
		if name == "-" {
			return ""
		}
	}
	return name
}

// doc converts a schemas struct into the map stored as the record's content:
// every field under its stored name, keeping Go types (so int64 stays an
// integer), minus `_id` (the record's own id) and `_key` (ArangoDB's key).
func doc(v any) map[string]any {
	rv := reflect.Indirect(reflect.ValueOf(v))
	rt := rv.Type()
	out := make(map[string]any, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		name := storedName(f)
		if name == "" || name == "_id" || name == "_key" {
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Pointer && fv.IsNil() {
			out[name] = nil
			continue
		}
		out[name] = fv.Interface()
	}
	return out
}

// decodeRow decodes one result row into dest: by json tags first, then the
// `json:"-"` fields from their stored names, which JSON decoding skips.
func decodeRow(row map[string]any, dest any) error {
	raw, err := json.Marshal(row)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return err
	}
	rv := reflect.Indirect(reflect.ValueOf(dest))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Tag.Get("json") != "-" {
			continue
		}
		name := storedName(f)
		val, ok := row[name]
		if name == "" || !ok || !rv.Field(i).CanSet() {
			continue
		}
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		nv := reflect.New(f.Type)
		if err := json.Unmarshal(b, nv.Interface()); err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
		rv.Field(i).Set(nv.Elem())
	}
	return nil
}

// create inserts v as a new record of table with the given id. A duplicate
// id or unique key is reported as the error isDuplicate recognises.
func (p *provider) create(ctx context.Context, table, id string, v any) error {
	return p.exec(ctx, "CREATE type::record($tb, $id) CONTENT $doc RETURN NONE;",
		map[string]any{"tb": table, "id": id, "doc": doc(v)})
}

// replace overwrites the content of an existing record. It does not create a
// missing one.
func (p *provider) replace(ctx context.Context, table, id string, v any) error {
	return p.exec(ctx, "UPDATE type::record($tb, $id) CONTENT $doc RETURN NONE;",
		map[string]any{"tb": table, "id": id, "doc": doc(v)})
}

// deleteByID deletes one record; deleting a missing record is not an error.
func (p *provider) deleteByID(ctx context.Context, table, id string) error {
	return p.exec(ctx, "DELETE type::record($tb, $id) RETURN NONE;", map[string]any{"tb": table, "id": id})
}

// getByID fetches one record by id.
func getByID[T any](ctx context.Context, p *provider, table, id, what string) (*T, error) {
	return queryOne[T](ctx, p, what, selectFields+"type::record($tb, $id);", map[string]any{"tb": table, "id": id})
}

// getWhere fetches the first record of table matching where (a SurrealQL
// condition over bound variables only).
func getWhere[T any](ctx context.Context, p *provider, table, where, what string, vars map[string]any) (*T, error) {
	vars["tb"] = table
	return queryOne[T](ctx, p, what, selectFields+"type::table($tb) WHERE "+where+" LIMIT 1;", vars)
}

// listWhere returns every record of table matching where ("" for all).
func listWhere[T any](ctx context.Context, p *provider, table, where string, vars map[string]any) ([]T, error) {
	if vars == nil {
		vars = map[string]any{}
	}
	vars["tb"] = table
	sql := selectFields + "type::table($tb)"
	if where != "" {
		sql += " WHERE " + where
	}
	return queryAll[T](ctx, p, sql+";", vars)
}

// listNewestFirst returns every record of table matching where, newest first.
func listNewestFirst[T any](ctx context.Context, p *provider, table, where string, vars map[string]any) ([]T, error) {
	vars["tb"] = table
	return queryAll[T](ctx, p, selectFields+"type::table($tb) WHERE "+where+" ORDER BY created_at DESC;", vars)
}

// page returns one page of table matching where, newest first, and the
// pagination with Total set to the number of matching records.
func page[T any](ctx context.Context, p *provider, table, where string, vars map[string]any, pagination *model.Pagination) ([]T, *model.Pagination, error) {
	if vars == nil {
		vars = map[string]any{}
	}
	vars["tb"] = table
	vars["limit"] = pagination.Limit
	vars["offset"] = pagination.Offset
	cond := ""
	if where != "" {
		cond = " WHERE " + where
	}
	rows, err := queryAll[T](ctx, p, selectFields+"type::table($tb)"+cond+" ORDER BY created_at DESC LIMIT $limit START $offset;", vars)
	if err != nil {
		return nil, nil, err
	}
	type countRow struct {
		Total int64 `json:"total"`
	}
	counts, err := queryAll[countRow](ctx, p, "SELECT count() AS total FROM type::table($tb)"+cond+" GROUP ALL;", vars)
	if err != nil {
		return nil, nil, err
	}
	out := *pagination
	out.Total = 0
	if len(counts) > 0 {
		out.Total = counts[0].Total
	}
	return rows, &out, nil
}

// ptrs converts a slice of values to a slice of pointers, the shape the
// storage interface returns lists in. An empty result is an empty, non-nil
// slice, as several other providers return.
func ptrs[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}
	return out
}
