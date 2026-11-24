package surrealdb

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reservedParams are SurrealQL's built-in parameters. Binding a query variable
// with one of these names fails at runtime ("'token' is a protected variable
// and cannot be set"), and only on the statement that uses it — so a typo
// here would surface in production, not at compile time.
var reservedParams = []string{"auth", "access", "token", "session", "before", "after", "value", "input", "this", "parent", "event"}

// TestNoReservedQueryParams scans every bound-variable reference ($name) in
// the provider's SurrealQL for a reserved name.
func TestNoReservedQueryParams(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v (%d files)", err, len(files))
	}
	ref := regexp.MustCompile(`\$([a-z_]+)\b`)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range ref.FindAllStringSubmatch(string(src), -1) {
			for _, r := range reservedParams {
				if m[1] == r {
					t.Errorf("%s binds reserved SurrealQL parameter $%s", path, r)
				}
			}
		}
	}
}
