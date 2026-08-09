package testutil

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// permissionGuard is one RequirePermission / RequirePermissionAny call site.
//
// keys holds every resource:action the guard accepts. A single-key guard has
// exactly one; an Any-guard passes as soon as the caller holds ANY of its keys,
// so the whole group is satisfied by one usable key — demanding all of them
// would fail on the legacy coarse aliases those guards deliberately keep around
// (see the RequirePermissionAny doc comment in internal/middleware/rbac.go).
type permissionGuard struct {
	keys []string
	pos  string
}

// TestEveryRouteGuardHasAUsablePermission is the standing guard for the lockout
// this repo keeps rediscovering: a route carries RequirePermission("x", "y")
// while no migration ever granted x:y to a role. The permission row may even
// exist — what matters is whether anybody holds it. Nobody holding it means the
// route answers 403 to every user, admin included, and nothing in the build or
// the type system says a word about it.
//
// Found this way on 2026-08-08: mentions:read, seeded by migration 000017 and
// granted by none, guarding GET /api/v1/messages/mentions. Migration 000306
// closed it; this test is here so the next one fails at test time instead of in
// production.
//
// Grants are checked against the SYSTEM presets (tenant_id IS NULL) only. A
// tenant's own custom roles are its business; the presets are what ships.
func TestEveryRouteGuardHasAUsablePermission(t *testing.T) {
	SkipIfNoDB(t)

	guards := scanPermissionGuards(t, "..")
	if len(guards) == 0 {
		t.Fatal("scanned internal/ and found no permission guards at all — the scanner is broken, not the code")
	}
	t.Logf("scanned %d permission guard call sites", len(guards))

	wanted := map[string]bool{}
	for _, g := range guards {
		for _, k := range g.keys {
			wanted[k] = true
		}
	}
	keys := make([]string, 0, len(wanted))
	for k := range wanted {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pool := PoolFromEnv(t)
	defer pool.Close()

	rows, err := pool.Query(context.Background(), `
		SELECT p.name,
		       EXISTS (
		           SELECT 1 FROM role_permissions rp
		           JOIN roles r ON r.id = rp.role_id
		           WHERE rp.permission_id = p.id AND r.tenant_id IS NULL
		       ) AS granted
		FROM permissions p
		WHERE p.name = ANY($1)`, keys)
	if err != nil {
		t.Fatalf("query permissions: %v", err)
	}
	defer rows.Close()

	granted := map[string]bool{}
	seeded := map[string]bool{}
	for rows.Next() {
		var name string
		var isGranted bool
		if err := rows.Scan(&name, &isGranted); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		seeded[name] = true
		granted[name] = isGranted
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate rows: %v", err)
	}

	var failures []string
	for _, g := range guards {
		usable := false
		for _, k := range g.keys {
			if granted[k] {
				usable = true
				break
			}
		}
		if usable {
			continue
		}
		reasons := make([]string, 0, len(g.keys))
		for _, k := range g.keys {
			switch {
			case !seeded[k]:
				reasons = append(reasons, k+" (no permission row)")
			default:
				reasons = append(reasons, k+" (seeded, granted to no preset role)")
			}
		}
		failures = append(failures, fmt.Sprintf("%s: %s", g.pos, strings.Join(reasons, ", ")))
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("%d route guard(s) no preset role can pass — every user including admin gets 403:\n  %s\n\n"+
			"Fix by adding a seed/grant migration for the key, NOT by loosening the guard.",
			len(failures), strings.Join(failures, "\n  "))
	}
}

// scanPermissionGuards walks the Go sources under root and returns every
// RequirePermission / RequirePermissionAny call whose arguments are string
// literals.
//
// lean: guards built from non-literal arguments are skipped and logged rather
// than resolved — there are none today. Reach for full type resolution (or fail
// loudly on a skip) if a route ever computes its permission key at runtime.
func scanPermissionGuards(t *testing.T, root string) []permissionGuard {
	t.Helper()

	var guards []permissionGuard
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == "vendor" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := calleeName(call.Fun)
			if name != "RequirePermission" && name != "RequirePermissionAny" {
				return true
			}
			pos := fset.Position(call.Pos()).String()

			if name == "RequirePermission" {
				if len(call.Args) != 2 {
					return true
				}
				resource, okR := stringLit(call.Args[0])
				action, okA := stringLit(call.Args[1])
				if !okR || !okA {
					t.Logf("skipping non-literal RequirePermission at %s", pos)
					return true
				}
				guards = append(guards, permissionGuard{keys: []string{resource + ":" + action}, pos: pos})
				return true
			}

			// RequirePermissionAny([2]string{resource, action}, ...)
			var keys []string
			for _, arg := range call.Args {
				lit, okC := arg.(*ast.CompositeLit)
				if !okC || len(lit.Elts) != 2 {
					continue
				}
				resource, okR := stringLit(lit.Elts[0])
				action, okA := stringLit(lit.Elts[1])
				if !okR || !okA {
					continue
				}
				keys = append(keys, resource+":"+action)
			}
			if len(keys) != len(call.Args) {
				t.Logf("skipping RequirePermissionAny with non-literal pair(s) at %s", pos)
				return true
			}
			guards = append(guards, permissionGuard{keys: keys, pos: pos})
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return guards
}

// calleeName returns the function name of a call target, for both middleware.X
// and a bare X inside package middleware itself.
func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	}
	return ""
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}
