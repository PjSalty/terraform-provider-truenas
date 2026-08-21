package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func lxcConfigBody() map[string]interface{} {
	return map[string]interface{}{
		"id":             1,
		"preferred_pool": "tank",
		"bridge":         nil,
		"v4_network":     "172.200.0.0/24",
		"v6_network":     "fd42:4c58:43ae::/64",
	}
}

// lxcClient answers lxc.config / lxc.update with the given body and
// lxc.bridge_choices with the choices TrueNAS actually returns.
func lxcClient(t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			if method == "lxc.bridge_choices" {
				return map[string]interface{}{"[AUTO]": "Automatic", "br0": "br0"}, nil
			}
			return body, nil
		})
}

func TestLXCConfigResource_CRUD(t *testing.T) {
	c := lxcClient(t, lxcConfigBody())
	r := &LXCConfigResource{client: c}
	crudDrive(t, r, c, lxcConfigSingletonID, map[string]tftypes.Value{
		"preferred_pool": str("tank"),
	})
}

// A null bridge or preferred_pool must refresh as a known empty string.
// Reading them back as null makes every plan show "(known after apply)".
func TestLXCConfigResource_nullFieldsReadAsKnown(t *testing.T) {
	ctx := context.Background()
	body := lxcConfigBody()
	body["preferred_pool"] = nil
	r := &LXCConfigResource{client: lxcClient(t, body)}

	var m LXCConfigResourceModel
	if err := r.refreshState(ctx, &m); err != nil {
		t.Fatalf("refreshState: %v", err)
	}
	for name, v := range map[string]types.String{
		"preferred_pool": m.PreferredPool,
		"bridge":         m.Bridge,
	} {
		if v.IsNull() || v.IsUnknown() {
			t.Errorf("%s read back null/unknown; plans would show a phantom diff", name)
		}
		if v.ValueString() != "" {
			t.Errorf("%s = %q, want the empty string", name, v.ValueString())
		}
	}
	if m.ID.ValueString() != lxcConfigSingletonID {
		t.Errorf("id = %q", m.ID.ValueString())
	}
	if m.V4Network.ValueString() != "172.200.0.0/24" {
		t.Errorf("v4_network = %q", m.V4Network.ValueString())
	}
}

func TestLXCConfigResource_refreshStateError(t *testing.T) {
	c := newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
		})
	r := &LXCConfigResource{client: c}
	var m LXCConfigResourceModel
	err := r.refreshState(context.Background(), &m)
	if err == nil {
		t.Fatal("a missing lxc namespace was treated as a successful read")
	}
	if !strings.Contains(err.Error(), "26.0") {
		t.Errorf("diagnostic should name the required version, got: %v", err)
	}
}

// applyConfig must send only what the plan set: lxc.update is a
// ForUpdateMetaclass model, so an omitted key leaves the stored value
// alone while an empty string would clear it.
func TestLXCConfigResource_applyConfigSendsOnlySetFields(t *testing.T) {
	var sent map[string]interface{}
	c := newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			if method == "lxc.update" && len(params) > 0 {
				sent, _ = params[0].(map[string]interface{})
			}
			return lxcConfigBody(), nil
		})
	r := &LXCConfigResource{client: c}

	plan := &LXCConfigResourceModel{
		PreferredPool: types.StringValue("tank"),
		Bridge:        types.StringNull(),
		V4Network:     types.StringUnknown(),
		V6Network:     types.StringNull(),
	}
	if err := r.applyConfig(context.Background(), plan); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if sent["preferred_pool"] != "tank" {
		t.Errorf("preferred_pool = %v", sent["preferred_pool"])
	}
	for _, k := range []string{"bridge", "v4_network", "v6_network"} {
		if _, present := sent[k]; present {
			t.Errorf("%s sent for a plan that did not set it: %v", k, sent)
		}
	}
}

