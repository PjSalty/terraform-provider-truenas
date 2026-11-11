package resources_test

// Plan-level tests for the truenas_replication attributes that are read back
// when a configuration leaves them out. What is planned for them is decided
// by the plugin framework (defaults, computed marking, plan modifiers), not
// by the resource's CRUD methods, so these tests drive the real protocol-6
// PlanResourceChange path of the provider. No TrueNAS and no terraform binary
// are involved: a plan makes no API call.
//
// The proposed new state is what Terraform core would send: the configured
// value where there is one, otherwise the prior value of a computed
// attribute, element by element for a nested list.

import (
	"context"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/provider"
)

const replicationType = "truenas_replication"

type replicationPlanner struct {
	srv tfprotov6.ProviderServer
	typ tftypes.Object
}

func newReplicationPlanner(ctx context.Context, t *testing.T) replicationPlanner {
	t.Helper()
	srv := providerserver.NewProtocol6(provider.New("test")())()
	schemaResp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	sch, ok := schemaResp.ResourceSchemas[replicationType]
	if !ok {
		t.Fatalf("provider does not declare %s", replicationType)
	}
	typ, ok := sch.ValueType().(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is %T, want an object", sch.ValueType())
	}
	return replicationPlanner{srv: srv, typ: typ}
}

// object fills every attribute the map leaves out with a null of its type.
func (p replicationPlanner) object(vals map[string]tftypes.Value) tftypes.Value {
	full := make(map[string]tftypes.Value, len(p.typ.AttributeTypes))
	for name, at := range p.typ.AttributeTypes {
		full[name] = tftypes.NewValue(at, nil)
		if v, ok := vals[name]; ok {
			full[name] = v
		}
	}
	return tftypes.NewValue(p.typ, full)
}

// schedule builds a schedule list. Each entry gives its fields; a missing
// field is null, as in a configuration that leaves it out.
func (p replicationPlanner) schedule(entries ...map[string]tftypes.Value) tftypes.Value {
	listType, _ := p.typ.AttributeTypes["schedule"].(tftypes.List)
	entryType, _ := listType.ElementType.(tftypes.Object)
	vals := make([]tftypes.Value, 0, len(entries))
	for _, fields := range entries {
		full := make(map[string]tftypes.Value, len(entryType.AttributeTypes))
		for name, at := range entryType.AttributeTypes {
			full[name] = tftypes.NewValue(at, nil)
			if v, ok := fields[name]; ok {
				full[name] = v
			}
		}
		vals = append(vals, tftypes.NewValue(entryType, full))
	}
	return tftypes.NewValue(listType, vals)
}

func (p replicationPlanner) tasks(ids ...int64) tftypes.Value {
	vals := make([]tftypes.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, tftypes.NewValue(tftypes.Number, id))
	}
	return tftypes.NewValue(p.typ.AttributeTypes["periodic_snapshot_tasks"], vals)
}

// plan runs PlanResourceChange and returns the planned attributes. prior is
// nil for a create. The proposed state is the configuration laid over the
// prior state, with the given overrides for the nested lists.
func (p replicationPlanner) plan(ctx context.Context, t *testing.T, prior, config, proposedOver map[string]tftypes.Value) map[string]tftypes.Value {
	t.Helper()
	priorVal := tftypes.NewValue(p.typ, nil)
	proposed := map[string]tftypes.Value{}
	if prior != nil {
		priorVal = p.object(prior)
		maps.Copy(proposed, prior)
	}
	maps.Copy(proposed, config)
	maps.Copy(proposed, proposedOver)

	dv := func(v tftypes.Value) *tfprotov6.DynamicValue {
		out, err := tfprotov6.NewDynamicValue(p.typ, v)
		if err != nil {
			t.Fatalf("NewDynamicValue: %v", err)
		}
		return &out
	}
	resp, err := p.srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         replicationType,
		PriorState:       dv(priorVal),
		ProposedNewState: dv(p.object(proposed)),
		Config:           dv(p.object(config)),
	})
	if err != nil {
		t.Fatalf("PlanResourceChange: %v", err)
	}
	for _, d := range resp.Diagnostics {
		t.Fatalf("unexpected diagnostic: [%s] %s: %s", d.Severity, d.Summary, d.Detail)
	}
	planned, err := resp.PlannedState.Unmarshal(p.typ)
	if err != nil {
		t.Fatalf("unmarshal planned state: %v", err)
	}
	var out map[string]tftypes.Value
	if err := planned.As(&out); err != nil {
		t.Fatalf("planned.As(object): %v", err)
	}
	return out
}

