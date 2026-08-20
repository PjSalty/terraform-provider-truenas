package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ModifyPlan keeps the deprecated enable_smb1 and its replacement
// minimum_protocol in lockstep, so buildUpdateRequest only ever has to read
// the modern one. These pin the translation in both directions.

func TestSMBConfigResource_ModifyPlan_derivesProtocolFromLegacyBool(t *testing.T) {
	r := &SMBConfigResource{}

	cases := []struct {
		name string
		smb1 bool
		want string
	}{
		// Same mapping middleware's own 26.0 migration uses.
		{"true becomes SMB1", true, truenas.SMBProtocolSMB1},
		{"false becomes SMB2", false, truenas.SMBProtocolSMB2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := callModifyPlan(t, r, map[string]tftypes.Value{
				"enable_smb1": tftypes.NewValue(tftypes.Bool, tc.smb1),
			})
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
			}
			var plan SMBConfigResourceModel
			resp.Plan.Get(context.Background(), &plan)
			if plan.MinimumProtocol.ValueString() != tc.want {
				t.Errorf("minimum_protocol = %q, want %q", plan.MinimumProtocol.ValueString(), tc.want)
			}
		})
	}
}

func TestSMBConfigResource_ModifyPlan_derivesLegacyBoolFromProtocol(t *testing.T) {
	r := &SMBConfigResource{}

	cases := []struct {
		proto string
		want  bool
	}{
		{truenas.SMBProtocolSMB1, true},
		{truenas.SMBProtocolSMB2, false},
		// SMB3 is not SMB1, so the legacy mirror is false. It is only a
		// mirror for state consistency; the wire value is the protocol.
		{truenas.SMBProtocolSMB3, false},
	}
	for _, tc := range cases {
		t.Run(tc.proto, func(t *testing.T) {
			resp := callModifyPlan(t, r, map[string]tftypes.Value{
				"minimum_protocol": tftypes.NewValue(tftypes.String, tc.proto),
			})
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
			}
			var plan SMBConfigResourceModel
			resp.Plan.Get(context.Background(), &plan)
			if plan.EnableSMB1.ValueBool() != tc.want {
				t.Errorf("enable_smb1 = %v, want %v", plan.EnableSMB1.ValueBool(), tc.want)
			}
		})
	}
}

// Declaring neither must leave both alone. UseStateForUnknown has already
// filled them from the refreshed server values, and overwriting here would
// invent a protocol the user never asked for.
func TestSMBConfigResource_ModifyPlan_neitherDeclaredLeavesBoth(t *testing.T) {
	ctx := context.Background()
	r := &SMBConfigResource{}
	resp := callModifyPlan(t, r, map[string]tftypes.Value{})
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
	}
	var plan SMBConfigResourceModel
	resp.Plan.Get(ctx, &plan)
	if !plan.MinimumProtocol.IsNull() {
		t.Errorf("minimum_protocol was invented as %q", plan.MinimumProtocol.ValueString())
	}
	if !plan.EnableSMB1.IsNull() {
		t.Errorf("enable_smb1 was invented as %v", plan.EnableSMB1.ValueBool())
	}
}

// Destroy has a null plan; touching it would panic.
func TestSMBConfigResource_ModifyPlan_destroyIsANoop(t *testing.T) {
	r := &SMBConfigResource{}
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Errorf("ModifyPlan on destroy: %v", resp.Diagnostics)
	}
}

// buildUpdateRequest must never look at the deprecated attribute. If it did,
// a plan carrying only enable_smb1 (which cannot happen after ModifyPlan, but
// would if the two ever drifted) would send the legacy key to a 26.0 box.
func TestSMBConfigResource_BuildUpdateRequest_onlyReadsProtocol(t *testing.T) {
	r := &SMBConfigResource{}
	req := r.buildUpdateRequest(&SMBConfigResourceModel{
		MinimumProtocol: types.StringValue(truenas.SMBProtocolSMB3),
	})
	if req.MinimumProtocol == nil || *req.MinimumProtocol != truenas.SMBProtocolSMB3 {
		t.Fatalf("MinimumProtocol = %v, want SMB3", req.MinimumProtocol)
	}
}
