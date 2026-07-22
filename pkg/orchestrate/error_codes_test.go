// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestEvaluationErrorCodeCatalogComplete(t *testing.T) {
	catalog := AllEvaluationErrorCodes()
	if len(catalog) == 0 {
		t.Fatal("empty catalog")
	}
	seen := make(map[string]struct{}, len(catalog))
	for _, code := range catalog {
		if code == "" {
			t.Fatal("empty catalog entry")
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("duplicate catalog code %q", code)
		}
		seen[code] = struct{}{}
		if exit := ProcessExitFor(code); exit < ExitInvalidRequest || exit > ExitEvidence {
			t.Fatalf("ProcessExitFor(%q)=%d outside taxonomy", code, exit)
		}
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	literals := map[string]struct{}{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident, ok := call.Fun.(*ast.Ident)
				if !ok || ident.Name != "evaluationError" || len(call.Args) == 0 {
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				code := strings.Trim(lit.Value, `"`)
				literals[code] = struct{}{}
				return true
			})
		}
	}
	if len(literals) > 0 {
		keys := make([]string, 0, len(literals))
		for code := range literals {
			keys = append(keys, code)
		}
		sort.Strings(keys)
		t.Fatalf("evaluationError still uses string literals; move to catalog consts: %v", keys)
	}
}