func tfString(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
func tfBool(b bool) tftypes.Value     { return tftypes.NewValue(tftypes.Bool, b) }

// replicationConfig is a manual LOCAL push task with every defaulted
// attribute written out, so only the attributes under test can differ.
func replicationConfig(name string) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"name": tfString(name), "direction": tfString("PUSH"), "transport": tfString("LOCAL"),
		"source_datasets": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tfString("tank/src")}),
		"target_dataset":  tfString("tank/dst"), "recursive": tfBool(false), "auto": tfBool(true),
		"enabled": tfBool(true), "retention_policy": tfString("NONE"),
	}
}

func fullSchedule(hour string) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"minute": tfString("00"), "hour": tfString(hour), "dom": tfString("*"), "month": tfString("*"),
		"dow": tfString("*"), "begin": tfString("00:00"), "end": tfString("23:59"),
	}
}

// scheduleEntries decodes a planned schedule list into its entries.
func scheduleEntries(t *testing.T, v tftypes.Value) []map[string]tftypes.Value {
	t.Helper()
	var elems []tftypes.Value
	if err := v.As(&elems); err != nil {
		t.Fatalf("schedule.As(elements): %v", err)
	}
	out := make([]map[string]tftypes.Value, 0, len(elems))
	for _, e := range elems {
		var fields map[string]tftypes.Value
		if err := e.As(&fields); err != nil {
			t.Fatalf("schedule entry As(object): %v", err)
		}
		out = append(out, fields)
	}
	return out
}

func stringOf(t *testing.T, v tftypes.Value) string {
	t.Helper()
	var s string
	if err := v.As(&s); err != nil {
		t.Fatalf("As(string) on %s: %v", v, err)
	}
	return s
}

// The five attributes this resource reads back when a configuration leaves
// them out.
func readBackAttributes() []string {
	return []string{
		"schedule", "restrict_schedule", "only_matching_schedule", "hold_pending_snapshots", "periodic_snapshot_tasks",
		"readonly", "allow_from_scratch", "retries", "exclude", "properties", "properties_exclude", "properties_override",
		"replicate", "compression", "speed_limit", "large_block", "embed", "compressed", "sudo", "logging_level",
		"name_regex", "lifetimes",
	}
}

// options gives every option attribute a value that differs from the TrueNAS
// default, as a task set up outside Terraform would have.
func (p replicationPlanner) options() map[string]tftypes.Value {
	num := func(n int64) tftypes.Value { return tftypes.NewValue(tftypes.Number, n) }
	lifetimes, _ := p.typ.AttributeTypes["lifetimes"].(tftypes.List)
	lifetime, _ := lifetimes.ElementType.(tftypes.Object)
	cron, _ := lifetime.AttributeTypes["schedule"].(tftypes.Object)
	return map[string]tftypes.Value{
		"readonly": tfString("REQUIRE"), "allow_from_scratch": tfBool(true), "retries": num(9),
		"exclude":            tftypes.NewValue(p.typ.AttributeTypes["exclude"], []tftypes.Value{tfString("tank/src/tmp")}),
		"properties":         tfBool(false),
		"properties_exclude": tftypes.NewValue(p.typ.AttributeTypes["properties_exclude"], []tftypes.Value{tfString("mountpoint")}),
		"properties_override": tftypes.NewValue(p.typ.AttributeTypes["properties_override"],
			map[string]tftypes.Value{"atime": tfString("off")}),
		"replicate": tfBool(true), "compression": tfString("LZ4"), "speed_limit": num(4096),
		"large_block": tfBool(false), "embed": tfBool(true), "compressed": tfBool(false), "sudo": tfBool(true),
		"logging_level": tfString("DEBUG"), "name_regex": tfString("auto-.*"),
		"lifetimes": tftypes.NewValue(lifetimes, []tftypes.Value{tftypes.NewValue(lifetime, map[string]tftypes.Value{
			"schedule": tftypes.NewValue(cron, map[string]tftypes.Value{
				"minute": tfString("0"), "hour": tfString("0"), "dom": tfString("*"), "month": tfString("*"), "dow": tfString("7"),
			}),
			"lifetime_value": num(4), "lifetime_unit": tfString("WEEK"),
		})}),
	}
}

