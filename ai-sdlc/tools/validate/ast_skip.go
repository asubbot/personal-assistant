package main

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// parseTestFuncsWithTSkip parses Go test source and returns a map from Test* function name to
// whether the function body contains a direct t.Skip(...) call.
func parseTestFuncsWithTSkip(src []byte, filename string) (map[string]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Body == nil {
			continue
		}
		name := fn.Name.Name
		if len(name) < 5 || name[:4] != "Test" {
			continue
		}
		out[name] = funcBodyHasTSkip(fn.Body)
	}
	return out, nil
}

func funcBodyHasTSkip(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "t" || sel.Sel == nil {
			return true
		}
		if sel.Sel.Name == "Skip" {
			found = true
			return false
		}
		return true
	})
	return found
}

// countTestFuncsWithSkip returns how many Test* functions are marked true in the map.
func countTestFuncsWithSkip(m map[string]bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}
