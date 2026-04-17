package pipeline

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestCandidateBaseGateHelpersAreSharedAcrossCodePaths(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	filePath := filepath.Join("candidate_materialize.go")
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filePath, err)
	}

	materializeFn := findFuncDecl(file, "MaterializeCandidates")
	if materializeFn == nil {
		t.Fatal("MaterializeCandidates not found")
	}
	inboundStatsFn := findFuncDecl(file, "buildInboundCounterpartyStats")
	if inboundStatsFn == nil {
		t.Fatal("buildInboundCounterpartyStats not found")
	}

	baseGateRules := map[string]struct{}{
		"gateNormalizationResolved": {},
		"gateSupportedAssetType":    {},
		"gateRelationInbound":       {},
	}

	materializeCalls := collectCallNames(materializeFn.Body)
	inboundStatsCalls := collectCallNames(inboundStatsFn.Body)

	if materializeCalls["evaluateBaseEmissionGates"] == 0 {
		t.Fatal("MaterializeCandidates must use evaluateBaseEmissionGates for shared base-gate logic")
	}
	if inboundStatsCalls["evaluateBaseEmissionGates"] == 0 {
		t.Fatal("buildInboundCounterpartyStats must use evaluateBaseEmissionGates for shared base-gate logic")
	}

	for fn := range baseGateRules {
		if materializeCalls[fn] > 0 {
			t.Fatalf("MaterializeCandidates must not duplicate base-gate rule %q directly", fn)
		}
		if inboundStatsCalls[fn] > 0 {
			t.Fatalf("buildInboundCounterpartyStats must not duplicate base-gate rule %q directly", fn)
		}
	}
}

func findFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

func collectCallNames(node ast.Node) map[string]int {
	calls := make(map[string]int)
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			calls[fn.Name]++
		case *ast.SelectorExpr:
			calls[fn.Sel.Name]++
		}
		return true
	})
	return calls
}