// A task that has these values in TrueNAS, and a configuration written before
// the attributes existed, must plan no change to them: an upgrade of the
// provider does not touch a schedule or a binding that was set elsewhere.
func TestReplicationPlan_UnsetAttributesKeepWhatTrueNASHas(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	prior := replicationConfig("rep")
	prior["id"] = tfString("7")
	prior["schedule"] = p.schedule(fullSchedule("*/6"))
	prior["restrict_schedule"] = p.schedule()
	prior["only_matching_schedule"] = tfBool(true)
	prior["hold_pending_snapshots"] = tfBool(true)
	prior["periodic_snapshot_tasks"] = p.tasks(4, 9)
	maps.Copy(prior, p.options())

	// The rename makes the plan differ from the state. The framework then
	// marks every computed attribute the configuration leaves out as
	// unknown, and only the plan modifiers bring the prior values back.
	planned := p.plan(ctx, t, prior, replicationConfig("renamed"), nil)

	for _, name := range readBackAttributes() {
		if !planned[name].Equal(prior[name]) {
			t.Errorf("%s is planned as %s, want the prior value %s", name, planned[name], prior[name])
		}
	}
}

// State written by an older provider version has no value for the new
// attributes until a refresh. They must then be unknown, not null: the apply
// reads the real values back, and a planned null would not match them.
func TestReplicationPlan_StateFromAnOlderVersionPlansUnknown(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	prior := replicationConfig("rep")
	prior["id"] = tfString("7")

	planned := p.plan(ctx, t, prior, replicationConfig("renamed"), nil)

	for _, name := range readBackAttributes() {
		if planned[name].IsKnown() {
			t.Errorf("%s is planned as %s, want unknown", name, planned[name])
		}
	}
}

func TestReplicationPlan_CreateLeavesUnsetAttributesUnknown(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)

	planned := p.plan(ctx, t, nil, replicationConfig("rep"), nil)

	for _, name := range readBackAttributes() {
		if planned[name].IsKnown() {
			t.Errorf("%s is planned as %s, want unknown so TrueNAS applies its default", name, planned[name])
		}
	}
}

// An empty list is a decision, not an absence: it plans the removal.
func TestReplicationPlan_EmptyListsClear(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	prior := replicationConfig("rep")
	prior["id"] = tfString("7")
	prior["schedule"] = p.schedule(fullSchedule("*/6"))
	prior["restrict_schedule"] = p.schedule(fullSchedule("*/2"))
	prior["only_matching_schedule"] = tfBool(false)
	prior["hold_pending_snapshots"] = tfBool(false)
	prior["periodic_snapshot_tasks"] = p.tasks(4)

	config := replicationConfig("rep")
	config["schedule"] = p.schedule()
	config["restrict_schedule"] = p.schedule()
	config["periodic_snapshot_tasks"] = p.tasks()

	planned := p.plan(ctx, t, prior, config, nil)

	for _, name := range []string{"schedule", "restrict_schedule"} {
		if entries := scheduleEntries(t, planned[name]); len(entries) != 0 {
			t.Errorf("%s is planned with %d entries, want none", name, len(entries))
		}
	}
	if !planned["periodic_snapshot_tasks"].Equal(p.tasks()) {
		t.Errorf("periodic_snapshot_tasks is planned as %s, want an empty set", planned["periodic_snapshot_tasks"])
	}
}

