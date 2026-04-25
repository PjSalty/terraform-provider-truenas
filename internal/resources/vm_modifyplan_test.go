package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestVM_ModifyPlan_CustomCPUNoModel(t *testing.T) {
	r := NewVMResource().(*VMResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":     str("testvm"),
		"memory":   num(1024),
		"cpu_mode": str("CUSTOM"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for cpu_mode=CUSTOM without cpu_model")
	}
	found := false
	for _, e := range resp.Diagnostics.Errors() {
		if e.Summary() == "Missing cpu_model" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Missing cpu_model' diag, got: %v", resp.Diagnostics)
	}
}

func TestVM_ModifyPlan_MinMemGEMemory(t *testing.T) {
	r := NewVMResource().(*VMResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":       str("testvm"),
		"memory":     num(1024),
		"min_memory": num(1024),
		"cpu_mode":   str("HOST-MODEL"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for min_memory >= memory")
	}
}

func TestVM_ModifyPlan_MinMemGreaterThanMemory(t *testing.T) {
	r := NewVMResource().(*VMResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":       str("testvm"),
		"memory":     num(1024),
		"min_memory": num(2048),
		"cpu_mode":   str("HOST-MODEL"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for min_memory > memory")
	}
}

func TestVM_ModifyPlan_PinVCPUsNoCpuset(t *testing.T) {
	r := NewVMResource().(*VMResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":      str("testvm"),
		"memory":    num(1024),
		"cpu_mode":  str("HOST-MODEL"),
		"pin_vcpus": flag(true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for pin_vcpus=true without cpuset")
	}
}

func TestVM_ModifyPlan_Valid(t *testing.T) {
	r := NewVMResource().(*VMResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":       str("testvm"),
		"memory":     num(2048),
		"min_memory": num(1024),
		"cpu_mode":   str("CUSTOM"),
		"cpu_model":  str("Broadwell"),
		"pin_vcpus":  flag(true),
		"cpuset":     str("0-3"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestVM_ModifyPlan_Delete(t *testing.T) {
	r := NewVMResource().(*VMResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}
