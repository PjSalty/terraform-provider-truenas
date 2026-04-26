package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// iscsiGroupObjType is the tftypes.Object for a single target group entry
// — it must match the schema order so the framework accepts the raw value.
func iscsiGroupObjType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"portal":      tftypes.Number,
			"initiator":   tftypes.Number,
			"auth_method": tftypes.String,
			"auth":        tftypes.Number,
		},
	}
}

func iscsiGroupValue(portal, initiator int64, method string, auth int64) tftypes.Value {
	return tftypes.NewValue(iscsiGroupObjType(), map[string]tftypes.Value{
		"portal":      num(portal),
		"initiator":   num(initiator),
		"auth_method": str(method),
		"auth":        num(auth),
	})
}

func iscsiGroupList(vals ...tftypes.Value) tftypes.Value {
	return tftypes.NewValue(
		tftypes.List{ElementType: iscsiGroupObjType()},
		vals,
	)
}

func TestISCSITarget_ModifyPlan_CHAPNoAuth(t *testing.T) {
	r := NewISCSITargetResource().(*ISCSITargetResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":   str("iqn-test"),
		"groups": iscsiGroupList(iscsiGroupValue(1, 1, "CHAP", 0)),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for CHAP with auth=0")
	}
}

func TestISCSITarget_ModifyPlan_CHAPMutualNoAuth(t *testing.T) {
	r := NewISCSITargetResource().(*ISCSITargetResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":   str("iqn-test"),
		"groups": iscsiGroupList(iscsiGroupValue(1, 1, "CHAP_MUTUAL", 0)),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for CHAP_MUTUAL with auth=0")
	}
}

func TestISCSITarget_ModifyPlan_CHAPWithAuth(t *testing.T) {
	r := NewISCSITargetResource().(*ISCSITargetResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":   str("iqn-test"),
		"groups": iscsiGroupList(iscsiGroupValue(1, 1, "CHAP", 5)),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestISCSITarget_ModifyPlan_NoneMethodOK(t *testing.T) {
	r := NewISCSITargetResource().(*ISCSITargetResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":   str("iqn-test"),
		"groups": iscsiGroupList(iscsiGroupValue(1, 1, "NONE", 0)),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestISCSITarget_ModifyPlan_NoGroups(t *testing.T) {
	r := NewISCSITargetResource().(*ISCSITargetResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("iqn-test"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestISCSITarget_ModifyPlan_Delete(t *testing.T) {
	r := NewISCSITargetResource().(*ISCSITargetResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}
