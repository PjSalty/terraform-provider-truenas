package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestReplication_ModifyPlan_SSHWithoutCreds(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":      str("rep1"),
		"transport": str("SSH"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestReplication_ModifyPlan_SSHNetcatWithoutCreds(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":      str("rep1"),
		"transport": str("SSH+NETCAT"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestReplication_ModifyPlan_LocalValid(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":      str("rep1"),
		"transport": str("LOCAL"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestReplication_ModifyPlan_SSHWithCreds(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":            str("rep1"),
		"transport":       str("SSH"),
		"ssh_credentials": num(5),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestReplication_ModifyPlan_SSHWithUnknownCreds(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":            str("rep1"),
		"transport":       str("SSH"),
		"ssh_credentials": unknownOf(tftypes.Number),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestReplication_ModifyPlan_CustomRetentionNoLifetime(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":             str("rep1"),
		"transport":        str("LOCAL"),
		"retention_policy": str("CUSTOM"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for CUSTOM retention without lifetime")
	}
}

func TestReplication_ModifyPlan_CustomRetentionComplete(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":             str("rep1"),
		"transport":        str("LOCAL"),
		"retention_policy": str("CUSTOM"),
		"lifetime_value":   num(30),
		"lifetime_unit":    str("DAY"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestReplication_ModifyPlan_Delete(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// lifetime_value and lifetime_unit wired from another resource are unknown at
// plan time. They are set, with the value pending, so CUSTOM retention must
// not reject them as missing.
func TestReplication_ModifyPlan_CustomRetentionUnknownLifetime(t *testing.T) {
	cases := map[string]map[string]tftypes.Value{
		"both unknown": {
			"lifetime_value": unknownOf(tftypes.Number),
			"lifetime_unit":  unknownOf(tftypes.String),
		},
		"value unknown": {
			"lifetime_value": unknownOf(tftypes.Number),
			"lifetime_unit":  str("DAY"),
		},
		"unit unknown": {
			"lifetime_value": num(30),
			"lifetime_unit":  unknownOf(tftypes.String),
		},
	}
	for name, lifetime := range cases {
		t.Run(name, func(t *testing.T) {
			vals := map[string]tftypes.Value{
				"name":             str("rep1"),
				"transport":        str("LOCAL"),
				"retention_policy": str("CUSTOM"),
			}
			for k, v := range lifetime {
				vals[k] = v
			}
			r := NewReplicationResource().(*ReplicationResource)
			resp := callModifyPlan(t, r, vals)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected: %v", resp.Diagnostics)
			}
		})
	}
}

// A known empty lifetime_unit is still missing.
func TestReplication_ModifyPlan_CustomRetentionEmptyUnit(t *testing.T) {
	r := NewReplicationResource().(*ReplicationResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":             str("rep1"),
		"transport":        str("LOCAL"),
		"retention_policy": str("CUSTOM"),
		"lifetime_value":   num(30),
		"lifetime_unit":    str(""),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for CUSTOM retention with an empty lifetime_unit")
	}
}
