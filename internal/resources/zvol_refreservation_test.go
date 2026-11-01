package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// zvolBody is a pool.dataset response for a zvol with the given
// refreservation rawvalue ("" leaves the property out).
func zvolBody(refres string) map[string]interface{} {
	b := map[string]interface{}{
		"id": "tank/vol1", "name": "tank/vol1", "pool": "tank", "type": "VOLUME",
		"volsize":      map[string]interface{}{"value": "16M", "rawvalue": "16777216"},
		"volblocksize": map[string]interface{}{"value": "16K"},
		"compression":  map[string]interface{}{"value": "LZ4"},
	}
	if refres != "" {
		b["refreservation"] = map[string]interface{}{"value": refres, "rawvalue": refres}
	}
	return b
}

// zvolCaptureClient answers every pool.dataset call with body and records
// the request map of the last create or update.
func zvolCaptureClient(ctx context.Context, t *testing.T, body map[string]interface{}, sent *map[string]interface{}) *wsclient.Client {
	t.Helper()
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "pool.dataset.create":
			if m, ok := params[0].(map[string]interface{}); ok {
				*sent = m
			}
		case "pool.dataset.update":
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

func zvolVals(refres tftypes.Value) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id": str("tank/vol1"), "name": str("vol1"), "pool": str("tank"),
		"volsize": num(16777216), "refreservation": refres,
	}
}

func TestZvolResource_Create_Refreservation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		planned  tftypes.Value
		wantSent bool
		wantVal  float64
	}{
		{"explicit zero is sent (thin)", num(0), true, 0},
		{"unknown is left out (server default, thick)", tftypes.NewValue(tftypes.Number, tftypes.UnknownValue), false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sent map[string]interface{}
			r := &ZvolResource{client: zvolCaptureClient(ctx, t, zvolBody("0"), &sent)}
			sch := schemaOf(t, ctx, r)
			resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
			r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, zvolVals(tc.planned))}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Create: %v", resp.Diagnostics)
			}
			v, ok := sent["refreservation"]
			if ok != tc.wantSent {
				t.Fatalf("refreservation sent = %v (%v), want sent = %v", ok, v, tc.wantSent)
			}
			if ok && v.(float64) != tc.wantVal {
				t.Errorf("refreservation = %v, want %v", v, tc.wantVal)
			}
		})
	}
}

func TestZvolResource_Update_RefreservationOnlyWhenChanged(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name          string
		state, planed tftypes.Value
		wantSent      bool
	}{
		{"unchanged read-back value is not re-sent", num(17825792), num(17825792), false},
		{"changed value is sent in place", num(17825792), num(0), true},
	}
	// A pinned value is re-sent with a volsize change, so ZFS cannot leave a
	// recomputed reservation behind.
	t.Run("pinned value is re-sent when volsize changes", func(t *testing.T) {
		var sent map[string]interface{}
		r := &ZvolResource{client: zvolCaptureClient(ctx, t, zvolBody("17825792"), &sent)}
		sch := schemaOf(t, ctx, r)
		state := stateFromValues(t, ctx, sch, zvolVals(num(17825792)))
		grown := zvolVals(num(17825792))
		grown["volsize"] = num(33554432)
		resp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{State: state, Plan: planFromValues(t, ctx, sch, grown)}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update: %v", resp.Diagnostics)
		}
		if v, ok := sent["refreservation"]; !ok || v.(float64) != 17825792 {
			t.Errorf("refreservation = %v (sent %v), want 17825792 re-sent", v, ok)
		}
	})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sent map[string]interface{}
			r := &ZvolResource{client: zvolCaptureClient(ctx, t, zvolBody("0"), &sent)}
			sch := schemaOf(t, ctx, r)
			state := stateFromValues(t, ctx, sch, zvolVals(tc.state))
			resp := &resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: planFromValues(t, ctx, sch, zvolVals(tc.planed))}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Update: %v", resp.Diagnostics)
			}
			if sent == nil {
				t.Fatal("pool.dataset.update was not called")
			}
			if _, ok := sent["refreservation"]; ok != tc.wantSent {
				t.Errorf("refreservation sent = %v, want %v (request %v)", ok, tc.wantSent, sent)
			}
		})
	}
}

