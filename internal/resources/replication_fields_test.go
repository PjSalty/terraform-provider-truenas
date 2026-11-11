package resources

import (
	"context"
	"maps"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// replBody is a ReplicationEntry as TrueNAS returns it for a manual LOCAL
// push task, with the given keys replacing the defaults.
func replBody(over map[string]interface{}) map[string]interface{} {
	b := map[string]interface{}{
		"id": 7, "name": "rep", "direction": "PUSH", "transport": "LOCAL",
		"source_datasets": []interface{}{"tank/src"}, "target_dataset": "tank/dst",
		"recursive": false, "auto": false, "enabled": true, "retention_policy": "NONE",
		"ssh_credentials": nil, "periodic_snapshot_tasks": []interface{}{},
		"naming_schema": []interface{}{}, "also_include_naming_schema": []interface{}{},
		"schedule": nil, "restrict_schedule": nil,
		"only_matching_schedule": false, "hold_pending_snapshots": false,
		"readonly": "SET", "allow_from_scratch": false, "retries": 5, "exclude": []interface{}{},
		"properties": true, "properties_exclude": []interface{}{}, "properties_override": map[string]interface{}{},
		"replicate": false, "compression": nil, "speed_limit": nil, "large_block": true, "embed": false,
		"compressed": true, "sudo": false, "logging_level": nil, "name_regex": nil, "lifetimes": []interface{}{},
		"encryption": false, "encryption_inherit": nil, "encryption_key": nil, "encryption_key_format": nil,
		"encryption_key_location": nil, "netcat_active_side": nil, "netcat_active_side_listen_address": nil,
		"netcat_active_side_port_min": nil, "netcat_active_side_port_max": nil, "netcat_passive_side_connect_address": nil,
	}
	maps.Copy(b, over)
	return b
}

// replCaptureClient answers every replication call with body and records
// the request map of the last create or update.
func replCaptureClient(ctx context.Context, t *testing.T, body map[string]interface{}, sent *map[string]interface{}) *wsclient.Client {
	t.Helper()
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "replication.create":
			if m, ok := params[0].(map[string]interface{}); ok {
				*sent = m
			}
		case "replication.update":
			if m, ok := params[1].(map[string]interface{}); ok {
				*sent = m
			}
		}
		return body, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c
}

func replVals(over map[string]tftypes.Value) map[string]tftypes.Value {
	vals := map[string]tftypes.Value{
		"id": str("7"), "name": str("rep"), "direction": str("PUSH"), "transport": str("LOCAL"),
		"source_datasets": strList("tank/src"),
		"target_dataset":  str("tank/dst"), "recursive": flag(false), "auto": flag(false), "enabled": flag(true),
		"retention_policy": str("NONE"),
	}
	maps.Copy(vals, over)
	return vals
}

func replScheduleType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"minute": tftypes.String, "hour": tftypes.String, "dom": tftypes.String, "month": tftypes.String,
		"dow": tftypes.String, "begin": tftypes.String, "end": tftypes.String,
	}}
}

// replSchedule is a planned schedule list. Each entry names the fields that
// are known; the others are unknown, as on a new schedule.
func replSchedule(entries ...map[string]string) tftypes.Value {
	typ := replScheduleType()
	vals := make([]tftypes.Value, 0, len(entries))
	for _, known := range entries {
		fields := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for name := range typ.AttributeTypes {
			fields[name] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			if v, ok := known[name]; ok {
				fields[name] = str(v)
			}
		}
		vals = append(vals, tftypes.NewValue(typ, fields))
	}
	return tftypes.NewValue(tftypes.List{ElementType: typ}, vals)
}

func replTaskSet(ids ...int64) tftypes.Value {
	vals := make([]tftypes.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, num(id))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, vals)
}