func TestLXCConfigResource_applyConfigBridgeValidation(t *testing.T) {
	ctx := context.Background()
	r := &LXCConfigResource{client: lxcClient(t, lxcConfigBody())}

	t.Run("named bridge that exists is accepted", func(t *testing.T) {
		if err := r.applyConfig(ctx, &LXCConfigResourceModel{Bridge: types.StringValue("br0")}); err != nil {
			t.Errorf("applyConfig: %v", err)
		}
	})

	t.Run("unknown bridge is rejected with the available list", func(t *testing.T) {
		err := r.applyConfig(ctx, &LXCConfigResourceModel{Bridge: types.StringValue("br9")})
		if err == nil {
			t.Fatal("a bridge that does not exist was accepted")
		}
		if !strings.Contains(err.Error(), "br0") {
			t.Errorf("diagnostic should list the available bridges, got: %v", err)
		}
	})

	// An empty bridge means "TrueNAS manages one". It names no interface,
	// so it is not checked against the choices list.
	t.Run("automatic bridge skips validation", func(t *testing.T) {
		if err := r.applyConfig(ctx, &LXCConfigResourceModel{Bridge: types.StringValue("")}); err != nil {
			t.Errorf("applyConfig(empty): %v", err)
		}
	})
}

func TestLXCConfigResource_applyConfigErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("bridge_choices failure surfaces", func(t *testing.T) {
		c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			if method == "lxc.bridge_choices" {
				return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EPERM] nope"}
			}
			return lxcConfigBody(), nil
		})
		r := &LXCConfigResource{client: c}
		err := r.applyConfig(ctx, &LXCConfigResourceModel{Bridge: types.StringValue("br0")})
		if err == nil || !strings.Contains(err.Error(), "bridge choices") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("update failure surfaces", func(t *testing.T) {
		c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] bad"}
		})
		r := &LXCConfigResource{client: c}
		err := r.applyConfig(ctx, &LXCConfigResourceModel{PreferredPool: types.StringValue("tank")})
		if err == nil || !strings.Contains(err.Error(), "applying LXC config") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestLXCBridgeNames(t *testing.T) {
	if got := lxcBridgeNames(map[string]string{"b": "b", "a": "a"}); got != "a, b" {
		t.Errorf("got %q, want sorted %q", got, "a, b")
	}
	// An empty list must still say what to do, not render as nothing.
	if got := lxcBridgeNames(nil); !strings.Contains(got, "leave bridge empty") {
		t.Errorf("empty choices rendered as %q", got)
	}
}

