package resources

import (
	"context"
	"maps"
	"reflect"
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