// replNewFields are the request keys this resource leaves out until their
// value is known.
var replNewFields = []string{
	"schedule", "restrict_schedule", "only_matching_schedule", "hold_pending_snapshots", "periodic_snapshot_tasks",
	"readonly", "allow_from_scratch", "retries", "exclude", "properties", "properties_exclude", "properties_override",
	"replicate", "compression", "speed_limit", "large_block", "embed", "compressed", "sudo", "logging_level",
	"name_regex", "lifetimes",
	"encryption", "encryption_inherit", "encryption_key", "encryption_key_format", "encryption_key_location",
	"netcat_active_side", "netcat_active_side_listen_address", "netcat_active_side_port_min",
	"netcat_active_side_port_max", "netcat_passive_side_connect_address",
}

func TestReplicationResource_Create_UnknownFieldsAreLeftOut(t *testing.T) {
	ctx := context.Background()
	var sent map[string]interface{}
	r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
	sch := schemaOf(t, ctx, r)
	unknownList := tftypes.NewValue(tftypes.List{ElementType: replScheduleType()}, tftypes.UnknownValue)
	plan := planFromValues(t, ctx, sch, replVals(map[string]tftypes.Value{
		"schedule":                unknownList,
		"restrict_schedule":       unknownList,
		"only_matching_schedule":  unknownOf(tftypes.Bool),
		"hold_pending_snapshots":  unknownOf(tftypes.Bool),
		"periodic_snapshot_tasks": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, tftypes.UnknownValue),
	}))
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if sent == nil {
		t.Fatal("replication.create was not called")
	}
	for _, key := range replNewFields {
		if v, ok := sent[key]; ok {
			t.Errorf("%s = %v was sent while unknown; TrueNAS must apply its own default", key, v)
		}
	}

	// The response then decides the state: no schedule is an empty list and
	// no bound task is an empty set, never null.
	var state ReplicationResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("State.Get: %v", resp.Diagnostics)
	}
	if state.Schedule.IsNull() || len(state.Schedule.Elements()) != 0 {
		t.Errorf("schedule = %v, want an empty list", state.Schedule)
	}
	if state.RestrictSchedule.IsNull() || len(state.RestrictSchedule.Elements()) != 0 {
		t.Errorf("restrict_schedule = %v, want an empty list", state.RestrictSchedule)
	}
	if state.PeriodicSnapshotTasks.IsNull() || len(state.PeriodicSnapshotTasks.Elements()) != 0 {
		t.Errorf("periodic_snapshot_tasks = %v, want an empty set", state.PeriodicSnapshotTasks)
	}
	if state.HoldPendingSnapshots.ValueBool() || state.OnlyMatchingSchedule.ValueBool() {
		t.Errorf("flags = %v %v, want false from the response", state.HoldPendingSnapshots, state.OnlyMatchingSchedule)
	}
}

func TestReplicationResource_Create_SendsConfiguredFields(t *testing.T) {
	ctx := context.Background()
	var sent map[string]interface{}
	body := replBody(map[string]interface{}{
		"auto": true, "hold_pending_snapshots": true, "only_matching_schedule": true,
		"schedule": map[string]interface{}{
			"minute": "00", "hour": "*/6", "dom": "*", "month": "*", "dow": "*", "begin": "00:00", "end": "23:59",
		},
		// TrueNAS returns the bound tasks expanded, not as the IDs it was sent.
		"periodic_snapshot_tasks": []interface{}{
			map[string]interface{}{"id": 4, "dataset": "tank/src"},
			map[string]interface{}{"id": 9, "dataset": "tank/src"},
		},
	})
	r := &ReplicationResource{client: replCaptureClient(ctx, t, body, &sent)}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, replVals(map[string]tftypes.Value{
		"auto":                    flag(true),
		"schedule":                replSchedule(map[string]string{"hour": "*/6"}),
		"restrict_schedule":       replSchedule(),
		"only_matching_schedule":  flag(true),
		"hold_pending_snapshots":  flag(true),
		"periodic_snapshot_tasks": replTaskSet(9, 4),
	}))
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	// Only the known field of the entry is sent; TrueNAS fills the rest.
	if want := map[string]interface{}{"hour": "*/6"}; !reflect.DeepEqual(sent["schedule"], want) {
		t.Errorf("schedule = %v, want %v", sent["schedule"], want)
	}
	if v, ok := sent["restrict_schedule"]; !ok || v != nil {
		t.Errorf("restrict_schedule = %v (sent %v), want an explicit null for []", v, ok)
	}
	if sent["only_matching_schedule"] != true || sent["hold_pending_snapshots"] != true {
		t.Errorf("flags = %v %v, want true true", sent["only_matching_schedule"], sent["hold_pending_snapshots"])
	}
	if want := []interface{}{float64(4), float64(9)}; !reflect.DeepEqual(sent["periodic_snapshot_tasks"], want) {
		t.Errorf("periodic_snapshot_tasks = %v, want %v (sorted IDs)", sent["periodic_snapshot_tasks"], want)
	}

	var hour, end types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("schedule").AtListIndex(0).AtName("hour"), &hour)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("schedule").AtListIndex(0).AtName("end"), &end)...)
	var tasks []int64
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("periodic_snapshot_tasks"), &tasks)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("State.GetAttribute: %v", resp.Diagnostics)
	}
	if hour.ValueString() != "*/6" || end.ValueString() != "23:59" {
		t.Errorf("schedule in state = hour %v end %v, want */6 and the default 23:59 read back", hour, end)
	}
	if !reflect.DeepEqual(tasks, []int64{4, 9}) {
		t.Errorf("periodic_snapshot_tasks in state = %v, want [4 9] from the expanded entries", tasks)
	}
}