// ValidateConfig mirrors what lxc.update rejects server-side, so a bad CIDR
// fails at plan time naming the attribute instead of at apply time as a
// generic middleware validation error.
func TestLXCConfigResource_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &LXCConfigResource{}
	sch := schemaOf(t, ctx, r)

	cases := []struct {
		name    string
		v4, v6  string
		wantErr bool
	}{
		{"defaults are valid", "172.200.0.0/24", "fd42:4c58:43ae::/64", false},
		{"empty is valid: the field is optional", "", "", false},
		{"v4 not a CIDR", "172.200.0.1", "", true},
		{"v4 is garbage", "not-a-network", "", true},
		{"v6 address in the v4 field", "fd42::/64", "", true},
		{"v4 address in the v6 field", "", "172.200.0.0/24", true},
		{"v4 too small: /31 has 2 addresses", "172.200.0.0/31", "", true},
		{"v4 /30 has exactly 4 addresses", "172.200.0.0/30", "", false},
		{"v6 too small", "", "fd42::/127", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tfsdk.Config{
				Schema: sch.Schema,
				Raw: stateFromValues(t, ctx, sch, map[string]tftypes.Value{
					"v4_network": str(tc.v4),
					"v6_network": str(tc.v6),
				}).Raw,
			}
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// lxc.bridge_choices advertises "[AUTO]", so a user copying a value out of
// it writes that literally. TrueNAS stores it as null, which reads back as
// "", and the framework forbids a provider rewriting a value the config set
// explicitly. Rejecting it at validate time with the working spelling is
// the only handling that converges.
func TestLXCConfigResource_ValidateConfigRejectsAutoSentinel(t *testing.T) {
	ctx := context.Background()
	r := &LXCConfigResource{}
	sch := schemaOf(t, ctx, r)

	for _, tc := range []struct {
		bridge  string
		wantErr bool
	}{
		{lxcBridgeAuto, true},
		{"", false},
		{"br0", false},
	} {
		t.Run("bridge "+tc.bridge, func(t *testing.T) {
			cfg := tfsdk.Config{
				Schema: sch.Schema,
				Raw: stateFromValues(t, ctx, sch, map[string]tftypes.Value{
					"bridge": str(tc.bridge),
				}).Raw,
			}
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("HasError = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
			// The message has to name the spelling that works, or the user
			// is left guessing what to write instead.
			if tc.wantErr && !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `bridge = ""`) {
				t.Errorf("diagnostic does not name the working spelling: %s",
					resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

// The singleton has exactly one valid import ID. Anything else would put
// state on a resource that does not exist.
func TestLXCConfigResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &LXCConfigResource{client: lxcClient(t, lxcConfigBody())}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{})

	ok := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: lxcConfigSingletonID}, ok)
	if ok.Diagnostics.HasError() {
		t.Errorf("the canonical import ID was rejected: %v", ok.Diagnostics)
	}

	bad := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("ImportState accepted an ID other than the singleton's")
	}
}

// failingLXCClient answers every call with an error, so the CRUD handlers
// take their diagnostic branches instead of the happy path.
func failingLXCClient(t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

// A failed write or refresh has to surface as a diagnostic. Swallowing it
// would leave state claiming a configuration the server never accepted.
func TestLXCConfigResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &LXCConfigResource{client: failingLXCClient(t)}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"id":             str(lxcConfigSingletonID),
		"preferred_pool": str("tank"),
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

// The write succeeds but the read-back fails. State must not be written
// from the plan alone, or it records values the server may have coerced.
func TestLXCConfigResource_refreshFailureAfterWriteSurfaces(t *testing.T) {
	ctx := context.Background()
	writes := 0
	c := newWSTestClient(ctx, t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			if method == "lxc.update" {
				writes++
				return lxcConfigBody(), nil
			}
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EFAULT] read failed"}
		})
	r := &LXCConfigResource{client: c}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"id":             str(lxcConfigSingletonID),
		"preferred_pool": str("tank"),
	}
	st := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)

	cResp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create reported success when the read-back failed")
	}

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update reported success when the read-back failed")
	}
	if writes != 2 {
		t.Errorf("expected one write each from Create and Update, got %d", writes)
	}
}

// Both networks are settable, and both have to reach the wire when set.
func TestLXCConfigResource_applyConfigSendsNetworks(t *testing.T) {
	var sent map[string]interface{}
	c := newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			if method == "lxc.update" && len(params) > 0 {
				sent, _ = params[0].(map[string]interface{})
			}
			return lxcConfigBody(), nil
		})
	r := &LXCConfigResource{client: c}
	if err := r.applyConfig(context.Background(), &LXCConfigResourceModel{
		V4Network: types.StringValue("172.201.0.0/24"),
		V6Network: types.StringValue("fd42:1::/64"),
	}); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if sent["v4_network"] != "172.201.0.0/24" {
		t.Errorf("v4_network = %v", sent["v4_network"])
	}
	if sent["v6_network"] != "fd42:1::/64" {
		t.Errorf("v6_network = %v", sent["v6_network"])
	}
}

// A configured bridge has to round-trip, not just a null one.
func TestLXCConfigResource_refreshStateNamedBridge(t *testing.T) {
	body := lxcConfigBody()
	body["bridge"] = "br0"
	r := &LXCConfigResource{client: lxcClient(t, body)}
	var m LXCConfigResourceModel
	if err := r.refreshState(context.Background(), &m); err != nil {
		t.Fatalf("refreshState: %v", err)
	}
	if m.Bridge.ValueString() != "br0" {
		t.Errorf("bridge = %q, want br0", m.Bridge.ValueString())
	}
}

// A plan or config that fails to decode must stop the handler, not be
// applied half-read.
func TestLXCConfigResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &LXCConfigResource{client: failingLXCClient(t)}
	sch := schemaOf(t, ctx, r)
	// A raw value of the wrong shape for the schema: Get returns an error.
	bogus := tftypes.NewValue(tftypes.String, "not-an-object")

	vResp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: sch.Schema, Raw: bogus},
	}, vResp)
	if !vResp.Diagnostics.HasError() {
		t.Error("ValidateConfig accepted a config it could not decode")
	}

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
