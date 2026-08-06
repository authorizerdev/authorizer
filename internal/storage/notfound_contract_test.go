package storage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// backends are the storage implementations that must agree on the not-found
// contract documented on Provider.
var backends = []string{"sql", "mongodb", "arangodb", "cassandradb", "dynamodb", "couchbase"}

// listReturnsNilNil are the List* methods that legitimately return (nil, nil)
// for an empty result (rule 2 on Provider), plus the one documented
// single-entity exception.
//
// Everything else must report an absent row as an ERROR (rule 1). A method
// added here without a documented reason is how the DynamoDB TOTP/verification
// panics got in.
var allowedNilNil = map[string]string{
	"GetClientByClientID":      "rule 3: callers distinguish absent from unavailable by err alone",
	"ListClients":              "rule 2: empty list",
	"ListEmailTemplate":        "rule 2: empty list",
	"ListOrgDomainsByOrg":      "rule 2: empty list",
	"ListOrganizations":        "rule 2: empty list",
	"ListSAMLServiceProviders": "rule 2: empty list",
	"ListTrustedIssuers":       "rule 2: empty list",
	"ListUsers":                "rule 2: empty list",
	"ListVerificationRequests": "rule 2: empty list",
	"ListWebhook":              "rule 2: empty list",
	"ListWebhookLogs":          "rule 2: empty list",
	"listOrgMemberships":       "rule 2: empty list (shared helper)",
}

// TestNotFoundContractIsUniform enforces the convention documented on Provider.
//
// It is a static check because the failure it prevents is backend-specific and
// silent: a getter that returns (nil, nil) on ONE backend passes every test run
// against SQLite (which CI uses) and only panics in production on the odd
// backend out. Comparing the backends against each other catches it without
// needing all six databases running.
func TestNotFoundContractIsUniform(t *testing.T) {
	// method -> backend -> returns (nil, nil) somewhere in its body
	seen := map[string]map[string]bool{}

	for _, backend := range backends {
		files, err := filepath.Glob(filepath.Join("db", backend, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", backend, err)
		}
		if len(files) == 0 {
			t.Fatalf("no source files found for backend %q — the check would vacuously pass", backend)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Body == nil {
					continue
				}
				name := fn.Name.Name
				if seen[name] == nil {
					seen[name] = map[string]bool{}
				}
				seen[name][backend] = seen[name][backend] || returnsNilNil(fn.Body)
			}
		}
	}

	shared := 0
	for name, byBackend := range seen {
		if len(byBackend) != len(backends) {
			// Not implemented by every backend (helpers, backend-specific code).
			continue
		}
		shared++

		nilNil, errs := []string{}, []string{}
		for _, b := range backends {
			if byBackend[b] {
				nilNil = append(nilNil, b)
			} else {
				errs = append(errs, b)
			}
		}

		// Divergence: the same method disagrees across backends.
		if len(nilNil) > 0 && len(errs) > 0 {
			sort.Strings(nilNil)
			sort.Strings(errs)
			t.Errorf("%s: not-found contract diverges across backends — "+
				"(nil,nil) in [%s] but an error in [%s]. Callers branch on err alone and "+
				"dereference the row, so this panics on the odd backend only. "+
				"See the not-found convention on storage.Provider.",
				name, strings.Join(nilNil, ","), strings.Join(errs, ","))
			continue
		}

		// Uniform (nil, nil) is only allowed for documented cases.
		if len(nilNil) == len(backends) {
			if _, ok := allowedNilNil[name]; !ok {
				t.Errorf("%s: returns (nil,nil) on every backend but is not in allowedNilNil. "+
					"Single-entity getters must report an absent row as an error (rule 1 on "+
					"storage.Provider). If this is intentional, document why on the method and "+
					"add it to allowedNilNil.", name)
			}
		}
	}

	if shared == 0 {
		t.Fatal("found no methods implemented by all backends — the check would vacuously pass")
	}
	t.Logf("checked %d methods implemented by all %d backends", shared, len(backends))
}

// returnsNilNil reports whether the body contains a literal `return nil, nil`.
func returnsNilNil(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 2 {
			return true
		}
		for _, r := range ret.Results {
			id, ok := r.(*ast.Ident)
			if !ok || id.Name != "nil" {
				return true
			}
		}
		found = true
		return false
	})
	return found
}