// An update has to be able to remove what a configuration no longer has: an
// empty list reaches TrueNAS as [] or null, and never as a missing key.
func TestReplicationResource_Update_ClearsAndSets(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		planned map[string]tftypes.Value
		want    map[string]interface{}
	}{
		{
			name: "empty lists clear",
			planned: map[string]tftypes.Value{
				"schedule": replSchedule(), "restrict_schedule": replSchedule(),
				"periodic_snapshot_tasks": replTaskSet(),
			},
			want: map[string]interface{}{
				"schedule": nil, "restrict_schedule": nil,
				"periodic_snapshot_tasks":    []interface{}{},
				"naming_schema":              []interface{}{},
				"also_include_naming_schema": []interface{}{},
			},
		},
		{
			name: "values are sent in full",
			planned: map[string]tftypes.Value{
				"auto": flag(true),
				"schedule": replSchedule(map[string]string{
					"minute": "30", "hour": "2", "dom": "*", "month": "*", "dow": "1", "begin": "01:00", "end": "05:00",
				}),
				"restrict_schedule":          replSchedule(map[string]string{"hour": "*/2"}),
				"only_matching_schedule":     flag(true),
				"hold_pending_snapshots":     flag(false),
				"periodic_snapshot_tasks":    replTaskSet(12),
				"also_include_naming_schema": strList("auto-%Y-%m-%d_%H-%M", "manual-%Y"),
			},
			want: map[string]interface{}{
				"schedule": map[string]interface{}{
					"minute": "30", "hour": "2", "dom": "*", "month": "*", "dow": "1", "begin": "01:00", "end": "05:00",
				},
				"restrict_schedule":          map[string]interface{}{"hour": "*/2"},
				"only_matching_schedule":     true,
				"hold_pending_snapshots":     false,
				"periodic_snapshot_tasks":    []interface{}{float64(12)},
				"naming_schema":              []interface{}{},
				"also_include_naming_schema": []interface{}{"auto-%Y-%m-%d_%H-%M", "manual-%Y"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sent map[string]interface{}
			r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
			sch := schemaOf(t, ctx, r)
			state := stateFromValues(t, ctx, sch, replVals(nil))
			resp := &resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: planFromValues(t, ctx, sch, replVals(tc.planned))}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Update: %v", resp.Diagnostics)
			}
			if sent == nil {
				t.Fatal("replication.update was not called")
			}
			for key, want := range tc.want {
				got, ok := sent[key]
				if !ok {
					t.Errorf("%s was left out of the request, want %v", key, want)
					continue
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s = %#v, want %#v", key, got, want)
				}
			}
		})
	}
}

