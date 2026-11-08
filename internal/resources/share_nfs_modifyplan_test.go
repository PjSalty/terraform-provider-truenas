package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

// One side of a pair wired from another resource (for example
// maproot_user = truenas_user.x.username) is unknown at plan time. The pair is
// not incomplete, and an unknown side does not count toward the conflict.
func TestNFSShare_ModifyPlan_UnknownMapping(t *testing.T) {
	cases := map[string]map[string]tftypes.Value{
		"maproot user unknown": {
			"maproot_user":  unknownOf(tftypes.String),
			"maproot_group": str("wheel"),
		},
		"maproot group unknown": {
			"maproot_user":  str("root"),
			"maproot_group": unknownOf(tftypes.String),
		},
		"mapall user unknown": {
			"mapall_user":  unknownOf(tftypes.String),
			"mapall_group": str("nogroup"),
		},
		"mapall group unknown": {
			"mapall_user":  str("nobody"),
			"mapall_group": unknownOf(tftypes.String),
		},
		"mapall known, maproot unknown": {
			"mapall_user":   str("nobody"),
			"mapall_group":  str("nogroup"),
			"maproot_user":  unknownOf(tftypes.String),
			"maproot_group": unknownOf(tftypes.String),
		},
	}
	for name, mapping := range cases {
		t.Run(name, func(t *testing.T) {
			vals := map[string]tftypes.Value{"path": str("/mnt/tank/data")}
			for k, v := range mapping {
				vals[k] = v
			}
			r := NewNFSShareResource().(*NFSShareResource)
			resp := callModifyPlan(t, r, vals)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected: %v", resp.Diagnostics)
			}
		})
	}
}

// A known empty side is still missing, and the error names that side.
func TestNFSShare_ModifyPlan_EmptySideTargeted(t *testing.T) {
	cases := []struct {
		name string
		vals map[string]tftypes.Value
		want path.Path
	}{
		{"maproot user empty", map[string]tftypes.Value{
			"maproot_user": str(""), "maproot_group": str("wheel"),
		}, path.Root("maproot_user")},
		{"maproot group empty", map[string]tftypes.Value{
			"maproot_user": str("root"), "maproot_group": str(""),
		}, path.Root("maproot_group")},
		{"mapall user empty", map[string]tftypes.Value{
			"mapall_user": str(""), "mapall_group": str("nogroup"),
		}, path.Root("mapall_user")},
		{"mapall group empty", map[string]tftypes.Value{
			"mapall_user": str("nobody"), "mapall_group": str(""),
		}, path.Root("mapall_group")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.vals["path"] = str("/mnt/tank/data")
			r := NewNFSShareResource().(*NFSShareResource)
			resp := callModifyPlan(t, r, tc.vals)
			errs := resp.Diagnostics.Errors()
			if len(errs) != 1 {
				t.Fatalf("want one error, got: %v", resp.Diagnostics)
			}
			wp, ok := errs[0].(diag.DiagnosticWithPath)
			if !ok || !wp.Path().Equal(tc.want) {
				t.Errorf("want error on %s, got: %v", tc.want, errs[0])
			}
		})
	}
}
