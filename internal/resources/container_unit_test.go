package resources

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func containerBody() map[string]interface{} {
	return map[string]interface{}{
		"id": 3, "uuid": "1b4e28ba-2fa1-11d2-883f-0016d3cca427", "name": "web",
		"description": "front end", "cpuset": nil, "autostart": true,
		"time": "LOCAL", "shutdown_timeout": 90, "dataset": "test/containers/web",
		"init": "/sbin/init", "initdir": nil, "initenv": map[string]interface{}{"TZ": "UTC"},
		"inituser": nil, "initgroup": nil,
		"idmap":               map[string]interface{}{"type": "DEFAULT"},
		"capabilities_policy": "DEFAULT",
		"capabilities_state":  map[string]interface{}{},
		"default_network":     "truenasbr0",
		"status":              map[string]interface{}{"state": "STOPPED", "pid": nil, "domain_state": nil},
	}
}

// containerClient answers reads directly and job-backed writes through a
// job round trip, matching how container.create/delete/stop behave.
func containerClient(t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	const jobID = int64(21)
	return newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			switch method {
			case "container.create", "container.delete", "container.stop":
				return jobID, nil
			case "core.get_jobs":
				return []interface{}{map[string]interface{}{
					"id": jobID, "state": "SUCCESS", "result": body,
				}}, nil
			case "container.pool_choices":
				return map[string]interface{}{"tank": "tank"}, nil
			}
			return body, nil
		})
}

func containerImageObject(name, version string) types.Object {
	o, _ := types.ObjectValue(containerImageAttrTypes, map[string]attr.Value{
		"name":    types.StringValue(name),
		"version": types.StringValue(version),
	})
	return o
}

func containerIdmapObject(t string, slice attr.Value) types.Object {
	o, _ := types.ObjectValue(containerIdmapAttrTypes, map[string]attr.Value{
		"type":  types.StringValue(t),
		"slice": slice,
	})
	return o
}

func TestContainerResource_CRUD(t *testing.T) {
	c := containerClient(t, containerBody())
	r := &ContainerResource{client: c}
	crudDrive(t, r, c, "3", map[string]tftypes.Value{
		"name": str("web"),
	})
}

// Every nullable upstream field must refresh as a known value. Reading one
// back as null makes every plan show "(known after apply)".
func TestContainerResource_nullFieldsReadAsKnown(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	var m ContainerResourceModel
	r.mapResponseToModel(ctx, &truenas.Container{
		ID: 3, Name: "web", Status: truenas.ContainerStatus{State: "STOPPED"},
	}, &m)

	for name, v := range map[string]types.String{
		"cpuset":          m.Cpuset,
		"initdir":         m.InitDir,
		"inituser":        m.InitUser,
		"initgroup":       m.InitGroup,
		"default_network": m.DefaultNetwork,
	} {
		if v.IsNull() || v.IsUnknown() {
			t.Errorf("%s read back null/unknown; plans would show a phantom diff", name)
		}
	}
	// A nil map must become an empty map, not null.
	if m.InitEnv.IsNull() || len(m.InitEnv.Elements()) != 0 {
		t.Errorf("initenv = %v, want a known empty map", m.InitEnv)
	}
	if m.CapabilitiesState.IsNull() || len(m.CapabilitiesState.Elements()) != 0 {
		t.Errorf("capabilities_state = %v, want a known empty map", m.CapabilitiesState)
	}
	// A missing PID reads as 0 rather than null, so status stays fully known.
	statusAttrs := m.Status.Attributes()
	if pid, _ := statusAttrs["pid"].(types.Int64); pid.IsNull() || pid.ValueInt64() != 0 {
		t.Errorf("status.pid = %v, want a known 0", statusAttrs["pid"])
	}
}

