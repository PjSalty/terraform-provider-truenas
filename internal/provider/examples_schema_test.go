package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestExamplesMatchSchema checks every attribute and block used in an example
// against the schema of the resource it configures, nested bodies included.
//
// TestExamplesResourceHCLValid only proves an example parses. An example can
// parse perfectly and still name an attribute the provider does not have, which
// fails at `terraform plan` for the first person who copies it out of the
// Registry. Nothing in the suite could see that: the acceptance tests use their
// own inline HCL and never read the published examples.
func TestExamplesMatchSchema(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}

	fields := resourceFields(t, context.Background())

	examplesDir := filepath.Join(root, "examples", "resources")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples: %v", err)
	}

	parser := hclparse.NewParser()
	var checked, blocks int
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "truenas_") {
			continue
		}
		path := filepath.Join(examplesDir, entry.Name(), "resource.tf")
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			continue // absence is TestDocsCoverage's job to report
		}
		f, diags := parser.ParseHCL(src, path)
		if diags.HasErrors() {
			continue // syntax is TestExamplesResourceHCLValid's job to report
		}
		body, ok := f.Body.(*hclsyntax.Body)
		if !ok {
			t.Errorf("%s: unexpected HCL body type", entry.Name())
			continue
		}

		for _, blk := range body.Blocks {
			if blk.Type != "resource" || len(blk.Labels) != 2 {
				continue
			}
			typeName := blk.Labels[0]
			known, registered := fields[typeName]
			if !registered {
				t.Errorf("%s: example configures %q, which the provider does not register",
					entry.Name(), typeName)
				continue
			}
			blocks++
			checked += len(blk.Body.Attributes) + len(blk.Body.Blocks)
			for _, problem := range checkExampleBody(blk.Body, known, typeName) {
				t.Errorf("%s: %s", entry.Name(), problem)
			}
		}
	}

	if blocks == 0 {
		t.Fatal("no example resource blocks were checked; the walk found nothing")
	}
	t.Logf("checked %d attributes and blocks across %d example resource blocks", checked, blocks)
}

// resourceFields collects, per resource type, the configurable shape of its
// schema: what each name is, and whether it can be set at all.
func resourceFields(t *testing.T, ctx context.Context) map[string]map[string]field {
	t.Helper()
	out := make(map[string]map[string]field)
	for _, fn := range (&TrueNASProvider{}).Resources(ctx) {
		r := fn()
		mResp := &resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "truenas"}, mResp)
		sResp := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, sResp)

		top := make(map[string]field, len(sResp.Schema.Attributes)+len(sResp.Schema.Blocks))
		for name, a := range sResp.Schema.Attributes {
			top[name] = field{
				typ: a.GetType(),
				// Computed without Optional means the provider owns the value
				// outright. Assigning it fails the plan with "Value for
				// unconfigurable attribute", so an example must not.
				readOnly: a.IsComputed() && !a.IsOptional(),
			}
		}
		for name, b := range sResp.Schema.Blocks {
			top[name] = field{typ: b.Type(), isBlock: true}
		}
		out[mResp.TypeName] = top
	}
	return out
}

// field is one schema entry: its type, and whether the schema models it as a
// block. The distinction is not cosmetic. Terraform requires `timeouts {}` for
// a block and `options = {}` for a nested attribute, and using the wrong one
// fails the plan, so an example has to get it right.
type field struct {
	typ      attr.Type
	isBlock  bool
	readOnly bool
}

// terraformMetaArguments are accepted in any resource block regardless of the
// provider schema, so they are not evidence of a broken example.
var terraformMetaArguments = map[string]bool{
	"depends_on": true, "count": true, "for_each": true, "provider": true,
	"lifecycle": true, "provisioner": true, "connection": true,
}

// checkExampleBody reports every name in body that known does not define or
// that uses the wrong syntax for its kind, then recurses into nested bodies.
func checkExampleBody(body *hclsyntax.Body, known map[string]field, path string) []string {
	var problems []string

	names := make([]string, 0, len(body.Attributes))
	for name := range body.Attributes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if terraformMetaArguments[name] {
			continue
		}
		f, ok := known[name]
		if !ok {
			problems = append(problems, unknownField(path, name, known))
			continue
		}
		if f.isBlock {
			problems = append(problems, fmt.Sprintf(
				"%s.%s is a block in the schema but the example assigns it; write %s { ... }",
				path, name, name))
			continue
		}
		if f.readOnly {
			problems = append(problems, fmt.Sprintf(
				"%s.%s is computed, not configurable; the example must not assign it",
				path, name))
			continue
		}
		// An object-typed attribute assigned inline still has field names to
		// check, and they are the ones most likely to be wrong.
		obj, isObj := body.Attributes[name].Expr.(*hclsyntax.ObjectConsExpr)
		child := objectFields(f.typ)
		if !isObj || child == nil {
			continue
		}
		problems = append(problems, checkObjectExpr(obj, child, path+"."+name)...)
	}

	for _, nested := range body.Blocks {
		if terraformMetaArguments[nested.Type] {
			continue
		}
		f, ok := known[nested.Type]
		if !ok {
			problems = append(problems, unknownField(path, nested.Type, known))
			continue
		}
		if !f.isBlock {
			problems = append(problems, fmt.Sprintf(
				"%s.%s is a nested attribute in the schema but the example uses block syntax; write %s = { ... }",
				path, nested.Type, nested.Type))
			continue
		}
		child := objectFields(f.typ)
		if child == nil {
			continue // not an object, so it has no field names to check
		}
		problems = append(problems, checkExampleBody(nested.Body, child, path+"."+nested.Type)...)
	}
	return problems
}

