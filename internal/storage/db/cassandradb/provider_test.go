package cassandradb

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestBuildCQLColumnMapIncludesJSONHyphenFields proves buildCQLColumnMap sources
// column names from the `cql` tag, not encoding/json — so a field tagged json:"-"
// (kept out of API/log JSON) is still written to the database. This is the exact
// bug class that silently dropped User.Password: the guard here protects ANY future
// entity whose sensitive field a maintainer tags json:"-", not just the handful of
// write paths currently wired to the helper.
func TestBuildCQLColumnMapIncludesJSONHyphenFields(t *testing.T) {
	type sample struct {
		Secret    string  `json:"-" cql:"secret"`
		Name      string  `json:"name" cql:"name"`
		Ignored   string  `json:"ignored" cql:"-"`
		Untagged  string  // no cql tag
		OmitEmpty *string `json:"omit_empty" cql:"omit_empty,omitempty"`
	}
	s := sample{Secret: "totp-seed", Name: "n", Ignored: "x", Untagged: "y"}
	m := buildCQLColumnMap(s)

	got, ok := m["secret"]
	if !ok {
		t.Fatal(`buildCQLColumnMap dropped the json:"-" secret field; it must persist via its cql tag`)
	}
	if got != "totp-seed" {
		t.Fatalf("secret column = %v, want totp-seed", got)
	}
	if _, ok := m["name"]; !ok {
		t.Fatal("name column missing")
	}
	if _, ok := m["-"]; ok {
		t.Fatal(`cql:"-" field must be skipped`)
	}
	if _, ok := m["Untagged"]; ok {
		t.Fatal("untagged field must be skipped")
	}
	if _, ok := m["omit_empty"]; ok {
		t.Fatal("omitempty field with a nil pointer must be omitted")
	}
	if len(m) != 2 {
		t.Fatalf("expected only secret and name columns, got %d: %v", len(m), m)
	}
}

// TestUpdateUserRejectsPartialStruct proves UpdateUser refuses a struct with a zero
// CreatedAt — a partial struct that would otherwise blank every column it does not
// carry. The guard returns before any DB access, so no live connection is required.
func TestUpdateUserRejectsPartialStruct(t *testing.T) {
	p := &provider{}
	if _, err := p.UpdateUser(context.Background(), &schemas.User{ID: "some-id"}); err == nil {
		t.Fatal("UpdateUser must reject a struct with CreatedAt == 0")
	}
}