// A null idmap upstream means the container has NO user namespace at all:
// container root is host root. Rendering it as DEFAULT would report the
// container as safer than it is, so it stays null and shows in the plan.
func TestContainerResource_nullIdmapIsNotReportedAsDefault(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	var m ContainerResourceModel
	r.mapResponseToModel(ctx, &truenas.Container{ID: 3, Name: "web", Idmap: nil}, &m)
	if !m.Idmap.IsNull() {
		t.Fatalf("an unmapped container reported idmap = %v; that reads as safer than it is", m.Idmap)
	}

	slice := 7
	var m2 ContainerResourceModel
	r.mapResponseToModel(ctx, &truenas.Container{
		ID: 3, Name: "web",
		Idmap: &truenas.ContainerIdmap{Type: containerIdmapIsolated, Slice: &slice},
	}, &m2)
	attrs := m2.Idmap.Attributes()
	if got, _ := attrs["type"].(types.String); got.ValueString() != containerIdmapIsolated {
		t.Errorf("idmap.type = %v", attrs["type"])
	}
	if got, _ := attrs["slice"].(types.Int64); got.ValueInt64() != 7 {
		t.Errorf("idmap.slice = %v", attrs["slice"])
	}
}

func TestContainerResource_buildCreateRequest(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}

	t.Run("omits what the plan did not set", func(t *testing.T) {
		req, err := r.buildCreateRequest(ctx, &ContainerResourceModel{
			Name:  types.StringValue("web"),
			Image: containerImageObject("alpine", "v1"),
			// uuid and pool are "let TrueNAS decide" when empty, so an
			// empty string must not be sent as a real value.
			UUID:  types.StringValue(""),
			Pool:  types.StringValue(""),
			Idmap: types.ObjectNull(containerIdmapAttrTypes),
		})
		if err != nil {
			t.Fatalf("buildCreateRequest: %v", err)
		}
		if req.UUID != nil || req.Pool != nil {
			t.Errorf("an empty uuid/pool was sent: uuid=%v pool=%v", req.UUID, req.Pool)
		}
		if req.Idmap != nil {
			t.Errorf("idmap sent for a plan that did not set it: %+v", req.Idmap)
		}
		if req.Image.Name != "alpine" || req.Image.Version != "v1" {
			t.Errorf("image = %+v", req.Image)
		}
	})

	t.Run("sends what the plan did set", func(t *testing.T) {
		env, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{"TZ": "UTC"})
		caps, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"sys_time": true})
		req, err := r.buildCreateRequest(ctx, &ContainerResourceModel{
			Name:               types.StringValue("web"),
			Image:              containerImageObject("alpine", "v1"),
			UUID:               types.StringValue("uuid-1"),
			Pool:               types.StringValue("tank"),
			Description:        types.StringValue("d"),
			Autostart:          types.BoolValue(false),
			ShutdownTimeout:    types.Int64Value(30),
			InitEnv:            env,
			CapabilitiesState:  caps,
			CapabilitiesPolicy: types.StringValue("DENY"),
			Idmap:              containerIdmapObject(containerIdmapIsolated, types.Int64Value(4)),
		})
		if err != nil {
			t.Fatalf("buildCreateRequest: %v", err)
		}
		if req.UUID == nil || *req.UUID != "uuid-1" || req.Pool == nil || *req.Pool != "tank" {
			t.Errorf("uuid/pool not sent: %+v", req)
		}
		if req.Autostart == nil || *req.Autostart {
			t.Errorf("autostart = %v, want false", req.Autostart)
		}
		if req.ShutdownTimeout == nil || *req.ShutdownTimeout != 30 {
			t.Errorf("shutdown_timeout = %v", req.ShutdownTimeout)
		}
		if req.InitEnv == nil || (*req.InitEnv)["TZ"] != "UTC" {
			t.Errorf("initenv = %v", req.InitEnv)
		}
		if req.CapabilitiesState == nil || !(*req.CapabilitiesState)["sys_time"] {
			t.Errorf("capabilities_state = %v", req.CapabilitiesState)
		}
		if req.Idmap == nil || req.Idmap.Slice == nil || *req.Idmap.Slice != 4 {
			t.Errorf("idmap = %+v", req.Idmap)
		}
	})

	// An explicitly empty map means "clear this", and it has to reach the
	// wire as {}. The request fields are `omitempty`, and encoding/json
	// omits an EMPTY map as readily as a nil one, so if these were plain
	// maps rather than pointers `initenv = {}` would be indistinguishable
	// from "not set" and the old environment would silently survive. This
	// asserts the encoded JSON, not just the Go value, because that is
	// where the distinction actually bites.
	t.Run("an explicitly empty map is sent, not omitted", func(t *testing.T) {
		empty, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
		emptyBools, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{})
		req, err := r.buildCreateRequest(ctx, &ContainerResourceModel{
			Name: types.StringValue("web"), Image: containerImageObject("a", "v"),
			InitEnv: empty, CapabilitiesState: emptyBools,
		})
		if err != nil {
			t.Fatalf("buildCreateRequest: %v", err)
		}
		encoded, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		for _, k := range []string{"initenv", "capabilities_state"} {
			raw, present := body[k]
			if !present {
				t.Errorf("%s was omitted; clearing it would silently do nothing: %s", k, encoded)
				continue
			}
			if string(raw) != "{}" {
				t.Errorf("%s encoded as %s, want {}", k, raw)
			}
		}
	})
}