// Changing one field of a schedule keeps the others. Without the modifier on
// the nested fields they would be planned unknown on every change.
func TestReplicationPlan_ScheduleEntryKeepsItsUnsetFields(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	prior := replicationConfig("rep")
	prior["id"] = tfString("7")
	prior["schedule"] = p.schedule(fullSchedule("*/6"))
	prior["restrict_schedule"] = p.schedule()
	prior["only_matching_schedule"] = tfBool(false)
	prior["hold_pending_snapshots"] = tfBool(false)
	prior["periodic_snapshot_tasks"] = p.tasks()

	config := replicationConfig("rep")
	config["schedule"] = p.schedule(map[string]tftypes.Value{"hour": tfString("2")})
	// Terraform core fills the entry's computed fields from the prior entry
	// at the same index.
	proposed := fullSchedule("2")

	planned := p.plan(ctx, t, prior, config, map[string]tftypes.Value{"schedule": p.schedule(proposed)})

	entries := scheduleEntries(t, planned["schedule"])
	if len(entries) != 1 {
		t.Fatalf("schedule is planned with %d entries, want 1", len(entries))
	}
	for name, want := range map[string]string{"hour": "2", "minute": "00", "dow": "*", "begin": "00:00", "end": "23:59"} {
		if !entries[0][name].IsKnown() {
			t.Errorf("schedule.%s is planned unknown, want %q", name, want)
			continue
		}
		if got := stringOf(t, entries[0][name]); got != want {
			t.Errorf("schedule.%s is planned as %q, want %q", name, got, want)
		}
	}
}

// A new schedule has no prior entry to take values from. The fields the
// configuration leaves out must be unknown, so that the defaults TrueNAS
// fills in are accepted after the apply. A planned null would be reported as
// an inconsistent result.
func TestReplicationPlan_NewScheduleEntryLeavesUnsetFieldsUnknown(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	prior := replicationConfig("rep")
	prior["id"] = tfString("7")
	prior["schedule"] = p.schedule()
	prior["restrict_schedule"] = p.schedule()
	prior["only_matching_schedule"] = tfBool(false)
	prior["hold_pending_snapshots"] = tfBool(false)
	prior["periodic_snapshot_tasks"] = p.tasks()

	config := replicationConfig("rep")
	config["schedule"] = p.schedule(map[string]tftypes.Value{"hour": tfString("*/6")})

	planned := p.plan(ctx, t, prior, config, nil)

	entries := scheduleEntries(t, planned["schedule"])
	if len(entries) != 1 {
		t.Fatalf("schedule is planned with %d entries, want 1", len(entries))
	}
	if got := stringOf(t, entries[0]["hour"]); got != "*/6" {
		t.Errorf("schedule.hour is planned as %q, want */6", got)
	}
	for _, name := range []string{"minute", "dom", "month", "dow", "begin", "end"} {
		if entries[0][name].IsKnown() {
			t.Errorf("schedule.%s is planned as %s, want unknown", name, entries[0][name])
		}
	}
}

