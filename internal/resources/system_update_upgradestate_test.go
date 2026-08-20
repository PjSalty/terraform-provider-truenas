package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The v0 -> v1 migration is a real one, not the seed-pattern no-op: the
// resource was rewritten off methods that never existed onto update.config,
// which renamed auto_download to autocheck and replaced trains with profiles.
func TestSystemUpdateResource_UpgradeState_v0(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}

	upgraders := r.UpgradeState(ctx)
	up, ok := upgraders[0]
	if !ok {
		t.Fatal("no upgrader registered for schema version 0")
	}
	if up.PriorSchema == nil {
		t.Fatal("the v0 upgrader has no PriorSchema; state cannot be decoded without it")
	}

	priorType := up.PriorSchema.Type().TerraformType(ctx).(tftypes.Object)
	priorRaw := tftypes.NewValue(priorType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "system_update"),
		"auto_download":     tftypes.NewValue(tftypes.Bool, true),
		"train":             tftypes.NewValue(tftypes.String, "TrueNAS-SCALE-Fangtooth"),
		"current_version":   tftypes.NewValue(tftypes.String, "25.10.4"),
		"available_status":  tftypes.NewValue(tftypes.String, "UNAVAILABLE"),
		"available_version": tftypes.NewValue(tftypes.String, ""),
		"timeouts":          tftypes.NewValue(priorType.AttributeTypes["timeouts"], nil),
	})

	newSchema := schemaOf(t, ctx, r)
	req := resource.UpgradeStateRequest{
		State: &tfsdk.State{Schema: *up.PriorSchema, Raw: priorRaw},
	}
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: newSchema.Schema, Raw: tftypes.NewValue(newSchema.Schema.Type().TerraformType(ctx), nil)},
	}
	up.StateUpgrader(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("UpgradeState: %v", resp.Diagnostics)
	}

	var got SystemUpdateResourceModel
	resp.State.Get(ctx, &got)

	// auto_download renamed to autocheck: same meaning, value carried over.
	if !got.Autocheck.ValueBool() {
		t.Error("auto_download = true did not carry over to autocheck")
	}
	// train has NO successor. A stored train name is not a valid profile, so
	// carrying it across would plan a profile middleware rejects.
	if got.Profile.ValueString() != "" {
		t.Errorf("profile = %q, want empty; a v0 train is not a valid profile", got.Profile.ValueString())
	}
	if got.Profile.IsNull() || got.Profile.IsUnknown() {
		t.Error("profile must be known-empty, not null/unknown, or the next plan shows a phantom diff")
	}
	// The v0 computed fields came from methods that never existed, so they
	// are not trusted; the refresh fills them.
	if got.Status.IsNull() || got.CurrentVersion.IsNull() || got.AvailableVersion.IsNull() {
		t.Error("computed fields must be known-empty after the upgrade")
	}
	if got.ID.ValueString() != systemUpdateSingletonID {
		t.Errorf("id = %q, want %q", got.ID.ValueString(), systemUpdateSingletonID)
	}
}

// A v0 state with auto_download unset must land on a concrete false rather
// than null, which would be an invalid value for a Computed attribute.
func TestSystemUpdateResource_UpgradeState_v0NullAutoDownload(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}
	up := r.UpgradeState(ctx)[0]

	priorType := up.PriorSchema.Type().TerraformType(ctx).(tftypes.Object)
	priorRaw := tftypes.NewValue(priorType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "system_update"),
		"auto_download":     tftypes.NewValue(tftypes.Bool, nil),
		"train":             tftypes.NewValue(tftypes.String, nil),
		"current_version":   tftypes.NewValue(tftypes.String, nil),
		"available_status":  tftypes.NewValue(tftypes.String, nil),
		"available_version": tftypes.NewValue(tftypes.String, nil),
		"timeouts":          tftypes.NewValue(priorType.AttributeTypes["timeouts"], nil),
	})

	newSchema := schemaOf(t, ctx, r)
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: newSchema.Schema, Raw: tftypes.NewValue(newSchema.Schema.Type().TerraformType(ctx), nil)},
	}
	up.StateUpgrader(ctx, resource.UpgradeStateRequest{
		State: &tfsdk.State{Schema: *up.PriorSchema, Raw: priorRaw},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("UpgradeState: %v", resp.Diagnostics)
	}

	var got SystemUpdateResourceModel
	resp.State.Get(ctx, &got)
	if got.Autocheck.IsNull() || got.Autocheck.IsUnknown() {
		t.Error("autocheck must be concrete after the upgrade")
	}
	if got.Autocheck.ValueBool() {
		t.Error("an unset auto_download must upgrade to false, the conservative value")
	}
}

// A prior state that cannot be decoded must produce diagnostics rather than
// writing a zero-valued model into state.
func TestSystemUpdateResource_UpgradeState_decodeFailure(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}

	bogus := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.BoolAttribute{Computed: true},
		},
	}
	bogusRaw := tftypes.NewValue(
		bogus.Type().TerraformType(ctx).(tftypes.Object),
		map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.Bool, true)},
	)

	sch := schemaOf(t, ctx, r)
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)},
	}
	r.UpgradeState(ctx)[0].StateUpgrader(ctx, resource.UpgradeStateRequest{
		State: &tfsdk.State{Schema: bogus, Raw: bogusRaw},
	}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("an undecodable prior state was silently upgraded")
	}
}