// pool, image and idmap are excluded from the upstream update model, so
// they must never appear in an update body no matter what the plan holds.
func TestContainerResource_buildUpdateRequestHasNoCreateOnlyFields(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	req, err := r.buildUpdateRequest(ctx, &ContainerResourceModel{
		Name:  types.StringValue("web"),
		Pool:  types.StringValue("tank"),
		Image: containerImageObject("alpine", "v1"),
		Idmap: containerIdmapObject(containerIdmapDefault, types.Int64Null()),
	})
	if err != nil {
		t.Fatalf("buildUpdateRequest: %v", err)
	}
	// The struct has no such fields at all, which is the point: this
	// asserts the shape by round-tripping what it does carry.
	if req.Name == nil || *req.Name != "web" {
		t.Errorf("name = %v", req.Name)
	}
}

func TestContainerResource_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	sch := schemaOf(t, ctx, r)

	newCfg := func(vals map[string]tftypes.Value) tfsdk.Config {
		return tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, vals).Raw}
	}
	idmapVal := func(typ string, slice interface{}) tftypes.Value {
		return tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"type": tftypes.String, "slice": tftypes.Number,
		}}, map[string]tftypes.Value{
			"type":  tftypes.NewValue(tftypes.String, typ),
			"slice": tftypes.NewValue(tftypes.Number, slice),
		})
	}

	cases := []struct {
		name    string
		vals    map[string]tftypes.Value
		wantErr bool
	}{
		{"no idmap is fine", map[string]tftypes.Value{}, false},
		{"DEFAULT without a slice", map[string]tftypes.Value{"idmap": idmapVal(containerIdmapDefault, nil)}, false},
		{"DEFAULT with a slice is a config the server ignores", map[string]tftypes.Value{"idmap": idmapVal(containerIdmapDefault, 3)}, true},
		{"ISOLATED without a slice lets TrueNAS pick", map[string]tftypes.Value{"idmap": idmapVal(containerIdmapIsolated, nil)}, false},
		{"ISOLATED with a valid slice", map[string]tftypes.Value{"idmap": idmapVal(containerIdmapIsolated, 999)}, false},
		{"slice at the upper bound is rejected", map[string]tftypes.Value{"idmap": idmapVal(containerIdmapIsolated, 1000)}, true},
		{"slice below 1 is rejected", map[string]tftypes.Value{"idmap": idmapVal(containerIdmapIsolated, 0)}, true},
		{"shutdown_timeout in range", map[string]tftypes.Value{"shutdown_timeout": tftypes.NewValue(tftypes.Number, 300)}, false},
		{"shutdown_timeout too high", map[string]tftypes.Value{"shutdown_timeout": tftypes.NewValue(tftypes.Number, 301)}, true},
		{"shutdown_timeout too low", map[string]tftypes.Value{"shutdown_timeout": tftypes.NewValue(tftypes.Number, 4)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: newCfg(tc.vals)}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// image lives only on the upstream CREATE model, so an imported container
// has none in state. Without the ModifyPlan carve-out, Required +
// RequiresReplace turns that into destroy-and-recreate on the first apply
// after an import.
func TestContainerResource_importedContainerIsNotReplaced(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	sch := schemaOf(t, ctx, r)

	withImage := map[string]tftypes.Value{"id": str("3"), "name": str("web")}
	plan := planFromValues(t, ctx, sch, withImage)
	// State as it looks straight after import: everything read back except
	// image, which nothing can report.
	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("3"), "name": str("web")})

	// uuid is the control: unlike image and pool it IS on the upstream
	// entry, so an imported container has it in state and a change to it
	// must still replace.
	resp := &resource.ModifyPlanResponse{
		Plan:            plan,
		RequiresReplace: []path.Path{path.Root("image"), path.Root("pool"), path.Root("uuid")},
	}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)

	for _, p := range resp.RequiresReplace {
		if p.Equal(path.Root("image")) || p.Equal(path.Root("pool")) {
			t.Errorf("an imported container would be destroyed and recreated on its first apply (%s)", p)
		}
	}
	var keptUUID bool
	for _, p := range resp.RequiresReplace {
		if p.Equal(path.Root("uuid")) {
			keptUUID = true
		}
	}
	if !keptUUID {
		t.Error("the carve-out dropped a replace reason it has no business touching")
	}
}

