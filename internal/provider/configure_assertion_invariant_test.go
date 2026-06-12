package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// configureTypeAssertRE matches every Configure body's
// `req.ProviderData.(*wsclient.Client)` type assertion. The pattern
// is intentionally narrow, we want to catch the canonical shape
// and flag deviations.
var configureTypeAssertRE = regexp.MustCompile(`req\.ProviderData\.\(\*(\w+)\.Client\)`)

// wrongPackageImportRE catches resources that import the deleted
// internal/client package directly, which would be a regression
// after the v2.0 WS cutover.
var wrongPackageImportRE = regexp.MustCompile(`"github\.com/PjSalty/terraform-provider-truenas/internal/client"`)

// TestConfigureUsesWSClient verifies every internal/resources/*.go
// file's Configure type-asserts *wsclient.Client (not *client.Client
// or any other type). The v2.0 cutover replaced the REST
// *client.Client with *wsclient.Client across the resource layer;
// this invariant catches a regression where a new resource is
// authored against the deleted REST type or someone re-introduces
// the old import.
//
// Why this matters: the type assertion failing at runtime is
// surfaced as a per-resource Configure diagnostic, which is hard to
// catch in unit tests but trivially caught here at the source-text
// level.
func TestConfigureUsesWSClient(t *testing.T) {
	root := filepath.Join("..", "resources")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("readdir %s: %v", root, err)
	}

	var offenders []string
	var importOffenders []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		full := filepath.Join(root, name)
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("read %s: %v", full, err)
		}
		src := string(data)

		// Every match of req.ProviderData.(*X.Client) must have X == wsclient.
		for _, m := range configureTypeAssertRE.FindAllStringSubmatch(src, -1) {
			pkg := m[1]
			if pkg != "wsclient" {
				offenders = append(offenders, name+": asserts *"+pkg+".Client (expected *wsclient.Client)")
			}
		}

		// And nobody should import the deleted REST package.
		if wrongPackageImportRE.MatchString(src) {
			importOffenders = append(importOffenders, name)
		}
	}

	sort.Strings(offenders)
	sort.Strings(importOffenders)

	if len(offenders) > 0 {
		t.Errorf("non-canonical type assertions found:\n  %s\n\n"+
			"Every resource's Configure must type-assert "+
			"*wsclient.Client. The v2.0 cutover removed *client.Client; "+
			"anything else is dead code that will fail at runtime.",
			strings.Join(offenders, "\n  "))
	}
	if len(importOffenders) > 0 {
		t.Errorf("stale internal/client import found in:\n  %s\n\n"+
			"The internal/client package was deleted in the v2.0 cutover. "+
			"Import internal/wsclient instead (and internal/types if you "+
			"need struct shapes, aliased as `truenas` in the resource "+
			"convention to avoid collision with the plugin-framework "+
			"types package).",
			strings.Join(importOffenders, "\n  "))
	}
}

// TestDataSourceConfigureUsesWSClient is the datasource-side mirror
// of TestConfigureUsesWSClient. Same logic, different directory.
func TestDataSourceConfigureUsesWSClient(t *testing.T) {
	root := filepath.Join("..", "datasources")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("readdir %s: %v", root, err)
	}

	var offenders []string
	var importOffenders []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		full := filepath.Join(root, name)
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("read %s: %v", full, err)
		}
		src := string(data)

		for _, m := range configureTypeAssertRE.FindAllStringSubmatch(src, -1) {
			pkg := m[1]
			if pkg != "wsclient" {
				offenders = append(offenders, name+": asserts *"+pkg+".Client")
			}
		}

		if wrongPackageImportRE.MatchString(src) {
			importOffenders = append(importOffenders, name)
		}
	}

	sort.Strings(offenders)
	sort.Strings(importOffenders)

	if len(offenders) > 0 {
		t.Errorf("datasource non-canonical type assertions:\n  %s",
			strings.Join(offenders, "\n  "))
	}
	if len(importOffenders) > 0 {
		t.Errorf("datasource stale internal/client import:\n  %s",
			strings.Join(importOffenders, "\n  "))
	}
}
