package resources

import (
	"context"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// resilverConfigBody is what pool.resilver.config returns once the window
// is switched on with its default hours.
func resilverConfigBody() map[string]interface{} {
	return map[string]interface{}{
		"id":      1,
		"enabled": true,
		"begin":   "18:00",
		"end":     "09:00",
		"weekday": []int{1, 2, 3, 4, 5, 6, 7},
	}
}

func resilverClient(t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return body, nil
		})
}

// failingResilverClient answers every call with an error, so the CRUD
// handlers take their diagnostic branches.
func failingResilverClient(t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

func resilverDaySet(days ...int64) tftypes.Value {
	vals := make([]tftypes.Value, 0, len(days))
	for _, day := range days {
		vals = append(vals, num(day))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, vals)
}

func resilverDays(days ...int64) types.Set {
	vals := make([]attr.Value, 0, len(days))
	for _, day := range days {
		vals = append(vals, types.Int64Value(day))
	}
	return types.SetValueMust(types.Int64Type, vals)
}

func TestResilverConfigResource_CRUD(t *testing.T) {
	c := resilverClient(t, resilverConfigBody())
	r := &ResilverConfigResource{client: c}
	crudDrive(t, r, c, resilverConfigSingletonID, map[string]tftypes.Value{
		"enabled": flag(true),
		"begin":   str("18:00"),
		"end":     str("09:00"),
		"weekday": resilverDaySet(1, 2, 3, 4, 5, 6, 7),
	})
}

// State has to hold what TrueNAS returned, not what the plan asked for: a
// plan that sets one attribute still gets the other three from the server.
func TestResilverConfigResource_CreateStoresServerValues(t *testing.T) {
	ctx := context.Background()
	body := resilverConfigBody()
	body["begin"] = "22:30"
	body["weekday"] = []int{6, 7}
	r := &ResilverConfigResource{client: resilverClient(t, body)}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"enabled": flag(true)})
	resp := &resource.CreateResponse{State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{})}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var got ResilverConfigResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if got.ID.ValueString() != resilverConfigSingletonID {
		t.Errorf("id = %q", got.ID.ValueString())
	}
	if !got.Enabled.ValueBool() || got.Begin.ValueString() != "22:30" || got.End.ValueString() != "09:00" {
		t.Errorf("state = %+v", got)
	}
	if !got.Weekday.Equal(resilverDays(6, 7)) {
		t.Errorf("weekday = %v", got.Weekday)
	}
}

// updateRequest must send only what the plan set: pool.resilver.update is
// a ForUpdateMetaclass model, so an omitted key keeps the stored value.
func TestResilverConfigResource_updateRequest(t *testing.T) {
	ctx := context.Background()
	r := &ResilverConfigResource{}

	t.Run("unset attributes are left out", func(t *testing.T) {
		var diags diag.Diagnostics
		req := r.updateRequest(ctx, &ResilverConfigResourceModel{
			Enabled: types.BoolValue(true),
			Begin:   types.StringNull(),
			End:     types.StringUnknown(),
			Weekday: types.SetNull(types.Int64Type),
		}, &diags)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if req.Enabled == nil || !*req.Enabled {
			t.Errorf("enabled = %v", req.Enabled)
		}
		if req.Begin != nil || req.End != nil || req.Weekday != nil {
			t.Errorf("unset attributes were sent: %+v", req)
		}
	})

	t.Run("unknown weekday is left out", func(t *testing.T) {
		var diags diag.Diagnostics
		req := r.updateRequest(ctx, &ResilverConfigResourceModel{
			Weekday: types.SetUnknown(types.Int64Type),
		}, &diags)
		if req.Weekday != nil || req.Enabled != nil {
			t.Errorf("unknown attributes were sent: %+v", req)
		}
	})

	// Switching the window off is a real value, not an unset one.
	t.Run("enabled false is sent", func(t *testing.T) {
		var diags diag.Diagnostics
		req := r.updateRequest(ctx, &ResilverConfigResourceModel{Enabled: types.BoolValue(false)}, &diags)
		if req.Enabled == nil || *req.Enabled {
			t.Errorf("enabled = %v, want a pointer to false", req.Enabled)
		}
	})

	// A set has no order. Sorting gives the same request for the same set.
	t.Run("the whole window is sent with sorted days", func(t *testing.T) {
		var diags diag.Diagnostics
		req := r.updateRequest(ctx, &ResilverConfigResourceModel{
			Begin:   types.StringValue("22:30"),
			End:     types.StringValue("06:00"),
			Weekday: resilverDays(7, 1, 6),
		}, &diags)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if req.Begin == nil || *req.Begin != "22:30" || req.End == nil || *req.End != "06:00" {
			t.Errorf("window = %v to %v", req.Begin, req.End)
		}
		if !slices.Equal(req.Weekday, []int{1, 6, 7}) {
			t.Errorf("weekday = %v, want [1 6 7]", req.Weekday)
		}
	})
}