// A null or unknown value is not a decision to clear: the key stays out and
// TrueNAS keeps what it has.
func TestReplicationResource_Update_NullFieldsAreLeftOut(t *testing.T) {
	ctx := context.Background()
	var sent map[string]interface{}
	r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
	sch := schemaOf(t, ctx, r)
	state := stateFromValues(t, ctx, sch, replVals(nil))
	resp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: planFromValues(t, ctx, sch, replVals(nil))}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if sent == nil {
		t.Fatal("replication.update was not called")
	}
	for _, key := range replNewFields {
		if v, ok := sent[key]; ok {
			t.Errorf("%s = %v was sent for a null plan value", key, v)
		}
	}
}

// A schedule entry that is unknown as a whole cannot be sent. The apply must
// stop with a diagnostic before it reaches TrueNAS, not send null and remove
// the schedule.
func TestReplicationResource_UnknownScheduleEntryStopsBeforeTheCall(t *testing.T) {
	ctx := context.Background()
	typ := replScheduleType()
	unknownEntry := tftypes.NewValue(tftypes.List{ElementType: typ}, []tftypes.Value{
		tftypes.NewValue(typ, tftypes.UnknownValue),
	})

	t.Run("create", func(t *testing.T) {
		var sent map[string]interface{}
		r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
		sch := schemaOf(t, ctx, r)
		plan := planFromValues(t, ctx, sch, replVals(map[string]tftypes.Value{"schedule": unknownEntry}))
		resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Create accepted an unknown schedule entry")
		}
		if sent != nil {
			t.Errorf("replication.create was called with %v", sent)
		}
	})

	t.Run("update", func(t *testing.T) {
		var sent map[string]interface{}
		r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
		sch := schemaOf(t, ctx, r)
		state := stateFromValues(t, ctx, sch, replVals(nil))
		plan := planFromValues(t, ctx, sch, replVals(map[string]tftypes.Value{"restrict_schedule": unknownEntry}))
		resp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Update accepted an unknown restrict_schedule entry")
		}
		if sent != nil {
			t.Errorf("replication.update was called with %v", sent)
		}
	})
}

func TestReplicationResource_Read_MapsSchedulesAndTasks(t *testing.T) {
	ctx := context.Background()
	body := replBody(map[string]interface{}{
		"auto": true, "hold_pending_snapshots": true,
		"restrict_schedule": map[string]interface{}{
			"minute": "00", "hour": "*/2", "dom": "*", "month": "*", "dow": "*", "begin": "08:00", "end": "20:00",
		},
		"periodic_snapshot_tasks": []interface{}{map[string]interface{}{"id": 31, "dataset": "tank/src"}},
	})
	var sent map[string]interface{}
	r := &ReplicationResource{client: replCaptureClient(ctx, t, body, &sent)}
	sch := schemaOf(t, ctx, r)
	state := stateFromValues(t, ctx, sch, replVals(nil))
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var got ReplicationResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	var restrict []replicationScheduleModel
	resp.Diagnostics.Append(got.RestrictSchedule.ElementsAs(ctx, &restrict, false)...)
	var tasks []int64
	resp.Diagnostics.Append(got.PeriodicSnapshotTasks.ElementsAs(ctx, &tasks, false)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("decode state: %v", resp.Diagnostics)
	}
	if len(restrict) != 1 || restrict[0].Hour.ValueString() != "*/2" ||
		restrict[0].Begin.ValueString() != "08:00" || restrict[0].End.ValueString() != "20:00" {
		t.Errorf("restrict_schedule = %+v, want one entry with hour */2 from 08:00 to 20:00", restrict)
	}
	if len(got.Schedule.Elements()) != 0 {
		t.Errorf("schedule = %v, want an empty list for a null", got.Schedule)
	}
	if !reflect.DeepEqual(tasks, []int64{31}) {
		t.Errorf("periodic_snapshot_tasks = %v, want [31]", tasks)
	}
	if !got.HoldPendingSnapshots.ValueBool() || got.OnlyMatchingSchedule.ValueBool() {
		t.Errorf("flags = hold %v only_matching %v, want true and false", got.HoldPendingSnapshots, got.OnlyMatchingSchedule)
	}
}

