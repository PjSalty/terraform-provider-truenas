package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestISCSIExtent_ModifyPlan_DiskMissingDisk(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("e1"),
		"type": str("DISK"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestISCSIExtent_ModifyPlan_FileMissingPath(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":     str("e1"),
		"type":     str("FILE"),
		"filesize": num(1073741824),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for FILE without path")
	}
	found := false
	for _, e := range resp.Diagnostics.Errors() {
		if e.Summary() == "Missing path" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Missing path diag: %v", resp.Diagnostics)
	}
}

func TestISCSIExtent_ModifyPlan_FileMissingFilesize(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("e1"),
		"type": str("FILE"),
		"path": str("/mnt/tank/vol1.img"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for FILE without filesize")
	}
}

func TestISCSIExtent_ModifyPlan_DiskValid(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("e1"),
		"type": str("DISK"),
		"disk": str("zvol/tank/v1"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestISCSIExtent_ModifyPlan_FileValid(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":     str("e1"),
		"type":     str("FILE"),
		"path":     str("/mnt/tank/v1.img"),
		"filesize": num(1073741824),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestISCSIExtent_ModifyPlan_UnknownType(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	// No type set, should not error on cross-attrs
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("e1"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// TestISCSIExtent_ModifyPlan_PathUnknownDefers covers the unknown
// carve-out: a wired-from-sibling `path` (e.g.
// `path = truenas_dataset.x.mount_point`) is Unknown at plan time and
// the hook must defer to apply, not flag a spurious "Missing path".
func TestISCSIExtent_ModifyPlan_PathUnknownDefers(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":     str("e1"),
		"type":     str("FILE"),
		"path":     tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"filesize": num(1073741824),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error for unknown path, got: %v", resp.Diagnostics)
	}
}

// TestISCSIExtent_ModifyPlan_FilesizeUnknownDefers covers the same
// pattern when `filesize` is the unknown attribute.
func TestISCSIExtent_ModifyPlan_FilesizeUnknownDefers(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":     str("e1"),
		"type":     str("FILE"),
		"path":     str("/mnt/test/file.img"),
		"filesize": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error for unknown filesize, got: %v", resp.Diagnostics)
	}
}

// TestISCSIExtent_ModifyPlan_DiskUnknownDefers covers the disk path.
func TestISCSIExtent_ModifyPlan_DiskUnknownDefers(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("e1"),
		"type": str("DISK"),
		"disk": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error for unknown disk, got: %v", resp.Diagnostics)
	}
}

func TestISCSIExtent_ModifyPlan_Delete(t *testing.T) {
	r := NewISCSIExtentResource().(*ISCSIExtentResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}
