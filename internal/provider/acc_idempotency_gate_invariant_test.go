package provider

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"testing"
)

// ungatedStepCeiling is a ratchet, not a target. It is the number of
// acceptance TestSteps that apply a config without asserting the plan settles
// afterwards. It may only go down, and it is now at zero.
//
// If this fails because the count went UP, the new step needs a gate, not a
// higher number here.
const ungatedStepCeiling = 0

// TestAcceptanceStepsHaveIdempotencyGate reports acceptance steps that apply a
// config and then never check that a follow-up plan is empty.
//
// This is the check that would have caught the SMB options block shipping
// broken. A resource whose attributes are Optional+Computed reads server
// defaults back into state, and if those come back as diffable changes, every
// later plan shows a phantom diff on something nobody touched. The apply
// succeeds, the Check functions pass, and nothing notices.
//
// Steps that cannot apply anything are not counted: an import round-trip, a
// plan-only step, one that expects an error, one that deliberately expects a
// non-empty plan, and a refresh or destroy step.
func TestAcceptanceStepsHaveIdempotencyGate(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}

	var ungated []string
	var applied, gated int
	for _, dir := range []string{"internal/provider", "internal/resources", "internal/datasources"} {
		files, globErr := filepath.Glob(filepath.Join(root, dir, "*_test.go"))
		if globErr != nil {
			t.Fatalf("globbing %s: %v", dir, globErr)
		}
		for _, file := range files {
			fset := token.NewFileSet()
			node, parseErr := parser.ParseFile(fset, file, nil, 0)
			if parseErr != nil {
				t.Errorf("parsing %s: %v", file, parseErr)
				continue
			}
			rel, _ := filepath.Rel(root, file)
			ast.Inspect(node, func(n ast.Node) bool {
				// Steps: []resource.TestStep{ {...}, {...} }. The elements have
				// elided types, so match the slice and walk it.
				slice, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				arr, ok := slice.Type.(*ast.ArrayType)
				if !ok {
					return true
				}
				sel, ok := arr.Elt.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "TestStep" {
					return true
				}
				for _, elt := range slice.Elts {
					step, ok := elt.(*ast.CompositeLit)
					if !ok {
						continue
					}
					keys := stepKeys(step)
					if !keys["Config"] || stepCannotApply(keys) {
						continue
					}
					applied++
					if hasPostApplyCheck(step) {
						gated++
						continue
					}
					ungated = append(ungated,
						fmt.Sprintf("%s:%d", rel, fset.Position(step.Pos()).Line))
				}
				return true
			})
		}
	}

	sort.Strings(ungated)
	t.Logf("acceptance steps that apply: %d, gated: %d, ungated: %d (ceiling %d)",
		applied, gated, len(ungated), ungatedStepCeiling)

	if len(ungated) > ungatedStepCeiling {
		shown := ungated
		if len(shown) > 20 {
			shown = shown[:20]
		}
		t.Errorf("%d acceptance steps apply a config without a ConfigPlanChecks gate, "+
			"ceiling is %d. A step that applies must assert the plan settles, or the "+
			"resource can drift on every plan without any test noticing. Add:\n"+
			"    ConfigPlanChecks: resource.ConfigPlanChecks{\n"+
			"        PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},\n"+
			"    },\nFirst 20 of %d:\n  %v",
			len(ungated), ungatedStepCeiling, len(ungated), shown)
	}
	if len(ungated) < ungatedStepCeiling {
		t.Errorf("ungated acceptance steps dropped to %d, below the ceiling of %d. "+
			"Lower ungatedStepCeiling to %d so the ratchet holds the ground you gained.",
			len(ungated), ungatedStepCeiling, len(ungated))
	}
}

// hasPostApplyCheck reports whether a step sets ConfigPlanChecks with
// PostApplyPostRefresh.
//
// The presence of ConfigPlanChecks alone is not enough, and treating it as
// enough was this invariant's own bug: 54 steps carried a PreApply check that
// asserts what the plan WILL do and nothing about what it leaves behind, and
// they counted as gated while asserting nothing about idempotency.
func hasPostApplyCheck(step *ast.CompositeLit) bool {
	for _, e := range step.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		id, ok := kv.Key.(*ast.Ident)
		if !ok || id.Name != "ConfigPlanChecks" {
			continue
		}
		cl, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			return false
		}
		for _, el := range cl.Elts {
			ekv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if eid, ok := ekv.Key.(*ast.Ident); ok && eid.Name == "PostApplyPostRefresh" {
				return true
			}
		}
	}
	return false
}

// stepKeys returns the field names set on one TestStep literal.
func stepKeys(step *ast.CompositeLit) map[string]bool {
	keys := make(map[string]bool, len(step.Elts))
	for _, e := range step.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if id, ok := kv.Key.(*ast.Ident); ok {
			keys[id.Name] = true
		}
	}
	return keys
}

// stepCannotApply reports whether a step never reaches an apply whose
// idempotency could be checked.
func stepCannotApply(keys map[string]bool) bool {
	for _, k := range []string{
		"ImportState",        // import round-trip, no apply
		"PlanOnly",           // plan is the assertion
		"ExpectError",        // never gets to apply
		"ExpectNonEmptyPlan", // a follow-up diff is the point
		"RefreshState",       // refresh-only
		"Destroy",            // teardown
	} {
		if keys[k] {
			return true
		}
	}
	return false
}
