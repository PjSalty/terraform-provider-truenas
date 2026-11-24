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

// iscsiGlobalBody is what iscsi.global.config returns with a server and a
// threshold set.
func iscsiGlobalBody() map[string]interface{} {
	return map[string]interface{}{
		"id":                   1,
		"basename":             "iqn.2005-10.org.freenas.ctl",
		"isns_servers":         []string{"10.0.0.5", "10.0.0.6:3205"},
		"listen_port":          3260,
		"pool_avail_threshold": 80,
		"alua":                 false,
		"iser":                 false,
	}
}

func iscsiGlobalClient(ctx context.Context, t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return body, nil
		})
}

// failingISCSIGlobalClient answers every call with an error, so the CRUD
// handlers take their diagnostic branches.
func failingISCSIGlobalClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

func iscsiServerSet(servers ...string) tftypes.Value {
	vals := make([]tftypes.Value, 0, len(servers))
	for _, s := range servers {
		vals = append(vals, str(s))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, vals)
}

func iscsiServers(servers ...string) types.Set {
	vals := make([]attr.Value, 0, len(servers))
	for _, s := range servers {
		vals = append(vals, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, vals)
}

func TestISCSIGlobalResource_CRUD(t *testing.T) {
	ctx := context.Background()
	c := iscsiGlobalClient(ctx, t, iscsiGlobalBody())
	r := &ISCSIGlobalResource{client: c}
	crudDrive(t, r, c, iscsiGlobalSingletonID, map[string]tftypes.Value{
		"basename":             str("iqn.2005-10.org.freenas.ctl"),
		"isns_servers":         iscsiServerSet("10.0.0.5", "10.0.0.6:3205"),
		"listen_port":          num(3260),
		"pool_avail_threshold": num(80),
	})
}

// State has to hold what TrueNAS returned, not what the plan asked for, and
// a null threshold has to read back as 0 so the plan settles.
func TestISCSIGlobalResource_CreateStoresServerValues(t *testing.T) {
	ctx := context.Background()
	body := iscsiGlobalBody()
	body["pool_avail_threshold"] = nil
	body["basename"] = "iqn.2024-01.example:ctl"
	body["iser"] = true
	r := &ISCSIGlobalResource{client: iscsiGlobalClient(ctx, t, body)}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"listen_port": num(3260)})
	resp := &resource.CreateResponse{State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{})}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var got ISCSIGlobalResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if got.ID.ValueString() != iscsiGlobalSingletonID {
		t.Errorf("id = %q", got.ID.ValueString())
	}
	if got.Basename.ValueString() != "iqn.2024-01.example:ctl" || got.ListenPort.ValueInt64() != 3260 {
		t.Errorf("state = %+v", got)
	}
	if got.PoolAvailThreshold.ValueInt64() != 0 {
		t.Errorf("a null threshold reads back as %v, want 0", got.PoolAvailThreshold)
	}
	if !got.IsnsServers.Equal(iscsiServers("10.0.0.5", "10.0.0.6:3205")) {
		t.Errorf("isns_servers = %v", got.IsnsServers)
	}
	if got.Alua.ValueBool() || !got.Iser.ValueBool() {
		t.Errorf("alua = %v, iser = %v", got.Alua, got.Iser)
	}
}