// With an image already in state, a change to it must still replace.
func TestContainerResource_createOnlyChangeStillReplaces(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	sch := schemaOf(t, ctx, r)

	vals := map[string]tftypes.Value{
		"id": str("3"), "name": str("web"), "pool": str("tank"),
		"image": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"name": tftypes.String, "version": tftypes.String,
		}}, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, "alpine"), "version": tftypes.NewValue(tftypes.String, "v1"),
		}),
	}
	plan := planFromValues(t, ctx, sch, vals)
	state := stateFromValues(t, ctx, sch, vals)

	resp := &resource.ModifyPlanResponse{Plan: plan, RequiresReplace: []path.Path{path.Root("image"), path.Root("pool")}}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	if len(resp.RequiresReplace) != 2 {
		t.Errorf("a real image or pool change stopped forcing a replace: %v", resp.RequiresReplace)
	}
}

// A destroy plan and a create plan both have a null side; ModifyPlan must
// not try to read the missing one.
func TestContainerResource_ModifyPlanNullSides(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	sch := schemaOf(t, ctx, r)
	nullPlan := tfsdk.Plan{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)}
	nullState := tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)}
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"name": str("web")})

	for _, tc := range []struct {
		name  string
		plan  tfsdk.Plan
		state tfsdk.State
	}{
		{"destroy", nullPlan, stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("3")})},
		{"create", plan, nullState},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ModifyPlanResponse{Plan: tc.plan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: tc.plan, State: tc.state}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("diagnostics on a %s plan: %v", tc.name, resp.Diagnostics)
			}
		})
	}
}

func TestContainerResource_validatePool(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{client: containerClient(t, containerBody())}

	if err := r.validatePool(ctx, "tank"); err != nil {
		t.Errorf("a pool the server offers was rejected: %v", err)
	}
	err := r.validatePool(ctx, "nope")
	if err == nil {
		t.Fatal("a pool that cannot host containers was accepted")
	}
	if !strings.Contains(err.Error(), "tank") {
		t.Errorf("diagnostic should list the usable pools, got: %v", err)
	}

	t.Run("no usable pools says so", func(t *testing.T) {
		c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
			return map[string]interface{}{}, nil
		})
		err := (&ContainerResource{client: c}).validatePool(ctx, "tank")
		if err == nil || !strings.Contains(err.Error(), "no usable pools") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("a lookup failure surfaces", func(t *testing.T) {
		c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EPERM] nope"}
		})
		err := (&ContainerResource{client: c}).validatePool(ctx, "tank")
		if err == nil || !strings.Contains(err.Error(), "pool choices") {
			t.Errorf("err = %v", err)
		}
	})
}

// failingContainerClient answers every call with an error so the CRUD
// handlers take their diagnostic branches.
func failingContainerClient(t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

func TestContainerResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{client: failingContainerClient(t)}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{"id": str("3"), "name": str("web")}
	st := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)

	cResp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create reported success against a server that rejected it")
	}

	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read reported success against a server that returned an error")
	}

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update reported success against a server that rejected it")
	}

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete reported success against a server that rejected it")
	}
}

// A pool the container service will not accept must fail before the image
// pull starts, not minutes later from inside the job.
func TestContainerResource_createRejectsBadPoolBeforeTheJob(t *testing.T) {
	ctx := context.Background()
	var created bool
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		if m == "container.create" {
			created = true
		}
		if m == "container.pool_choices" {
			return map[string]interface{}{"tank": "tank"}, nil
		}
		return containerBody(), nil
	})
	r := &ContainerResource{client: c}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name": str("web"), "pool": str("nope"),
	})
	resp := &resource.CreateResponse{State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{})}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a pool that cannot host containers was accepted")
	}
	if created {
		t.Error("the create job was started despite an invalid pool")
	}
}