func TestZvolResource_ModifyPlan_VolsizeChangeUnpinsRefreservation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name        string
		configured  tftypes.Value
		newVolsize  int64
		wantUnknown bool
	}{
		{"unset, volsize grows: ZFS recomputes it", tftypes.NewValue(tftypes.Number, nil), 33554432, true},
		{"pinned to 0, volsize grows: stays 0", num(0), 33554432, false},
		{"unset, volsize unchanged: kept from state", tftypes.NewValue(tftypes.Number, nil), 16777216, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &ZvolResource{}
			sch := schemaOf(t, ctx, r)
			state := stateFromValues(t, ctx, sch, zvolVals(num(17825792)))
			planVals := zvolVals(num(17825792))
			planVals["volsize"] = num(tc.newVolsize)
			cfgVals := zvolVals(tc.configured)
			cfgVals["volsize"] = num(tc.newVolsize)
			plan := planFromValues(t, ctx, sch, planVals)
			config := tfsdk.Config{Schema: plan.Schema, Raw: planFromValues(t, ctx, sch, cfgVals).Raw}
			resp := &resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
			}
			var got types.Int64
			resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("refreservation"), &got)...)
			if got.IsUnknown() != tc.wantUnknown {
				t.Errorf("refreservation unknown = %v, want %v (value %v)", got.IsUnknown(), tc.wantUnknown, got)
			}
		})
	}
}

// A config that does not decode into the model stops ModifyPlan with an
// error diagnostic instead of planning on a half-read value.
func TestZvolResource_ModifyPlan_UndecodableConfig(t *testing.T) {
	ctx := context.Background()
	r := &ZvolResource{}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, zvolVals(num(17825792)))
	state := stateFromValues(t, ctx, sch, zvolVals(num(17825792)))
	bad := tfsdk.Config{Schema: plan.Schema, Raw: tftypes.NewValue(tftypes.String, "not an object")}
	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: bad, Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("ModifyPlan accepted an undecodable config without an error diagnostic")
	}
}

func TestZvolResource_MapResponseToModel_Refreservation(t *testing.T) {
	r := &ZvolResource{}
	cases := []struct {
		name  string
		prop  *truenas.PropertyRawVal
		prior types.Int64
		want  types.Int64
	}{
		{"none reads as 0", &truenas.PropertyRawVal{Value: "none", Rawvalue: "0"}, types.Int64Unknown(), types.Int64Value(0)},
		{"thick reservation in bytes", &truenas.PropertyRawVal{Value: "17M", Rawvalue: "17825792"}, types.Int64Unknown(), types.Int64Value(17825792)},
		{"absent and unknown becomes null", nil, types.Int64Unknown(), types.Int64Null()},
		{"absent keeps a known value", nil, types.Int64Value(5), types.Int64Value(5)},
		{"unparsable is treated as absent", &truenas.PropertyRawVal{Rawvalue: "abc"}, types.Int64Unknown(), types.Int64Null()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := ZvolResourceModel{Refreservation: tc.prior}
			r.mapResponseToModel(&truenas.DatasetResponse{ID: "tank/v", Type: "VOLUME", Refreservation: tc.prop}, &m)
			if !m.Refreservation.Equal(tc.want) {
				t.Errorf("Refreservation = %v, want %v", m.Refreservation, tc.want)
			}
		})
	}
}

func TestZvolResource_Schema_Refreservation(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewZvolResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	a, ok := resp.Schema.GetAttributes()["refreservation"]
	if !ok {
		t.Fatal("missing attribute refreservation")
	}
	if !a.IsOptional() || !a.IsComputed() || a.IsRequired() {
		t.Errorf("refreservation should be optional+computed, got optional=%v computed=%v required=%v",
			a.IsOptional(), a.IsComputed(), a.IsRequired())
	}
}