// Every update has to carry the iSNS servers: iscsi.global.update empties
// them when the key is missing. The server here records what it was sent.
func TestISCSIGlobalResource_updateRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("unset attributes are left out and the current servers are sent", func(t *testing.T) {
		r := &ISCSIGlobalResource{client: iscsiGlobalClient(ctx, t, iscsiGlobalBody())}
		var diags diag.Diagnostics
		req, ok := r.updateRequest(ctx, &ISCSIGlobalResourceModel{
			Basename:           types.StringNull(),
			IsnsServers:        types.SetNull(types.StringType),
			ListenPort:         types.Int64Unknown(),
			PoolAvailThreshold: types.Int64Null(),
		}, &diags)
		if !ok || diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if req.Basename != nil || req.ListenPort != nil || !req.PoolAvailThreshold.IsZero() {
			t.Errorf("unset attributes were sent: %+v", req)
		}
		if !slices.Equal(req.IsnsServers, []string{"10.0.0.5", "10.0.0.6:3205"}) {
			t.Errorf("isns_servers = %v, want the current ones", req.IsnsServers)
		}
	})

	t.Run("unknown servers are read from the server too", func(t *testing.T) {
		r := &ISCSIGlobalResource{client: iscsiGlobalClient(ctx, t, iscsiGlobalBody())}
		var diags diag.Diagnostics
		req, ok := r.updateRequest(ctx, &ISCSIGlobalResourceModel{IsnsServers: types.SetUnknown(types.StringType)}, &diags)
		if !ok || len(req.IsnsServers) != 2 {
			t.Errorf("ok = %v, servers = %v", ok, req.IsnsServers)
		}
	})

	t.Run("no servers anywhere is an empty list, not null", func(t *testing.T) {
		body := iscsiGlobalBody()
		body["isns_servers"] = nil
		r := &ISCSIGlobalResource{client: iscsiGlobalClient(ctx, t, body)}
		var diags diag.Diagnostics
		req, ok := r.updateRequest(ctx, &ISCSIGlobalResourceModel{IsnsServers: types.SetNull(types.StringType)}, &diags)
		if !ok || req.IsnsServers == nil || len(req.IsnsServers) != 0 {
			t.Errorf("ok = %v, servers = %#v, want an empty non-nil list", ok, req.IsnsServers)
		}
	})

	t.Run("an empty planned set clears the servers", func(t *testing.T) {
		r := &ISCSIGlobalResource{client: failingISCSIGlobalClient(ctx, t)}
		var diags diag.Diagnostics
		req, ok := r.updateRequest(ctx, &ISCSIGlobalResourceModel{IsnsServers: iscsiServers()}, &diags)
		if !ok || req.IsnsServers == nil || len(req.IsnsServers) != 0 {
			t.Errorf("ok = %v, servers = %#v", ok, req.IsnsServers)
		}
	})

	t.Run("planned servers are sent sorted without a read", func(t *testing.T) {
		r := &ISCSIGlobalResource{client: failingISCSIGlobalClient(ctx, t)}
		var diags diag.Diagnostics
		req, ok := r.updateRequest(ctx, &ISCSIGlobalResourceModel{
			Basename:           types.StringValue("iqn.2024-01.example:ctl"),
			IsnsServers:        iscsiServers("10.0.0.9", "10.0.0.1:3205"),
			ListenPort:         types.Int64Value(3261),
			PoolAvailThreshold: types.Int64Value(75),
		}, &diags)
		if !ok || diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if req.Basename == nil || *req.Basename != "iqn.2024-01.example:ctl" || req.ListenPort == nil || *req.ListenPort != 3261 {
			t.Errorf("request = %+v", req)
		}
		if req.PoolAvailThreshold.IsZero() {
			t.Error("a threshold of 75 was left out")
		}
		if !slices.Equal(req.IsnsServers, []string{"10.0.0.1:3205", "10.0.0.9"}) {
			t.Errorf("isns_servers = %v, want sorted", req.IsnsServers)
		}
	})

	// 0 is how the configuration says "off", and the API says null. It has to
	// be sent, or the threshold could never be removed again.
	t.Run("a threshold of 0 is sent as null", func(t *testing.T) {
		r := &ISCSIGlobalResource{client: failingISCSIGlobalClient(ctx, t)}
		var diags diag.Diagnostics
		req, _ := r.updateRequest(ctx, &ISCSIGlobalResourceModel{
			IsnsServers:        iscsiServers(),
			PoolAvailThreshold: types.Int64Value(0),
		}, &diags)
		if req.PoolAvailThreshold.IsZero() {
			t.Fatal("a threshold of 0 was left out, so it could not be removed")
		}
		raw, err := req.PoolAvailThreshold.MarshalJSON()
		if err != nil || string(raw) != "null" {
			t.Errorf("threshold marshals to %s (%v), want null", raw, err)
		}
	})

	t.Run("a failed read of the current servers stops the update", func(t *testing.T) {
		r := &ISCSIGlobalResource{client: failingISCSIGlobalClient(ctx, t)}
		var diags diag.Diagnostics
		req, ok := r.updateRequest(ctx, &ISCSIGlobalResourceModel{IsnsServers: types.SetNull(types.StringType)}, &diags)
		if ok || req != nil || !diags.HasError() {
			t.Errorf("ok = %v, req = %v, diags = %v", ok, req, diags)
		}
	})
}

func validateISCSIInt(ctx context.Context, t *testing.T, name string, v types.Int64) bool {
	t.Helper()
	a, ok := schemaOf(t, ctx, &ISCSIGlobalResource{}).Schema.Attributes[name].(schema.Int64Attribute)
	if !ok || len(a.Validators) == 0 {
		t.Fatalf("%s is not an int64 attribute with validators", name)
	}
	resp := &validator.Int64Response{}
	for _, val := range a.Validators {
		val.ValidateInt64(ctx, validator.Int64Request{Path: path.Root(name), ConfigValue: v}, resp)
	}
	return resp.Diagnostics.HasError()
}

// The validators are what stand between a configuration and a server-side
// rejection after a partial apply.
func TestISCSIGlobalResource_ListenPortValidator(t *testing.T) {
	ctx := context.Background()
	for port, wantErr := range map[int64]bool{
		1024: true, 1025: false, 3260: false, 65535: false, 65536: true, 0: true,
	} {
		if got := validateISCSIInt(ctx, t, "listen_port", types.Int64Value(port)); got != wantErr {
			t.Errorf("listen_port %d: HasError = %v, want %v", port, got, wantErr)
		}
	}
}