// A container deleted out of band drops from state so the next plan
// recreates it, instead of failing every run.
func TestContainerResource_readNotFoundRemovesFromState(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[ENOENT] does not exist"}
	})
	r := &ContainerResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("3"), "name": str("web")})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a deleted container errored instead of dropping from state: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a container deleted out of band stayed in state")
	}
}

// Deleting something already gone is success: the desired end state is
// reached either way.
func TestContainerResource_deleteAlreadyGoneIsSuccess(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[ENOENT] does not exist"}
	})
	r := &ContainerResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("3")})
	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("deleting an already-absent container errored: %v", resp.Diagnostics)
	}
}

// A non-numeric ID must be rejected everywhere it is parsed, rather than
// silently becoming 0 and operating on the wrong container.
func TestContainerResource_nonNumericIDRejected(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{client: containerClient(t, containerBody())}
	sch := schemaOf(t, ctx, r)
	bad := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("nope"), "name": str("web")})
	badPlan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("nope"), "name": str("web")})

	rResp := &resource.ReadResponse{State: bad}
	r.Read(ctx, resource.ReadRequest{State: bad}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted a non-numeric ID")
	}
	uResp := &resource.UpdateResponse{State: bad}
	r.Update(ctx, resource.UpdateRequest{State: bad, Plan: badPlan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted a non-numeric ID")
	}
	dResp := &resource.DeleteResponse{State: bad}
	r.Delete(ctx, resource.DeleteRequest{State: bad}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete accepted a non-numeric ID")
	}
	iResp := &resource.ImportStateResponse{State: bad}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "nope"}, iResp)
	if !iResp.Diagnostics.HasError() {
		t.Error("ImportState accepted a non-numeric ID")
	}
	okResp := &resource.ImportStateResponse{State: bad}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "3"}, okResp)
	if okResp.Diagnostics.HasError() {
		t.Errorf("a numeric import ID was rejected: %v", okResp.Diagnostics)
	}
}

// A plan or config that fails to decode must stop the handler, not be
// applied half-read.
func TestContainerResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{client: failingContainerClient(t)}
	sch := schemaOf(t, ctx, r)
	bogus := tftypes.NewValue(tftypes.String, "not-an-object")

	vResp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sch.Schema, Raw: bogus}}, vResp)
	if !vResp.Diagnostics.HasError() {
		t.Error("ValidateConfig accepted a config it could not decode")
	}

	mResp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: bogus}}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  tfsdk.Plan{Schema: sch.Schema, Raw: bogus},
		State: tfsdk.State{Schema: sch.Schema, Raw: bogus},
	}, mResp)
	if !mResp.Diagnostics.HasError() {
		t.Error("ModifyPlan accepted a plan it could not decode")
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

	dResp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: sch.Schema, Raw: bogus}}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete accepted a state it could not decode")
	}
}

// The map helpers are the one place a schema-typed value can still fail to
// convert: an element that is unknown at apply time has no Go string to
// become. That must surface as an error rather than silently dropping the
// entry from the body.
func TestContainerResource_mapHelpers(t *testing.T) {
	ctx := context.Background()

	t.Run("unset stays unset", func(t *testing.T) {
		if got, err := containerStringMap(ctx, types.MapNull(types.StringType)); got != nil || err != nil {
			t.Errorf("got %v, %v", got, err)
		}
		if got, err := containerBoolMap(ctx, types.MapUnknown(types.BoolType)); got != nil || err != nil {
			t.Errorf("got %v, %v", got, err)
		}
	})

	// An empty map must survive as an allocated map: a non-nil pointer to a
	// nil map encodes as JSON null, which middleware rejects.
	t.Run("empty maps are allocated", func(t *testing.T) {
		s, err := containerStringMap(ctx, types.MapValueMust(types.StringType, map[string]attr.Value{}))
		if err != nil || s == nil || *s == nil {
			t.Errorf("string map: %v, %v", s, err)
		}
		b, err := containerBoolMap(ctx, types.MapValueMust(types.BoolType, map[string]attr.Value{}))
		if err != nil || b == nil || *b == nil {
			t.Errorf("bool map: %v, %v", b, err)
		}
	})

	t.Run("an unknown element is an error, not a silent drop", func(t *testing.T) {
		if _, err := containerStringMap(ctx, types.MapValueMust(types.StringType, map[string]attr.Value{
			"TZ": types.StringUnknown(),
		})); err == nil {
			t.Error("an unknown initenv element was accepted")
		}
		if _, err := containerBoolMap(ctx, types.MapValueMust(types.BoolType, map[string]attr.Value{
			"sys_time": types.BoolUnknown(),
		})); err == nil {
			t.Error("an unknown capabilities_state element was accepted")
		}
	})
}

