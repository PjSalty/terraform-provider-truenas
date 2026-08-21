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
	req := r.buildUpdateRequest(context.Background(), &SMBConfigResourceModel{
		MinimumProtocol: types.StringValue(truenas.SMBProtocolSMB3),
	})
	if req.MinimumProtocol == nil || *req.MinimumProtocol != truenas.SMBProtocolSMB3 {
		t.Fatalf("MinimumProtocol = %v, want SMB3", req.MinimumProtocol)
	}
}

// --- search_protocols, new in TrueNAS 26.0 ---

func TestSMBConfigResource_searchProtocolsRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &SMBConfigResource{}

	// Set in the plan: forwarded to the request.
	spVal, _ := types.ListValueFrom(ctx, types.StringType, []string{"SPOTLIGHT"})
	req := r.buildUpdateRequest(ctx, &SMBConfigResourceModel{
		MinimumProtocol: types.StringValue(truenas.SMBProtocolSMB2),
		SearchProtocols: spVal,
	})
	if req.SearchProtocols == nil || len(*req.SearchProtocols) != 1 {
		t.Fatalf("search_protocols not forwarded: %v", req.SearchProtocols)
	}

	// Unset: omitted, so it never reaches a server without the field.
	req2 := r.buildUpdateRequest(ctx, &SMBConfigResourceModel{
		MinimumProtocol: types.StringValue(truenas.SMBProtocolSMB2),
		SearchProtocols: types.ListNull(types.StringType),
	})
	if req2.SearchProtocols != nil {
		t.Error("search_protocols sent for a plan that did not set it")
	}
}

// A server without the field must read back a known empty list, not null,
// or every plan on 25.10 shows a phantom diff.
func TestSMBConfigResource_searchProtocolsAbsentReadsEmpty(t *testing.T) {
	ctx := context.Background()
	r := &SMBConfigResource{}
	var m SMBConfigResourceModel
	r.mapResponseToModel(ctx, &truenas.SMBConfig{
		Protocol: truenas.SMBProtocolSMB2, SearchProtocols: nil,
	}, &m)
	if m.SearchProtocols.IsNull() || m.SearchProtocols.IsUnknown() {
		t.Error("search_protocols read back null/unknown from a server without the field")
	}
	if len(m.SearchProtocols.Elements()) != 0 {
		t.Errorf("expected an empty list, got %v", m.SearchProtocols)
	}

	sp := []string{"SPOTLIGHT"}
	var m2 SMBConfigResourceModel
	r.mapResponseToModel(ctx, &truenas.SMBConfig{
		Protocol: truenas.SMBProtocolSMB2, SearchProtocols: &sp,
	}, &m2)
	if len(m2.SearchProtocols.Elements()) != 1 {
		t.Errorf("search_protocols did not round-trip: %v", m2.SearchProtocols)
	}
}