func TestReplicationConfig_ScheduleIsValidated(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	cases := map[string]tftypes.Value{
		"two entries":             p.schedule(fullSchedule("1"), fullSchedule("2")),
		"begin not written HH:MM": p.schedule(map[string]tftypes.Value{"begin": tfString("9:00")}),
		"end past the day":        p.schedule(map[string]tftypes.Value{"end": tfString("24:00")}),
	}
	for name, schedule := range cases {
		t.Run(name, func(t *testing.T) {
			config := replicationConfig("rep")
			config["schedule"] = schedule
			dv, err := tfprotov6.NewDynamicValue(p.typ, p.object(config))
			if err != nil {
				t.Fatalf("NewDynamicValue: %v", err)
			}
			resp, err := p.srv.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: replicationType, Config: &dv,
			})
			if err != nil {
				t.Fatalf("ValidateResourceConfig: %v", err)
			}
			if len(resp.Diagnostics) == 0 {
				t.Error("the configuration was accepted")
			}
		})
	}

	t.Run("one well-formed entry", func(t *testing.T) {
		config := replicationConfig("rep")
		config["schedule"] = p.schedule(map[string]tftypes.Value{"hour": tfString("*/6"), "begin": tfString("09:00"), "end": tfString("17:30")})
		dv, err := tfprotov6.NewDynamicValue(p.typ, p.object(config))
		if err != nil {
			t.Fatalf("NewDynamicValue: %v", err)
		}
		resp, err := p.srv.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
			TypeName: replicationType, Config: &dv,
		})
		if err != nil {
			t.Fatalf("ValidateResourceConfig: %v", err)
		}
		for _, d := range resp.Diagnostics {
			t.Errorf("unexpected diagnostic: [%s] %s: %s", d.Severity, d.Summary, d.Detail)
		}
	})
}

// The empty value of a nullable option is a decision too: it plans the
// change from what TrueNAS holds, and is not taken for "left out".
func TestReplicationPlan_EmptyOptionValuesClear(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	prior := replicationConfig("rep")
	prior["id"] = tfString("7")
	prior["schedule"] = p.schedule()
	prior["restrict_schedule"] = p.schedule()
	prior["only_matching_schedule"] = tfBool(false)
	prior["hold_pending_snapshots"] = tfBool(false)
	prior["periodic_snapshot_tasks"] = p.tasks()
	maps.Copy(prior, p.options())

	config := replicationConfig("rep")
	empty := map[string]tftypes.Value{
		"compression": tfString(""), "logging_level": tfString(""), "name_regex": tfString(""),
		"speed_limit":         tftypes.NewValue(tftypes.Number, int64(0)),
		"exclude":             tftypes.NewValue(p.typ.AttributeTypes["exclude"], []tftypes.Value{}),
		"properties_override": tftypes.NewValue(p.typ.AttributeTypes["properties_override"], map[string]tftypes.Value{}),
		"lifetimes":           tftypes.NewValue(p.typ.AttributeTypes["lifetimes"], []tftypes.Value{}),
	}
	maps.Copy(config, empty)

	planned := p.plan(ctx, t, prior, config, nil)

	for name, want := range empty {
		if !planned[name].Equal(want) {
			t.Errorf("%s is planned as %s, want %s", name, planned[name], want)
		}
	}
	// What the configuration does not mention stays as it is.
	for _, name := range []string{"readonly", "retries", "properties_exclude", "sudo"} {
		if !planned[name].Equal(prior[name]) {
			t.Errorf("%s is planned as %s, want the prior value %s", name, planned[name], prior[name])
		}
	}
}

func TestReplicationConfig_OptionsAreValidated(t *testing.T) {
	ctx := context.Background()
	p := newReplicationPlanner(ctx, t)
	cases := map[string]map[string]tftypes.Value{
		"readonly outside its set":      {"readonly": tfString("OFF")},
		"compression outside its set":   {"compression": tfString("ZSTD")},
		"logging_level outside its set": {"logging_level": tfString("TRACE")},
		"retries below one":             {"retries": tftypes.NewValue(tftypes.Number, int64(0))},
		"negative speed_limit":          {"speed_limit": tftypes.NewValue(tftypes.Number, int64(-1))},
	}
	for name, over := range cases {
		t.Run(name, func(t *testing.T) {
			config := replicationConfig("rep")
			maps.Copy(config, over)
			dv, err := tfprotov6.NewDynamicValue(p.typ, p.object(config))
			if err != nil {
				t.Fatalf("NewDynamicValue: %v", err)
			}
			resp, err := p.srv.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: replicationType, Config: &dv,
			})
			if err != nil {
				t.Fatalf("ValidateResourceConfig: %v", err)
			}
			if len(resp.Diagnostics) == 0 {
				t.Error("the configuration was accepted")
			}
		})
	}
}