// The schema validators are what stand between a configuration and a
// permanent diff ("9:00" reads back as "09:00") or a server-side rejection
// (no days, or a day outside 1 to 7).
func TestResilverConfigResource_TimeValidators(t *testing.T) {
	ctx := context.Background()
	attrs := schemaOf(t, ctx, &ResilverConfigResource{}).Schema.Attributes

	cases := []struct {
		value   string
		wantErr bool
	}{
		{"09:00", false},
		{"00:00", false},
		{"23:59", false},
		{"9:00", true}, // accepted by TrueNAS, read back as 09:00
		{"24:00", true},
		{"18:60", true},
		{"18:5", true},
		{"18:00:00", true},
		{"morning", true},
		{"", true},
	}
	for _, name := range []string{"begin", "end"} {
		timeAttr, ok := attrs[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is %T, want a string attribute", name, attrs[name])
		}
		if len(timeAttr.Validators) == 0 {
			t.Fatalf("%s has no validators", name)
		}
		for _, tc := range cases {
			t.Run(name+" "+tc.value, func(t *testing.T) {
				resp := &validator.StringResponse{}
				for _, v := range timeAttr.Validators {
					v.ValidateString(ctx, validator.StringRequest{
						Path:        path.Root(name),
						ConfigValue: types.StringValue(tc.value),
					}, resp)
				}
				if got := resp.Diagnostics.HasError(); got != tc.wantErr {
					t.Errorf("HasError = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
				}
			})
		}
	}
}

func TestResilverConfigResource_WeekdayValidators(t *testing.T) {
	ctx := context.Background()
	attrs := schemaOf(t, ctx, &ResilverConfigResource{}).Schema.Attributes
	weekday, ok := attrs["weekday"].(schema.SetAttribute)
	if !ok {
		t.Fatalf("weekday is %T, want a set attribute", attrs["weekday"])
	}
	if len(weekday.Validators) == 0 {
		t.Fatal("weekday has no validators")
	}

	cases := []struct {
		name    string
		days    types.Set
		wantErr bool
	}{
		{"every day", resilverDays(1, 2, 3, 4, 5, 6, 7), false},
		{"one day", resilverDays(7), false},
		{"no days is rejected by TrueNAS", resilverDays(), true},
		{"0 is cron's Sunday, not TrueNAS's", resilverDays(0, 1), true},
		{"8", resilverDays(8), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.SetResponse{}
			for _, v := range weekday.Validators {
				v.ValidateSet(ctx, validator.SetRequest{
					Path:        path.Root("weekday"),
					ConfigValue: tc.days,
				}, resp)
			}
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// The singleton has exactly one valid import ID. Anything else would put
// state on a resource that does not exist.
func TestResilverConfigResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &ResilverConfigResource{client: resilverClient(t, resilverConfigBody())}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{})

	ok := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: resilverConfigSingletonID}, ok)
	if ok.Diagnostics.HasError() {
		t.Errorf("the canonical import ID was rejected: %v", ok.Diagnostics)
	}
	var id types.String
	if diags := ok.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() || id.ValueString() != resilverConfigSingletonID {
		t.Errorf("imported id = %v (%v)", id, diags)
	}

	bad := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("ImportState accepted an ID other than the singleton's")
	}
}

// A failed write or read has to surface as a diagnostic. Swallowing it
// would leave state claiming a window the server never accepted.
func TestResilverConfigResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &ResilverConfigResource{client: failingResilverClient(t)}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"id":      str(resilverConfigSingletonID),
		"enabled": flag(true),
	}
	st := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)

	cResp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create reported success against a server that rejected the write")
	}

	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read reported success against a server that returned an error")
	}

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update reported success against a server that rejected the write")
	}
}

// Removing the resource must not touch the server: a destroy that switched
// the window off would slow down a resilver that is running.
func TestResilverConfigResource_DeleteSendsNothing(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			calls++
			return resilverConfigBody(), nil
		})
	r := &ResilverConfigResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(resilverConfigSingletonID)})

	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", resp.Diagnostics)
	}
	if calls != 0 {
		t.Errorf("Delete made %d API calls, want none", calls)
	}
}

// A plan or state that fails to decode must stop the handler, not be
// applied half-read.
func TestResilverConfigResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &ResilverConfigResource{client: failingResilverClient(t)}
	sch := schemaOf(t, ctx, r)
	// A raw value of the wrong shape for the schema: Get returns an error.
	bogus := tftypes.NewValue(tftypes.String, "not-an-object")

	cResp := &resource.CreateResponse{}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: bogus}}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create accepted a plan it could not decode")
	}

	rResp := &resource.ReadResponse{}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: sch.Schema, Raw: bogus}}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted a state it could not decode")
	}

	uResp := &resource.UpdateResponse{}
	r.Update(ctx, resource.UpdateRequest{
		State: tfsdk.State{Schema: sch.Schema, Raw: bogus},
		Plan:  tfsdk.Plan{Schema: sch.Schema, Raw: bogus},
	}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted a plan it could not decode")
	}
}
