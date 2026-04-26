package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNFSShare_ModifyPlan_MapallUserWithoutGroup(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"path":        str("/mnt/tank/data"),
		"mapall_user": str("nobody"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for mapall_user without mapall_group")
	}
}

func TestNFSShare_ModifyPlan_MapallGroupWithoutUser(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"path":         str("/mnt/tank/data"),
		"mapall_group": str("nogroup"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for mapall_group without mapall_user")
	}
}

func TestNFSShare_ModifyPlan_MaprootIncomplete(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"path":         str("/mnt/tank/data"),
		"maproot_user": str("root"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for maproot_user without maproot_group")
	}
}

func TestNFSShare_ModifyPlan_MaprootGroupWithoutUser(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"path":          str("/mnt/tank/data"),
		"maproot_group": str("wheel"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestNFSShare_ModifyPlan_MapallAndMaproot(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"path":          str("/mnt/tank/data"),
		"mapall_user":   str("nobody"),
		"mapall_group":  str("nogroup"),
		"maproot_user":  str("root"),
		"maproot_group": str("wheel"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for mapall + maproot conflict")
	}
}

func TestNFSShare_ModifyPlan_Valid(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"path":         str("/mnt/tank/data"),
		"mapall_user":  str("nobody"),
		"mapall_group": str("nogroup"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestNFSShare_ModifyPlan_Delete(t *testing.T) {
	r := NewNFSShareResource().(*NFSShareResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}