// begin and end only take the HH:MM form TrueNAS returns, so the planned
// value and the value read back cannot differ in spelling.
func TestReplicationSchedule_TimeOfDay(t *testing.T) {
	for _, ok := range []string{"00:00", "09:30", "19:05", "23:59"} {
		if !timeOfDay.MatchString(ok) {
			t.Errorf("%q was rejected", ok)
		}
	}
	for _, bad := range []string{"9:30", "24:00", "12:60", "12:5", "1230", "12:30:00", ""} {
		if timeOfDay.MatchString(bad) {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func replStringSet(items ...string) tftypes.Value {
	vals := make([]tftypes.Value, 0, len(items))
	for _, s := range items {
		vals = append(vals, str(s))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, vals)
}

func replStringMap(items map[string]string) tftypes.Value {
	vals := make(map[string]tftypes.Value, len(items))
	for k, v := range items {
		vals[k] = str(v)
	}
	return tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, vals)
}

// replLifetimes is a planned lifetimes list of (cron hour, value, unit).
func replLifetimes(entries ...[3]string) tftypes.Value {
	cron := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"minute": tftypes.String, "hour": tftypes.String, "dom": tftypes.String, "month": tftypes.String, "dow": tftypes.String,
	}}
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"schedule": cron, "lifetime_value": tftypes.Number, "lifetime_unit": tftypes.String,
	}}
	vals := make([]tftypes.Value, 0, len(entries))
	for _, e := range entries {
		value, _ := strconv.ParseInt(e[1], 10, 64)
		vals = append(vals, tftypes.NewValue(typ, map[string]tftypes.Value{
			"schedule": tftypes.NewValue(cron, map[string]tftypes.Value{
				"minute": str("0"), "hour": str(e[0]), "dom": str("*"), "month": str("*"), "dow": str("*"),
			}),
			"lifetime_value": num(value), "lifetime_unit": str(e[2]),
		}))
	}
	return tftypes.NewValue(tftypes.List{ElementType: typ}, vals)
}