// Both build paths must report a map they cannot convert instead of
// sending a body missing the entries.
func TestContainerResource_buildRequestsReportBadMaps(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	badStr := types.MapValueMust(types.StringType, map[string]attr.Value{"TZ": types.StringUnknown()})
	badBool := types.MapValueMust(types.BoolType, map[string]attr.Value{"sys_time": types.BoolUnknown()})

	for _, tc := range []struct {
		name  string
		model ContainerResourceModel
		want  string
	}{
		{"initenv", ContainerResourceModel{Name: types.StringValue("w"), Image: containerImageObject("a", "v"), InitEnv: badStr}, "initenv"},
		{"capabilities_state", ContainerResourceModel{Name: types.StringValue("w"), Image: containerImageObject("a", "v"), CapabilitiesState: badBool}, "capabilities_state"},
	} {
		t.Run("create/"+tc.name, func(t *testing.T) {
			m := tc.model
			if _, err := r.buildCreateRequest(ctx, &m); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v", err)
			}
		})
		t.Run("update/"+tc.name, func(t *testing.T) {
			m := tc.model
			if _, err := r.buildUpdateRequest(ctx, &m); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

// Create and Update must stop on a body they could not build, rather than
// sending a partial one.
func TestContainerResource_buildFailureStopsWrite(t *testing.T) {
	ctx := context.Background()
	var wrote bool
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		if m == "container.create" || m == "container.update" {
			wrote = true
		}
		return containerBody(), nil
	})
	r := &ContainerResource{client: c}
	sch := schemaOf(t, ctx, r)

	mapType := tftypes.Map{ElementType: tftypes.String}
	vals := map[string]tftypes.Value{
		"id": str("3"), "name": str("web"),
		"initenv": tftypes.NewValue(mapType, map[string]tftypes.Value{
			"TZ": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	plan := planFromValues(t, ctx, sch, vals)
	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("3"), "name": str("web")})

	cResp := &resource.CreateResponse{State: state}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create accepted a body it could not build")
	}

	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted a body it could not build")
	}
	if wrote {
		t.Error("a write went out despite the body failing to build")
	}
}

func TestContainerResource_buildUpdateRequestScalars(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	req, err := r.buildUpdateRequest(ctx, &ContainerResourceModel{
		Autostart:       types.BoolValue(false),
		ShutdownTimeout: types.Int64Value(45),
	})
	if err != nil {
		t.Fatalf("buildUpdateRequest: %v", err)
	}
	if req.Autostart == nil || *req.Autostart {
		t.Errorf("autostart = %v, want false", req.Autostart)
	}
	if req.ShutdownTimeout == nil || *req.ShutdownTimeout != 45 {
		t.Errorf("shutdown_timeout = %v", req.ShutdownTimeout)
	}
}

// A running container reports a PID; it must round-trip rather than being
// flattened to the 0 that a stopped container reports.
func TestContainerResource_runningStatusRoundTrips(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	pid, domain := 4242, "running"
	var m ContainerResourceModel
	r.mapResponseToModel(ctx, &truenas.Container{
		ID: 3, Name: "web",
		Status: truenas.ContainerStatus{State: "RUNNING", PID: &pid, DomainState: &domain},
	}, &m)
	attrs := m.Status.Attributes()
	if got, _ := attrs["pid"].(types.Int64); got.ValueInt64() != 4242 {
		t.Errorf("status.pid = %v", attrs["pid"])
	}
	if got, _ := attrs["domain_state"].(types.String); got.ValueString() != "running" {
		t.Errorf("status.domain_state = %v", attrs["domain_state"])
	}
}

