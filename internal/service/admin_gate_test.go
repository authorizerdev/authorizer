package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every AdminProvider method must open with its own authorization gate.
//
// This is load-bearing for the gRPC/REST auth model. The interceptor
// (internal/grpcsrv/interceptors/auth.go) does NOT require super-admin for the
// admin service — it cannot, because org-scoped operations legitimately accept
// an org-admin, and requiring super-admin there made those ops reachable over
// GraphQL but not over gRPC/REST. Instead it authenticates the caller, attaches
// the principal, and delegates the decision to the service layer.
//
// That delegation is only safe while every admin method actually decides. A new
// method that forgets requireSuperAdmin/requireOrgAdmin would be reachable by
// ANY authenticated user — no compiler error, no failing handler test, just a
// silent privilege escalation. This test is the thing standing in the way.
//
// The gate must also be at the top level of the function body: one reached only
// inside an `if` is not a gate, it is a suggestion.
func TestAdminMethodsAreGated(t *testing.T) {
	// AdminLogin establishes admin auth and therefore cannot require it. It is
	// the single intentional exception, and the interceptor treats it as public.
	exempt := map[string]bool{"AdminLogin": true}

	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	// Collect the AdminProvider method set from the interface declaration.
	want := map[string]bool{}
	ifaceSrc, err := os.ReadFile("admin_provider.go")
	if err != nil {
		t.Fatalf("read admin_provider.go: %v", err)
	}
	af, err := parser.ParseFile(fset, "admin_provider.go", ifaceSrc, 0)
	if err != nil {
		t.Fatalf("parse admin_provider.go: %v", err)
	}
	ast.Inspect(af, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "AdminProvider" {
			return true
		}
		it, ok := ts.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}
		for _, m := range it.Methods.List {
			for _, name := range m.Names {
				want[name.Name] = true
			}
		}
		return false
	})
	if len(want) == 0 {
		t.Fatal("found no AdminProvider methods — the check would vacuously pass")
	}

	found := map[string]bool{}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil || !want[fn.Name.Name] {
				continue
			}
			found[fn.Name.Name] = true
			if exempt[fn.Name.Name] {
				continue
			}
			if !gatedAtTopLevel(fn.Body) {
				t.Errorf("%s: %s has no top-level requireSuperAdmin/requireOrgAdmin call. "+
					"The gRPC interceptor delegates the admin authorization decision to this "+
					"method, so without a gate it is callable by any authenticated user.",
					path, fn.Name.Name)
			}
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("%s is declared on AdminProvider but no implementation was found to check", name)
		}
	}
	t.Logf("checked %d AdminProvider methods (%d exempt)", len(found), len(exempt))
}

// gatedAtTopLevel reports whether the function body calls requireSuperAdmin or
// requireOrgAdmin as a direct statement — not nested inside a conditional. The
// canonical form is `if err := p.requireX(...); err != nil { return ... }`,
// where the call lives in the if-statement's Init, so that counts; a call
// buried in the if-BODY does not.
func gatedAtTopLevel(body *ast.BlockStmt) bool {
	isGate := func(n ast.Node) bool {
		found := false
		ast.Inspect(n, func(x ast.Node) bool {
			sel, ok := x.(*ast.SelectorExpr)
			if ok && (sel.Sel.Name == "requireSuperAdmin" || sel.Sel.Name == "requireOrgAdmin") {
				found = true
				return false
			}
			return true
		})
		return found
	}
	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			if s.Init != nil && isGate(s.Init) {
				return true
			}
			if isGate(s.Cond) {
				return true
			}
		case *ast.AssignStmt, *ast.ExprStmt:
			if isGate(s) {
				return true
			}
		}
	}
	return false
}