// Create and update build their options the same way, so each case runs
// through both and compares the request keys.
func TestReplicationResource_Options_AreSent(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		planned map[string]tftypes.Value
		want    map[string]interface{}
	}{
		{
			name: "configured values",
			planned: map[string]tftypes.Value{
				"readonly": str("IGNORE"), "allow_from_scratch": flag(true), "retries": num(3),
				"exclude": replStringSet("tank/src/tmp", "tank/src/cache"), "properties": flag(false),
				"properties_exclude":  replStringSet("mountpoint"),
				"properties_override": replStringMap(map[string]string{"atime": "off"}),
				"replicate":           flag(false), "compression": str("LZ4"), "speed_limit": num(1048576),
				"large_block": flag(false), "embed": flag(true), "compressed": flag(false), "sudo": flag(true),
				"logging_level": str("DEBUG"), "name_regex": str("auto-.*"),
				"lifetimes": replLifetimes([3]string{"0", "2", "WEEK"}),
			},
			want: map[string]interface{}{
				"readonly": "IGNORE", "allow_from_scratch": true, "retries": float64(3),
				"exclude":    []interface{}{"tank/src/cache", "tank/src/tmp"},
				"properties": false, "properties_exclude": []interface{}{"mountpoint"},
				"properties_override": map[string]interface{}{"atime": "off"},
				"replicate":           false, "compression": "LZ4", "speed_limit": float64(1048576),
				"large_block": false, "embed": true, "compressed": false, "sudo": true,
				"logging_level": "DEBUG", "name_regex": "auto-.*",
				"lifetimes": []interface{}{map[string]interface{}{
					"schedule":       map[string]interface{}{"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*"},
					"lifetime_value": float64(2), "lifetime_unit": "WEEK",
				}},
			},
		},
		{
			// The empty value of a nullable field is sent as null, and an
			// empty collection as itself: both clear what TrueNAS holds.
			name: "empty values clear",
			planned: map[string]tftypes.Value{
				"compression": str(""), "speed_limit": num(0), "logging_level": str(""), "name_regex": str(""),
				"exclude": replStringSet(), "properties_exclude": replStringSet(),
				"properties_override": replStringMap(nil), "lifetimes": replLifetimes(),
			},
			want: map[string]interface{}{
				"compression": nil, "speed_limit": nil, "logging_level": nil, "name_regex": nil,
				"exclude": []interface{}{}, "properties_exclude": []interface{}{},
				"properties_override": map[string]interface{}{}, "lifetimes": []interface{}{},
			},
		},
		{
			name: "encryption and netcat",
			planned: map[string]tftypes.Value{
				"encryption": flag(true), "encryption_inherit": flag(false),
				"encryption_key": str("correct horse battery staple"), "encryption_key_format": str("PASSPHRASE"),
				"encryption_key_location": str("/mnt/keys/rep"), "netcat_active_side": str("LOCAL"),
				"netcat_active_side_listen_address": str("192.0.2.5"), "netcat_active_side_port_min": num(2000),
				"netcat_active_side_port_max": num(2100), "netcat_passive_side_connect_address": str("192.0.2.9"),
			},
			want: map[string]interface{}{
				"encryption": true, "encryption_inherit": false,
				"encryption_key": "correct horse battery staple", "encryption_key_format": "PASSPHRASE",
				"encryption_key_location": "/mnt/keys/rep", "netcat_active_side": "LOCAL",
				"netcat_active_side_listen_address": "192.0.2.5", "netcat_active_side_port_min": float64(2000),
				"netcat_active_side_port_max": float64(2100), "netcat_passive_side_connect_address": "192.0.2.9",
			},
		},
		{
			name: "empty encryption and netcat values clear",
			planned: map[string]tftypes.Value{
				"encryption": flag(false), "encryption_key": str(""), "encryption_key_format": str(""),
				"encryption_key_location": str(""), "netcat_active_side": str(""),
				"netcat_active_side_listen_address": str(""), "netcat_active_side_port_min": num(0),
				"netcat_active_side_port_max": num(0), "netcat_passive_side_connect_address": str(""),
			},
			want: map[string]interface{}{
				"encryption": false, "encryption_key": nil, "encryption_key_format": nil,
				"encryption_key_location": nil, "netcat_active_side": nil,
				"netcat_active_side_listen_address": nil, "netcat_active_side_port_min": nil,
				"netcat_active_side_port_max": nil, "netcat_passive_side_connect_address": nil,
			},
		},
	}
	check := func(t *testing.T, sent, want map[string]interface{}) {
		t.Helper()
		if sent == nil {
			t.Fatal("the request was not sent")
		}
		for key, w := range want {
			got, ok := sent[key]
			if !ok {
				t.Errorf("%s was left out of the request, want %#v", key, w)
				continue
			}
			if !reflect.DeepEqual(got, w) {
				t.Errorf("%s = %#v, want %#v", key, got, w)
			}
		}
	}
	for _, tc := range cases {
		t.Run(tc.name+", create", func(t *testing.T) {
			var sent map[string]interface{}
			r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
			sch := schemaOf(t, ctx, r)
			resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
			r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, replVals(tc.planned))}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Create: %v", resp.Diagnostics)
			}
			check(t, sent, tc.want)
		})
		t.Run(tc.name+", update", func(t *testing.T) {
			var sent map[string]interface{}
			r := &ReplicationResource{client: replCaptureClient(ctx, t, replBody(nil), &sent)}
			sch := schemaOf(t, ctx, r)
			state := stateFromValues(t, ctx, sch, replVals(nil))
			resp := &resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: planFromValues(t, ctx, sch, replVals(tc.planned))}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Update: %v", resp.Diagnostics)
			}
			check(t, sent, tc.want)
		})
	}
}