func TestISCSIGlobalResource_ThresholdValidator(t *testing.T) {
	ctx := context.Background()
	for v, wantErr := range map[int64]bool{
		-1: true, 0: false, 1: false, 99: false, 100: true,
	} {
		if got := validateISCSIInt(ctx, t, "pool_avail_threshold", types.Int64Value(v)); got != wantErr {
			t.Errorf("pool_avail_threshold %d: HasError = %v, want %v", v, got, wantErr)
		}
	}
}

func TestISCSIGlobalResource_BasenameValidator(t *testing.T) {
	ctx := context.Background()
	a, ok := schemaOf(t, ctx, &ISCSIGlobalResource{}).Schema.Attributes["basename"].(schema.StringAttribute)
	if !ok || len(a.Validators) == 0 {
		t.Fatal("basename is not a string attribute with validators")
	}
	for value, wantErr := range map[string]bool{"": true, "iqn.2005-10.org.freenas.ctl": false} {
		resp := &validator.StringResponse{}
		for _, v := range a.Validators {
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("basename"), ConfigValue: types.StringValue(value)}, resp)
		}
		if got := resp.Diagnostics.HasError(); got != wantErr {
			t.Errorf("basename %q: HasError = %v, want %v", value, got, wantErr)
		}
	}
}

// alua and iser are license and hardware gated, so the configuration must
// not be able to set them.
func TestISCSIGlobalResource_AluaAndIserAreReadOnly(t *testing.T) {
	ctx := context.Background()
	attrs := schemaOf(t, ctx, &ISCSIGlobalResource{}).Schema.Attributes
	for _, name := range []string{"alua", "iser"} {
		a, ok := attrs[name].(schema.BoolAttribute)
		if !ok {
			t.Fatalf("%s is %T, want a bool attribute", name, attrs[name])
		}
		if a.Optional || a.Required || !a.Computed {
			t.Errorf("%s: optional=%v required=%v computed=%v, want computed only", name, a.Optional, a.Required, a.Computed)
		}
	}
}

func TestISCSIGlobalResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &ISCSIGlobalResource{client: iscsiGlobalClient(ctx, t, iscsiGlobalBody())}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{})

	ok := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: iscsiGlobalSingletonID}, ok)
	if ok.Diagnostics.HasError() {
		t.Errorf("the canonical import ID was rejected: %v", ok.Diagnostics)
	}
	var id types.String
	if diags := ok.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() || id.ValueString() != iscsiGlobalSingletonID {
		t.Errorf("imported id = %v (%v)", id, diags)
	}

	bad := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("ImportState accepted an ID other than the singleton's")
	}
}

// A failed write or read has to surface as a diagnostic. The first plan has
// no servers, so Create fails reading them; the second has them, so it
// fails on the write itself.
func TestISCSIGlobalResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &ISCSIGlobalResource{client: failingISCSIGlobalClient(ctx, t)}
	sch := schemaOf(t, ctx, r)

	for name, vals := range map[string]map[string]tftypes.Value{
		"servers unknown": {"id": str(iscsiGlobalSingletonID), "listen_port": num(3260)},
		"servers planned": {"id": str(iscsiGlobalSingletonID), "isns_servers": iscsiServerSet("10.0.0.5")},
	} {
		t.Run(name, func(t *testing.T) {
			st := stateFromValues(t, ctx, sch, vals)
			plan := planFromValues(t, ctx, sch, vals)

			cResp := &resource.CreateResponse{State: st}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
			if !cResp.Diagnostics.HasError() {
				t.Error("Create reported success against a server that rejected the write")
			}

			uResp := &resource.UpdateResponse{State: st}
			r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
			if !uResp.Diagnostics.HasError() {
				t.Error("Update reported success against a server that rejected the write")
			}
		})
	}

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(iscsiGlobalSingletonID)})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read reported success against a server that returned an error")
	}
}

// Removing the resource must not touch the server: a reset would rename
// every target IQN.
func TestISCSIGlobalResource_DeleteSendsNothing(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			calls++
			return iscsiGlobalBody(), nil
		})
	r := &ISCSIGlobalResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(iscsiGlobalSingletonID)})

	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", resp.Diagnostics)
	}
	if calls != 0 {
		t.Errorf("Delete made %d API calls, want none", calls)
	}
}

// A plan or state that fails to decode must stop the handler.
func TestISCSIGlobalResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &ISCSIGlobalResource{client: failingISCSIGlobalClient(ctx, t)}
	sch := schemaOf(t, ctx, r)
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
