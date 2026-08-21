package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func webshareBody() map[string]interface{} {
	return map[string]interface{}{
		"id": 1, "name": "docs", "path": "/mnt/tank/docs",
		"enabled": true, "is_home_base": false,
		"dataset": "tank/docs", "relative_path": "", "locked": false,
	}
}

func TestWebshareResource_CRUD(t *testing.T) {
	c := newWSJSONServerClient(t, webshareBody())
	r := &WebshareResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name": str("docs"),
		"path": str("/mnt/tank/docs"),
	})
}

// The derived fields are nullable upstream. They must come back as known
// values, or every plan shows them as "(known after apply)".
func TestWebshareResource_nullDerivedFieldsReadAsKnown(t *testing.T) {
	r := &WebshareResource{}
	var m WebshareResourceModel
	r.mapResponseToModel(&truenas.Webshare{
		ID: 3, Name: "n", Path: "/p",
		Dataset: nil, RelativePath: nil, Locked: nil,
	}, &m)
	for name, v := range map[string]bool{
		"dataset":       m.Dataset.IsNull() || m.Dataset.IsUnknown(),
		"relative_path": m.RelativePath.IsNull() || m.RelativePath.IsUnknown(),
		"locked":        m.Locked.IsNull() || m.Locked.IsUnknown(),
	} {
		if v {
			t.Errorf("%s read back null/unknown; plans would show a phantom diff", name)
		}
	}
	if m.Locked.ValueBool() {
		t.Error("a null locked must read as false")
	}

	// And a populated response round-trips.
	ds, rp, lk := "tank/docs", "sub", true
	var m2 WebshareResourceModel
	r.mapResponseToModel(&truenas.Webshare{
		ID: 4, Name: "n", Path: "/p",
		Dataset: &ds, RelativePath: &rp, Locked: &lk,
	}, &m2)
	if m2.Dataset.ValueString() != "tank/docs" || m2.RelativePath.ValueString() != "sub" || !m2.Locked.ValueBool() {
		t.Errorf("derived fields did not round-trip: %+v", m2)
	}
}

// A non-numeric ID must be rejected everywhere it is parsed, rather than
// silently becoming 0 and operating on the wrong share.
func TestWebshareResource_nonNumericIDRejected(t *testing.T) {
	ctx := context.Background()
	c := newWSJSONServerClient(t, webshareBody())
	r := &WebshareResource{client: c}
	sch := schemaOf(t, ctx, r)
	bad := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("not-a-number"), "name": str("n"), "path": str("/p"),
	})

	rResp := &resource.ReadResponse{State: bad}
	r.Read(ctx, resource.ReadRequest{State: bad}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted a non-numeric ID")
	}

	badPlan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("not-a-number"), "name": str("n"), "path": str("/p"),
	})
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
}

// A share deleted out of band drops from state so the next plan recreates
// it, instead of failing every run.
func TestWebshareResource_readNotFoundRemovesFromState(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[ENOENT] does not exist"}
	})
	r := &WebshareResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("n"), "path": str("/p"),
	})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a share deleted out of band was not removed from state")
	}
}

// Deleting something already gone is success: the desired end state holds.
func TestWebshareResource_deleteAlreadyGoneIsSuccess(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[ENOENT] does not exist"}
	})
	r := &WebshareResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("n"), "path": str("/p"),
	})
	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("deleting an already-absent share errored: %v", resp.Diagnostics)
	}
}

// The version diagnostic has to survive all the way to the operator.
func TestWebshareResource_preV26ErrorSurfaces(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &WebshareResource{client: c}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name": str("docs"), "path": str("/mnt/tank/docs"),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("create succeeded against a server with no webshare namespace")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "26.0") {
		t.Errorf("diagnostic should name the required version: %s", resp.Diagnostics.Errors()[0].Detail())
	}
}

func TestWebshareResource_Configure(t *testing.T) {
	ctx := context.Background()
	r := &WebshareResource{}

	// Nil provider data is a no-op, not an error.
	resp := &resource.ConfigureResponse{}
	r.Configure(ctx, resource.ConfigureRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("nil ProviderData errored: %v", resp.Diagnostics)
	}

	// The wrong type must be rejected loudly.
	resp2 := &resource.ConfigureResponse{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: "not a client"}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("a wrongly-typed ProviderData was accepted")
	}
}

func TestWebshareResource_ConfigureSuccessAndImport(t *testing.T) {
	ctx := context.Background()
	c := newWSJSONServerClient(t, webshareBody())

	r := &WebshareResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure with a real client errored: %v", resp.Diagnostics)
	}
	if r.client == nil {
		t.Error("Configure did not store the client")
	}

	sch := schemaOf(t, ctx, r)
	iResp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, sch)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, iResp)
	if iResp.Diagnostics.HasError() {
		t.Errorf("importing a numeric ID failed: %v", iResp.Diagnostics)
	}
}

// A real API failure must surface, distinct from not-found which is
// handled by dropping or accepting.
func TestWebshareResource_apiErrorsSurface(t *testing.T) {
	ctx := context.Background()
	mk := func() *wsclient.Client {
		return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
		})
	}
	r := &WebshareResource{client: mk()}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("n"), "path": str("/p"),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("n2"), "path": str("/p"),
	})

	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read swallowed an internal error")
	}
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update swallowed an internal error")
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete swallowed an internal error")
	}
}

// An undecodable plan or state must abort with diagnostics rather than
// operating on a zero-valued model.
func TestWebshareResource_undecodableStateAborts(t *testing.T) {
	ctx := context.Background()
	r := &WebshareResource{client: newWSJSONServerClient(t, webshareBody())}
	sch := schemaOf(t, ctx, r)

	bogus := bogusSchemaState(t, ctx)

	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: bogusPlan(t, ctx)}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create accepted an undecodable plan")
	}

	rResp := &resource.ReadResponse{State: bogus}
	r.Read(ctx, resource.ReadRequest{State: bogus}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted an undecodable state")
	}

	uResp := &resource.UpdateResponse{State: bogus}
	r.Update(ctx, resource.UpdateRequest{State: bogus, Plan: bogusPlan(t, ctx)}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted an undecodable plan")
	}

	dResp := &resource.DeleteResponse{State: bogus}
	r.Delete(ctx, resource.DeleteRequest{State: bogus}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete accepted an undecodable state")
	}
}

// bogusSchemaState / bogusPlan build a State and Plan whose schema does not
// match WebshareResourceModel, so Get fails and the CRUD guards fire.
func bogusSchemaObject(t *testing.T, ctx context.Context) (fwschema.Schema, tftypes.Value) {
	t.Helper()
	s := fwschema.Schema{
		Attributes: map[string]fwschema.Attribute{
			"id": fwschema.BoolAttribute{Computed: true},
		},
	}
	raw := tftypes.NewValue(
		s.Type().TerraformType(ctx).(tftypes.Object),
		map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.Bool, true)},
	)
	return s, raw
}

func bogusSchemaState(t *testing.T, ctx context.Context) tfsdk.State {
	t.Helper()
	s, raw := bogusSchemaObject(t, ctx)
	return tfsdk.State{Schema: s, Raw: raw}
}

func bogusPlan(t *testing.T, ctx context.Context) tfsdk.Plan {
	t.Helper()
	s, raw := bogusSchemaObject(t, ctx)
	return tfsdk.Plan{Schema: s, Raw: raw}
}