// What TrueNAS holds as null is read back as the attribute's empty value, so
// that no option is ever null in state.
func TestReplicationResource_Read_MapsOptions(t *testing.T) {
	ctx := context.Background()
	read := func(t *testing.T, body map[string]interface{}) ReplicationResourceModel {
		t.Helper()
		var sent map[string]interface{}
		r := &ReplicationResource{client: replCaptureClient(ctx, t, body, &sent)}
		sch := schemaOf(t, ctx, r)
		state := stateFromValues(t, ctx, sch, replVals(nil))
		resp := &resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, resp)
		var got ReplicationResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		return got
	}

	t.Run("defaults and nulls", func(t *testing.T) {
		got := read(t, replBody(nil))
		if got.Readonly.ValueString() != "SET" || got.Retries.ValueInt64() != 5 || !got.Properties.ValueBool() ||
			!got.LargeBlock.ValueBool() || !got.Compressed.ValueBool() {
			t.Errorf("defaults lost: readonly %v retries %v properties %v large_block %v compressed %v",
				got.Readonly, got.Retries, got.Properties, got.LargeBlock, got.Compressed)
		}
		for name, v := range map[string]interface{ IsNull() bool }{
			"compression": got.Compression, "speed_limit": got.SpeedLimit, "logging_level": got.LoggingLevel,
			"name_regex": got.NameRegex, "exclude": got.Exclude, "properties_exclude": got.PropertiesExclude,
			"properties_override": got.PropertiesOverride, "lifetimes": got.Lifetimes,
		} {
			if v.IsNull() {
				t.Errorf("%s is null in state, want its empty value", name)
			}
		}
		if got.Compression.ValueString() != "" || got.SpeedLimit.ValueInt64() != 0 {
			t.Errorf("compression = %v, speed_limit = %v, want the empty values", got.Compression, got.SpeedLimit)
		}
	})

	t.Run("values", func(t *testing.T) {
		got := read(t, replBody(map[string]interface{}{
			"readonly": "REQUIRE", "allow_from_scratch": true, "replicate": true, "embed": true, "sudo": true,
			"compression": "PLZIP", "speed_limit": 4096, "logging_level": "WARNING", "name_regex": "snap-.*",
			"exclude": []interface{}{"tank/src/tmp"}, "properties_exclude": []interface{}{"mountpoint"},
			"properties_override": map[string]interface{}{"atime": "off"},
			"lifetimes": []interface{}{map[string]interface{}{
				"schedule":       map[string]interface{}{"minute": "0", "hour": "3", "dom": "*", "month": "*", "dow": "1"},
				"lifetime_value": 6, "lifetime_unit": "MONTH",
			}},
		}))
		if got.Readonly.ValueString() != "REQUIRE" || !got.AllowFromScratch.ValueBool() || !got.Replicate.ValueBool() ||
			!got.Embed.ValueBool() || !got.Sudo.ValueBool() {
			t.Errorf("scalars lost: %v %v %v %v %v", got.Readonly, got.AllowFromScratch, got.Replicate, got.Embed, got.Sudo)
		}
		if got.Compression.ValueString() != "PLZIP" || got.SpeedLimit.ValueInt64() != 4096 ||
			got.LoggingLevel.ValueString() != "WARNING" || got.NameRegex.ValueString() != "snap-.*" {
			t.Errorf("nullable values lost: %v %v %v %v", got.Compression, got.SpeedLimit, got.LoggingLevel, got.NameRegex)
		}
		var lifetimes []replicationLifetimeModel
		if d := got.Lifetimes.ElementsAs(ctx, &lifetimes, false); d.HasError() {
			t.Fatalf("lifetimes: %v", d)
		}
		if len(lifetimes) != 1 || lifetimes[0].Schedule.Hour.ValueString() != "3" || lifetimes[0].Schedule.Dow.ValueString() != "1" ||
			lifetimes[0].LifetimeValue.ValueInt64() != 6 || lifetimes[0].LifetimeUnit.ValueString() != "MONTH" {
			t.Errorf("lifetimes = %+v", lifetimes)
		}
		var overrides map[string]string
		if d := got.PropertiesOverride.ElementsAs(ctx, &overrides, false); d.HasError() || overrides["atime"] != "off" {
			t.Errorf("properties_override = %v (%v)", overrides, d)
		}
		if len(got.Exclude.Elements()) != 1 || len(got.PropertiesExclude.Elements()) != 1 {
			t.Errorf("sets lost: %v %v", got.Exclude, got.PropertiesExclude)
		}
	})
}

