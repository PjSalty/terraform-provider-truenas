package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResourcesHaveTimeoutsBlock verifies that every resource schema
// declares a `timeouts` block. The Plugin Framework ships a first-class
// timeouts helper (github.com/hashicorp/terraform-plugin-framework-timeouts)
// that plugs directly into the schema.Block map; the established convention
// is that every resource uses it so operators can override the default
// CRUD deadlines per-resource in HCL:
//
//	resource "truenas_dataset" "slow_prod" {
//	  pool = "tank"
//	  name = "bulk"
//	  timeouts {
//	    create = "30m"
//	    read   = "5m"
//	    update = "10m"
//	    delete = "30m"
//	  }
//	}
//
// Without this block, a slow prod endpoint (e.g. pool scrub status on a
// 300TB array) will be capped at the provider-level default and operators
// have no way to raise it for the one resource they need.
//
// Currently 62/62 resources pass this check. The test exists to
// guard against a drive-by refactor silently dropping the block.
func TestResourcesHaveTimeoutsBlock(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	var missing []string
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		text := string(src)
		// Every resource file that declares a Schema must contain at
		// least one reference to the plugin-framework-timeouts package.
		// Allow either the Block() helper (most common) or the
		// Attributes() helper (used on a few resource schemas that
		// prefer flat attributes over nested blocks).
		if !strings.Contains(text, "timeouts.Block(") &&
			!strings.Contains(text, "timeouts.Attributes(") {
			missing = append(missing, base)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the following resources do not declare a timeouts block/attribute — "+
			"operators have no way to override CRUD deadlines per-resource:\n%s\n\n"+
			"Fix by adding a timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, "+
			"Update: true, Delete: true}) entry to the resource's schema.Blocks map. "+
			"See internal/resources/dataset.go:69 for the reference pattern.",
			strings.Join(missing, "\n  "))
	}
}
