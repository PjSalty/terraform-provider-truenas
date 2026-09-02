package provider

import (
	"os"
	"path/filepath"
	"sort"
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
		t.Fatalf("the following resources do not declare a timeouts block/attribute, "+
			"operators have no way to override CRUD deadlines per-resource:\n%s\n\n"+
			"Fix by adding a timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, "+
			"Update: true, Delete: true}) entry to the resource's schema.Blocks map. "+
			"See internal/resources/dataset.go:69 for the reference pattern.",
			strings.Join(missing, "\n  "))
	}
}

// TestResourcesConsumeTimeoutsBlock is the enforcement half of
// TestResourcesHaveTimeoutsBlock above.
//
// That test checks a resource DECLARES a timeouts block. All 68 did, and not
// one of them read it: a practitioner could write `timeouts { create = "45m" }`
// and change nothing at all, because no CRUD method ever derived a context
// from the value. Presence checked, enforcement not, which is the failure mode
// this repo's own principles name and the reason GitHub #34 exists.
//
// A CRUD method is exempt only when it makes no API call, so there is nothing
// to bound. Those are listed with a reason.
func TestResourcesConsumeTimeoutsBlock(t *testing.T) {
	t.Parallel()

	// method key is "<file>::<Method>". A no-op has no call to bound.
	exempt := map[string]string{
		"lxc_config.go::Delete":    "no-op, singleton config is not removable",
		"nvmet_global.go::Delete":  "state-only removal, makes no API call",
		"system_update.go::Delete": "no-op, singleton config is not removable",
		"service.go::Delete":       "state-only removal, makes no API call",
	}

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(root, "internal", "resources", "*.go"))
	if err != nil {
		t.Fatalf("globbing resources: %v", err)
	}

	var missing []string
	checked := 0
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatalf("read %s: %v", f, readErr)
		}
		src := string(data)
		if !strings.Contains(src, "timeouts.Block(") && !strings.Contains(src, "timeouts.Attributes(") {
			continue
		}
		for _, m := range crudMethodInvariantRE.FindAllStringSubmatchIndex(src, -1) {
			method := src[m[2]:m[3]]
			key := base + "::" + method
			if _, ok := exempt[key]; ok {
				continue
			}
			body := methodBodyAfter(src, m[0])
			checked++
			if !strings.Contains(body, "planhelpers.With"+method+"Timeout(") {
				missing = append(missing, key)
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d CRUD method(s) declare a timeouts block and never read it:\n  %s\n\n"+
			"Derive the context from it, e.g.\n"+
			"  ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)\n"+
			"  defer cancel()\n"+
			"A method that makes no API call has nothing to bound; add it to exempt with a reason.",
			len(missing), strings.Join(missing, "\n  "))
	}
	if checked == 0 {
		t.Fatal("no CRUD methods were checked; the walk is broken, not the resources")
	}
	t.Logf("checked %d CRUD methods, %d exempt", checked, len(exempt))
}
