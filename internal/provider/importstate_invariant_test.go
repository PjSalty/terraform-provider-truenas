package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// importStateDeclRE matches the framework interface-satisfaction line
// that resources include when they support `terraform import`:
//
//	_ resource.ResourceWithImportState = &<Type>{}
//
// We use this as the "this resource is importable" signal rather than
// looking for the ImportState method itself, because the interface
// assertion is what Terraform's plugin framework actually wires up.
var importStateDeclRE = regexp.MustCompile(`resource\.ResourceWithImportState\s*=\s*&\w+\{\}`)

// passthroughRE matches the framework-provided import helpers that
// most resources delegate to:
//
//	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
//	resource.ImportStatePassthroughString(ctx, path.Root("name"), req, resp)
//
// A resource that opts out of the passthrough helpers and implements
// ImportState by hand must include a literal "// import: custom" tag
// in the file (so this invariant doesn't fail on intentional custom
// implementations). At time of writing zero resources need that
// escape hatch.
var passthroughRE = regexp.MustCompile(`resource\.ImportStatePassthrough(ID|String)\b`)

// TestResourcesHaveImportStateImplemented verifies that every resource
// declaring itself ResourceWithImportState actually provides a working
// ImportState implementation. The framework's interface alone does not
// catch a resource that registers the interface and then forgets to
// wire the body, which surfaces as a confusing "Resource Import Not
// Implemented" runtime error rather than a build/test failure.
//
// The check is intentionally cheap: look for the framework's passthrough
// helper in the same file as the interface assertion. A custom
// implementation must opt out with a `// import: custom` comment so the
// invariant remains useful as a drive-by-refactor guard.
//
// This is the same shape of static check as TestResourcesHaveTimeouts
// Block and TestResourceRequiresReplaceConsistent — it runs in the
// unit-test layer with no real TrueNAS required, so it gates every PR
// without needing the acceptance test VM.
func TestResourcesHaveImportStateImplemented(t *testing.T) {
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

		// Resource doesn't claim importability — skip. Singleton
		// resources (e.g. ssh_config) intentionally do not implement
		// ResourceWithImportState because there's nothing to import.
		if !importStateDeclRE.MatchString(text) {
			continue
		}
		// Custom implementation explicitly opted out of the
		// passthrough helpers. Accept and move on.
		if strings.Contains(text, "// import: custom") {
			continue
		}
		// Importable resource with no passthrough call — bug.
		if !passthroughRE.MatchString(text) {
			missing = append(missing, base)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the following resources declare ResourceWithImportState but do not "+
			"use the framework's ImportStatePassthrough{ID,String} helper. Operators "+
			"running `terraform import` against these will see runtime errors:\n"+
			"  %s\n\n"+
			"Fix by adding the standard one-liner in the ImportState method:\n"+
			"  resource.ImportStatePassthroughID(ctx, path.Root(\"id\"), req, resp)\n\n"+
			"If the resource genuinely needs a custom import path (e.g. composite "+
			"IDs), add a `// import: custom` comment near the method to opt out of "+
			"this check.",
			strings.Join(missing, "\n  "))
	}
}