func TestContainerImageUnset(t *testing.T) {
	if !containerImageUnset(types.ObjectNull(containerImageAttrTypes)) {
		t.Error("a null image object was treated as set")
	}
	if !containerImageUnset(types.ObjectUnknown(containerImageAttrTypes)) {
		t.Error("an unknown image object was treated as set")
	}
	if !containerImageUnset(containerImageObject("", "v1")) {
		t.Error("an image with no name was treated as set")
	}
	if containerImageUnset(containerImageObject("alpine", "v1")) {
		t.Error("a real image was treated as unset")
	}
}

// A config with no idmap block at all must validate: the block is optional
// and its absence means the TrueNAS default.
func TestContainerResource_ValidateConfigNullIdmap(t *testing.T) {
	ctx := context.Background()
	r := &ContainerResource{}
	sch := schemaOf(t, ctx, r)
	cfg := tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"idmap": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"type": tftypes.String, "slice": tftypes.Number,
		}}, nil),
	}).Raw}
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("an omitted idmap block was rejected: %v", resp.Diagnostics)
	}
}

// The single most destructive thing this resource can do is pass the API's
// recursive delete flag: it destroys the container dataset's child datasets
// and snapshots, any clones of those snapshots anywhere in the pool, and
// any holds on them, none of it recoverable. Nothing Terraform was asked to
// manage is in that set, so Delete must never send it.
//
// force IS sent, so a running container can be torn down at all.
func TestContainerResource_deleteNeverRecurses(t *testing.T) {
	ctx := context.Background()
	var opts map[string]interface{}
	const jobID = int64(31)
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		switch m {
		case "container.delete":
			if len(p) > 1 {
				opts, _ = p[1].(map[string]interface{})
			}
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		}
		return containerBody(), nil
	})
	r := &ContainerResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("3"), "name": str("web")})

	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", resp.Diagnostics)
	}
	if opts == nil {
		t.Fatal("container.delete was called without options")
	}
	if opts["recursive"] != false {
		t.Errorf("recursive = %v, want false: it destroys snapshots and clones the operator never asked to remove", opts["recursive"])
	}
	if opts["force"] != true {
		t.Errorf("force = %v, want true: without it a running container cannot be removed", opts["force"])
	}
}

// The derived attributes must carry UseStateForUnknown.
//
// Without it the framework replans every Computed attribute as unknown on
// any update, so `terraform plan` right after a successful apply reports a
// change with nothing behind it. That was found by an acceptance run
// against a live 26.0 box, which does not gate CI, so it is pinned here
// where a plain `go test` catches it.
//
// Refresh still rewrites all three from the server, so pinning the PLANNED
// value hides no real drift.
func TestContainerResource_derivedAttributesPinPlannedValues(t *testing.T) {
	ctx := context.Background()
	sch := schemaOf(t, ctx, &ContainerResource{})

	for _, name := range []string{"dataset", "default_network"} {
		t.Run(name, func(t *testing.T) {
			attr, ok := sch.Schema.Attributes[name].(fwschema.StringAttribute)
			if !ok {
				t.Fatalf("%s is not a StringAttribute", name)
			}
			var found bool
			for _, pm := range attr.PlanModifiers {
				if pm.Description(ctx) == stringplanmodifier.UseStateForUnknown().Description(ctx) {
					found = true
				}
			}
			if !found {
				t.Errorf("%s has no UseStateForUnknown; every plan after an apply would show a phantom diff", name)
			}
		})
	}

	t.Run("status", func(t *testing.T) {
		attr, ok := sch.Schema.Attributes["status"].(fwschema.SingleNestedAttribute)
		if !ok {
			t.Fatal("status is not a SingleNestedAttribute")
		}
		var found bool
		for _, pm := range attr.PlanModifiers {
			if pm.Description(ctx) == objectplanmodifier.UseStateForUnknown().Description(ctx) {
				found = true
			}
		}
		if !found {
			t.Error("status has no UseStateForUnknown; every plan after an apply would show a phantom diff")
		}
	})
}