func TestReplicationResource_Read_MapsEncryptionAndNetcat(t *testing.T) {
	ctx := context.Background()
	read := func(t *testing.T, body map[string]interface{}) ReplicationResourceModel {
		t.Helper()
		var sent map[string]interface{}
		r := &ReplicationResource{client: replCaptureClient(ctx, t, body, &sent)}
		sch := schemaOf(t, ctx, r)
		state := stateFromValues(t, ctx, sch, replVals(nil))
		resp := &resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, resp)
		var got ReplicationResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		return got
	}

	// A task without encryption or netcat holds nulls. They are read back
	// as the empty values, never as null.
	none := read(t, replBody(nil))
	if none.Encryption.ValueBool() || none.EncryptionInherit.IsNull() || none.EncryptionInherit.ValueBool() {
		t.Errorf("encryption = %v, encryption_inherit = %v, want false and false", none.Encryption, none.EncryptionInherit)
	}
	for name, v := range map[string]types.String{
		"encryption_key": none.EncryptionKey, "encryption_key_format": none.EncryptionKeyFormat,
		"encryption_key_location": none.EncryptionKeyLocation, "netcat_active_side": none.NetcatActiveSide,
		"netcat_active_side_listen_address":   none.NetcatActiveSideListenAddress,
		"netcat_passive_side_connect_address": none.NetcatPassiveSideConnectAddress,
	} {
		if v.IsNull() || v.ValueString() != "" {
			t.Errorf("%s = %v, want the empty string", name, v)
		}
	}
	if none.NetcatActiveSidePortMin.IsNull() || none.NetcatActiveSidePortMin.ValueInt64() != 0 ||
		none.NetcatActiveSidePortMax.IsNull() || none.NetcatActiveSidePortMax.ValueInt64() != 0 {
		t.Errorf("ports = %v %v, want 0 and 0", none.NetcatActiveSidePortMin, none.NetcatActiveSidePortMax)
	}

	set := read(t, replBody(map[string]interface{}{
		"transport": "SSH+NETCAT", "encryption": true, "encryption_inherit": true,
		"encryption_key": "00ff", "encryption_key_format": "HEX", "encryption_key_location": "/mnt/keys/rep",
		"netcat_active_side": "REMOTE", "netcat_active_side_listen_address": "192.0.2.5",
		"netcat_active_side_port_min": 2000, "netcat_active_side_port_max": 2100,
		"netcat_passive_side_connect_address": "192.0.2.9",
	}))
	if !set.Encryption.ValueBool() || !set.EncryptionInherit.ValueBool() || set.EncryptionKey.ValueString() != "00ff" ||
		set.EncryptionKeyFormat.ValueString() != "HEX" || set.EncryptionKeyLocation.ValueString() != "/mnt/keys/rep" {
		t.Errorf("encryption values lost: %v %v %v %v", set.Encryption, set.EncryptionInherit, set.EncryptionKeyFormat, set.EncryptionKeyLocation)
	}
	if set.NetcatActiveSide.ValueString() != "REMOTE" || set.NetcatActiveSideListenAddress.ValueString() != "192.0.2.5" ||
		set.NetcatActiveSidePortMin.ValueInt64() != 2000 || set.NetcatActiveSidePortMax.ValueInt64() != 2100 ||
		set.NetcatPassiveSideConnectAddress.ValueString() != "192.0.2.9" {
		t.Errorf("netcat values lost: %v %v %v %v %v", set.NetcatActiveSide, set.NetcatActiveSideListenAddress,
			set.NetcatActiveSidePortMin, set.NetcatActiveSidePortMax, set.NetcatPassiveSideConnectAddress)
	}
}

// The key is marked sensitive, so Terraform masks it in plans and output.
func TestReplicationResource_EncryptionKeyIsSensitive(t *testing.T) {
	ctx := context.Background()
	sch := schemaOf(t, ctx, &ReplicationResource{})
	if !sch.Schema.Attributes["encryption_key"].IsSensitive() {
		t.Error("encryption_key is not marked sensitive")
	}
}
