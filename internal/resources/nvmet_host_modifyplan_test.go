package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNVMetHost_ModifyPlan_KeyWithoutHash(t *testing.T) {
	r := NewNVMetHostResource().(*NVMetHostResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"hostnqn":    str("nqn.2024-01.com.example:host"),
		"dhchap_key": str("secret-key"),
		// dhchap_hash not set
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when dhchap_key set without dhchap_hash")
	}
	if len(resp.Diagnostics.Errors()) == 0 || resp.Diagnostics.Errors()[0].Summary() != "Missing dhchap_hash" {
		t.Errorf("unexpected diag: %v", resp.Diagnostics)
	}
}

func TestNVMetHost_ModifyPlan_CtrlKeyWithoutKey(t *testing.T) {
	r := NewNVMetHostResource().(*NVMetHostResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"hostnqn":         str("nqn.2024-01.com.example:host"),
		"dhchap_ctrl_key": str("ctrl-secret"),
		// dhchap_key not set
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when dhchap_ctrl_key set without dhchap_key")
	}
	foundCtrl := false
	for _, e := range resp.Diagnostics.Errors() {
		if e.Summary() == "Missing dhchap_key" {
			foundCtrl = true
		}
	}
	if !foundCtrl {
		t.Errorf("expected 'Missing dhchap_key' diag, got: %v", resp.Diagnostics)
	}
}

func TestNVMetHost_ModifyPlan_Valid(t *testing.T) {
	r := NewNVMetHostResource().(*NVMetHostResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"hostnqn":     str("nqn.2024-01.com.example:host"),
		"dhchap_key":  str("secret-key"),
		"dhchap_hash": str("SHA-256"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestNVMetHost_ModifyPlan_NoAuth(t *testing.T) {
	r := NewNVMetHostResource().(*NVMetHostResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"hostnqn": str("nqn.2024-01.com.example:host"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestNVMetHost_ModifyPlan_Delete(t *testing.T) {
	r := NewNVMetHostResource().(*NVMetHostResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on delete: %v", resp.Diagnostics)
	}
}