// checkObjectExpr checks the keys of an inline object against the schema. Only
// literal keys can be checked; a computed key is skipped rather than guessed at.
func checkObjectExpr(obj *hclsyntax.ObjectConsExpr, known map[string]field, path string) []string {
	var problems []string
	var names []string
	for _, item := range obj.Items {
		key, diags := item.KeyExpr.Value(nil)
		if diags.HasErrors() || key.Type().FriendlyName() != "string" {
			continue
		}
		names = append(names, key.AsString())
	}
	sort.Strings(names)
	for _, name := range names {
		if _, ok := known[name]; !ok {
			problems = append(problems, unknownField(path, name, known))
		}
	}
	return problems
}

// objectFields returns the field types of t when t is an object, or a
// collection of objects, and nil otherwise.
func objectFields(t attr.Type) map[string]field {
	switch v := t.(type) {
	case types.ObjectType:
		out := make(map[string]field, len(v.AttributeTypes()))
		for name, at := range v.AttributeTypes() {
			out[name] = field{typ: at}
		}
		return out
	case types.ListType:
		return objectFields(v.ElemType)
	case types.SetType:
		return objectFields(v.ElemType)
	}
	// A map is deliberately excluded: its keys are chosen by the practitioner,
	// so there is no fixed set of names to check them against.
	return nil
}

func unknownField(path, name string, known map[string]field) string {
	return fmt.Sprintf("%s.%s is not in the schema%s", path, name, nearestField(name, known))
}

// nearestField suggests a real field when the used one differs only in
// underscores, which is the common case: read_only for readonly.
func nearestField(used string, known map[string]field) string {
	squash := func(s string) string { return strings.ReplaceAll(s, "_", "") }
	target := squash(used)
	candidates := make([]string, 0, 1)
	for name := range known {
		if squash(name) == target {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Strings(candidates)
	return fmt.Sprintf(" (did you mean %q?)", candidates[0])
}

// TestDocsExamplesMatchSchema runs the same schema check over every fenced
// terraform snippet in docs/resources/, which is the copy people actually read
// on the Registry.
//
// The docs here are hand-authored rather than generated, deliberately:
// `tfplugindocs generate` strips the custom subcategory and prose. The cost is
// that a doc snippet is a copy, and a copy drifts. This does not force it to
// equal resource.tf, since several docs carry richer or narrower variants on
// purpose. It only requires that whatever they show would actually plan.
func TestDocsExamplesMatchSchema(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	fields := resourceFields(t, context.Background())

	docsDir := filepath.Join(root, "docs", "resources")
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("reading docs: %v", err)
	}

	fence := regexp.MustCompile("(?s)```(?:terraform|hcl)\\n(.*?)\\n```")
	parser := hclparse.NewParser()
	var snippets int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(docsDir, entry.Name())
		doc, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("%s: %v", entry.Name(), readErr)
			continue
		}
		for i, m := range fence.FindAllSubmatch(doc, -1) {
			f, diags := parser.ParseHCL(m[1], fmt.Sprintf("%s#%d", path, i))
			if diags.HasErrors() {
				t.Errorf("%s: snippet %d does not parse:\n%s", entry.Name(), i, diags.Error())
				continue
			}
			body, ok := f.Body.(*hclsyntax.Body)
			if !ok {
				continue
			}
			for _, blk := range body.Blocks {
				if blk.Type != "resource" || len(blk.Labels) != 2 {
					continue
				}
				known, registered := fields[blk.Labels[0]]
				if !registered {
					t.Errorf("%s: snippet %d configures %q, which the provider does not register",
						entry.Name(), i, blk.Labels[0])
					continue
				}
				snippets++
				for _, problem := range checkExampleBody(blk.Body, known, blk.Labels[0]) {
					t.Errorf("%s snippet %d: %s", entry.Name(), i, problem)
				}
			}
		}
	}
	if snippets == 0 {
		t.Fatal("no doc snippets were checked; the walk found nothing")
	}
	t.Logf("checked %d resource snippets across the docs", snippets)
}
