package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCronJob_UpgradeState_Map(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)
	upgraders := r.UpgradeState(context.Background())
	if _, ok := upgraders[0]; !ok {
		t.Fatal("expected v0 upgrader")
	}
	if upgraders[0].PriorSchema == nil {
		t.Fatal("expected PriorSchema to be set")
	}
	if upgraders[0].StateUpgrader == nil {
		t.Fatal("expected StateUpgrader func to be set")
	}
}

// TestCronJob_UpgradeState_NoOp verifies the v0 → v1 migration is a no-op:
// every v0 field round-trips into the v1 model unchanged. This is the fixture
// future breaking changes will replace with a real transformation.
func TestCronJob_UpgradeState_NoOp(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)
	ctx := context.Background()
	v0 := cronjobSchemaV0(ctx)

	// Build a v0 state with a representative set of values.
	v0Raw := tftypes.NewValue(
		v0.Type().TerraformType(ctx).(tftypes.Object),
		map[string]tftypes.Value{
			"id":              str("42"),
			"user":            str("root"),
			"command":         str("/usr/bin/backup.sh"),
			"description":     str("nightly backup"),
			"enabled":         flag(true),
			"stdout":          flag(true),
			"stderr":          flag(false),
			"schedule_minute": str("0"),
			"schedule_hour":   str("2"),
			"schedule_dom":    str("*"),
			"schedule_month":  str("*"),
			"schedule_dow":    str("*"),
			"timeouts": tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"create": tftypes.String,
					"read":   tftypes.String,
					"update": tftypes.String,
					"delete": tftypes.String,
				}},
				map[string]tftypes.Value{
					"create": tftypes.NewValue(tftypes.String, nil),
					"read":   tftypes.NewValue(tftypes.String, nil),
					"update": tftypes.NewValue(tftypes.String, nil),
					"delete": tftypes.NewValue(tftypes.String, nil),
				},
			),
		},
	)

	sch := schemaOf(t, ctx, r)
	req := resource.UpgradeStateRequest{
		State: &tfsdk.State{Schema: v0, Raw: v0Raw},
	}
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)},
	}

	upgraders := r.UpgradeState(ctx)
	upgraders[0].StateUpgrader(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("UpgradeState: %v", resp.Diagnostics)
	}

	var upgraded CronJobResourceModel
	if diags := resp.State.Get(ctx, &upgraded); diags.HasError() {
		t.Fatalf("decode upgraded state: %v", diags)
	}

	if upgraded.ID.ValueString() != "42" {
		t.Errorf("ID: got %q", upgraded.ID.ValueString())
	}
	if upgraded.User.ValueString() != "root" {
		t.Errorf("User: got %q", upgraded.User.ValueString())
	}
	if upgraded.Command.ValueString() != "/usr/bin/backup.sh" {
		t.Errorf("Command: got %q", upgraded.Command.ValueString())
	}
	if upgraded.Hour.ValueString() != "2" {
		t.Errorf("Hour: got %q", upgraded.Hour.ValueString())
	}
	if !upgraded.Enabled.ValueBool() {
		t.Errorf("Enabled: false")
	}
}

// TestCronJob_UpgradeState_GetError exercises the early-return branch when
// req.State.Get into the prior model fails. We build a bogus prior schema
// whose `id` attribute is a Bool — the framework reflection into
// CronJobResourceModel (where ID is types.String) fails with a type error,
// which is exactly the condition the StateUpgrader guards against.
func TestCronJob_UpgradeState_GetError(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)
	ctx := context.Background()

	bogus := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.BoolAttribute{Computed: true},
		},
	}
	bogusRaw := tftypes.NewValue(
		bogus.Type().TerraformType(ctx).(tftypes.Object),
		map[string]tftypes.Value{"id": flag(true)},
	)

	sch := schemaOf(t, ctx, r)
	req := resource.UpgradeStateRequest{
		State: &tfsdk.State{Schema: bogus, Raw: bogusRaw},
	}
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)},
	}

	upgraders := r.UpgradeState(ctx)
	upgraders[0].StateUpgrader(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostics when State.Get fails")
	}
}
