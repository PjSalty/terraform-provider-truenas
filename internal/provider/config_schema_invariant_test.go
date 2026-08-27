package provider

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// hclBlock matches a Go raw string that contains terraform configuration.
// Acceptance configs are written as backtick literals, often through
// fmt.Sprintf.
var hclBlock = regexp.MustCompile("(?s)`\\s*\\n((?:[^`]*?)(?:resource|data|provider)\\s+\"[^\"]+\"[^`]*)`")

// goFormatVerb matches the Sprintf verbs those configs interpolate. They are
// replaced with literals of the right shape so the result parses as HCL.
var goFormatVerb = regexp.MustCompile(`%[-+# 0-9.]*[sqdtvf]`)

// TestAcceptanceConfigsMatchSchema checks the inline terraform configuration in
// every acceptance test against the schema of the resource or data source it
// configures.
//
// TestExamplesMatchSchema covers examples/ and TestDocsExamplesMatchSchema
// covers docs/, and neither sees the configurations the tests themselves use.
// That gap hid a real one: catalog_data_test.go set `id = "TRUENAS"` on a data
// source whose id is Computed, so the step failed with "Cannot set value for
// this attribute as the provider has marked it as read-only". Nothing caught
// it offline because the test was env-gated and had not run in a long time.
//
// A block that does not parse is skipped rather than reported. The verb
// substitution below is a best effort, and a parse failure is more likely to
// be this test's fault than the configuration's.
func TestAcceptanceConfigsMatchSchema(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	ctx := context.Background()
	known := resourceFields(t, ctx)
	for name, f := range dataSourceFields(t, ctx) {
		known["data."+name] = f
	}

	var files []string
	for _, dir := range []string{"provider", "resources", "datasources"} {
		found, globErr := filepath.Glob(filepath.Join(root, "internal", dir, "*_test.go"))
		if globErr != nil {
			t.Fatalf("globbing %s: %v", dir, globErr)
		}
		files = append(files, found...)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}

	parser := hclparse.NewParser()
	var checked, parsed int
	for _, file := range files {
		src, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Errorf("%s: %v", filepath.Base(file), readErr)
			continue
		}
		for i, m := range hclBlock.FindAllSubmatch(src, -1) {
			cfg := goFormatVerb.ReplaceAllFunc(m[1], func(v []byte) []byte {
				switch v[len(v)-1] {
				case 'q':
					return []byte(`"x"`)
				case 'd':
					return []byte("1")
				case 't':
					return []byte("true")
				case 'f':
					return []byte("1.0")
				default:
					return []byte("x")
				}
			})
			f, diags := parser.ParseHCL(cfg, filepath.Base(file))
			if diags.HasErrors() {
				continue // see the doc comment: our substitution, not their config
			}
			body, ok := f.Body.(*hclsyntax.Body)
			if !ok {
				continue
			}
			parsed++
			for _, blk := range body.Blocks {
				var key string
				switch {
				case blk.Type == "resource" && len(blk.Labels) == 2:
					key = blk.Labels[0]
				case blk.Type == "data" && len(blk.Labels) == 2:
					key = "data." + blk.Labels[0]
				default:
					continue // provider blocks and anything else
				}
				fields, registered := known[key]
				if !registered {
					continue // a type this provider does not serve
				}
				checked++
				for _, problem := range checkExampleBody(blk.Body, fields, key) {
					t.Errorf("%s block %d: %s", filepath.Base(file), i, problem)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no acceptance configurations were checked; the walk found nothing")
	}
	t.Logf("checked %d resource and data blocks across %d parsed configurations", checked, parsed)
}

// dataSourceFields is the data source counterpart of resourceFields.
func dataSourceFields(t *testing.T, ctx context.Context) map[string]map[string]field {
	t.Helper()
	out := make(map[string]map[string]field)
	for _, fn := range (&TrueNASProvider{}).DataSources(ctx) {
		d := fn()
		mResp := &datasource.MetadataResponse{}
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "truenas"}, mResp)
		sResp := &datasource.SchemaResponse{}
		d.Schema(ctx, datasource.SchemaRequest{}, sResp)

		top := make(map[string]field, len(sResp.Schema.Attributes)+len(sResp.Schema.Blocks))
		for name, a := range sResp.Schema.Attributes {
			top[name] = field{
				typ:      a.GetType(),
				readOnly: a.IsComputed() && !a.IsOptional(),
			}
		}
		for name, b := range sResp.Schema.Blocks {
			top[name] = field{typ: b.Type(), isBlock: true}
		}
		out[strings.TrimPrefix(mResp.TypeName, "")] = top
	}
	return out
}

var _ = attr.Type(nil)
