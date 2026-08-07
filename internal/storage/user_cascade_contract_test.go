package storage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestDeleteUserCascadeIsUniform enforces that every backend's DeleteUser
// cascades over the shared schemas.UserOwnedCollections list rather than a
// hand-rolled subset.
//
// It is a static check for the same reason TestNotFoundContractIsUniform is:
// the failure it prevents is backend-specific and silent. CI runs SQLite only,
// so a cascade that covers six tables on SQL and one on Couchbase passes every
// test run and only strands orphans in production — and an orphaned
// federated-identity row is a permanent SSO lockout, not a tidiness problem.
//
// SQL is the one exception: GORM deletes by model, not by table name, so it
// carries a parallel userOwnedModels list. sql.TestUserOwnedModelsMatchCollections
// asserts the two resolve to the same set of tables.
func TestDeleteUserCascadeIsUniform(t *testing.T) {
	// backend -> identifier its DeleteUser must reference.
	want := map[string]string{
		"sql":         "userOwnedModels",
		"mongodb":     "UserOwnedCollections",
		"arangodb":    "UserOwnedCollections",
		"cassandradb": "UserOwnedCollections",
		"dynamodb":    "UserOwnedCollections",
		"couchbase":   "UserOwnedCollections",
	}

	for _, backend := range backends {
		ident, ok := want[backend]
		if !ok {
			t.Fatalf("backend %q has no expected cascade identifier — add it here when adding a backend", backend)
		}
		path := filepath.Join("db", backend, "user.go")
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		var found, seen bool
		for _, decl := range f.Decls {
			fn, isFn := decl.(*ast.FuncDecl)
			if !isFn || fn.Recv == nil || fn.Body == nil || fn.Name.Name != "DeleteUser" {
				continue
			}
			seen = true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, isIdent := n.(*ast.Ident); isIdent && id.Name == ident {
					found = true
				}
				return !found
			})
		}
		if !seen {
			t.Fatalf("%s: no DeleteUser method found — the check would vacuously pass", path)
		}
		if !found {
			t.Errorf("%s: DeleteUser does not cascade over %s — a user-keyed table added to schemas.UserOwnedCollections would be skipped on this backend", path, ident)
		}
	}
}
