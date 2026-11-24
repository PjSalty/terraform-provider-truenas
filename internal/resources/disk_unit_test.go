package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

const diskTestID = "{serial_lunid}WD-X1_5000"

// diskBody is what disk.get_instance returns for a disk in a pool.
func diskBody() map[string]interface{} {
	return map[string]interface{}{
		"identifier":   diskTestID,
		"name":         "sda",
		"serial":       "WD-X1",
		"model":        "WDC WD40",
		"size":         4000787030016,
		"type":         "HDD",
		"pool":         "tank",
		"description":  "bay 3",
		"hddstandby":   "ALWAYS ON",
		"advpowermgmt": "DISABLED",
	}
}

// diskCalls records what the resource sent: method names in order, and the
// disk.update body.
type diskCalls struct {
	methods []string
	update  map[string]interface{}
}

// diskClient serves body for disk.get_instance and accepts disk.update. Any
// other method is a test failure: the resource must never call one.
func diskClient(t *testing.T, body map[string]interface{}) (*wsclient.Client, *diskCalls) {
	t.Helper()
	calls := &diskCalls{}
	c := newWSTestClient(context.Background(), t,
		func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			calls.methods = append(calls.methods, method)
			switch method {
			case "disk.get_instance":
				return body, nil
			case "disk.update":
				calls.update, _ = params[1].(map[string]interface{})
				return body, nil
			}
			t.Errorf("unexpected API call %q", method)
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
		})
	return c, calls
}

// diskErrorClient answers every call with the given error.
func diskErrorClient(ctx context.Context, t *testing.T, rpcErr *wsclient.RPCError) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, rpcErr
		})
}

var (
	diskNotFound = &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Data: json.RawMessage(`{"errname":"ENOENT"}`), Message: "disk does not exist"}
	diskRejected = &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
)

func diskUnknown() tftypes.Value { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }

func TestDiskResource_CRUD(t *testing.T) {
	c, _ := diskClient(t, diskBody())
	r := &DiskResource{client: c}
	crudDrive(t, r, c, diskTestID, map[string]tftypes.Value{
		"identifier": str(diskTestID),
		"hddstandby": str("60"),
	})
}

// State has to hold what TrueNAS returned, not what the plan asked for, and
// only the settings the plan set may be sent.
func TestDiskResource_CreateAdoptsAndSendsOnlySetFields(t *testing.T) {
	ctx := context.Background()
	body := diskBody()
	body["hddstandby"] = "60"
	c, calls := diskClient(t, body)
	r := &DiskResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"identifier":   str(diskTestID),
		"hddstandby":   str("60"),
		"advpowermgmt": diskUnknown(),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	if len(calls.update) != 1 || calls.update["hddstandby"] != "60" {
		t.Errorf("update body = %#v, want only hddstandby", calls.update)
	}
	var got DiskResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if got.ID.ValueString() != diskTestID || got.Identifier.ValueString() != diskTestID {
		t.Errorf("id = %q, identifier = %q", got.ID.ValueString(), got.Identifier.ValueString())
	}
	if got.HDDStandby.ValueString() != "60" || got.AdvPowerMgmt.ValueString() != "DISABLED" || got.Description.ValueString() != "bay 3" {
		t.Errorf("settings = %+v", got)
	}
	if got.Name.ValueString() != "sda" || got.Serial.ValueString() != "WD-X1" || got.Model.ValueString() != "WDC WD40" ||
		got.Size.ValueInt64() != 4000787030016 || got.Type.ValueString() != "HDD" || got.Pool.ValueString() != "tank" {
		t.Errorf("read-only attributes = %+v", got)
	}
}

// A configuration that sets only the identifier adopts the disk without
// writing to it.
func TestDiskResource_CreateWithNoSettingsWritesNothing(t *testing.T) {
	ctx := context.Background()
	body := diskBody()
	body["pool"] = nil
	c, calls := diskClient(t, body)
	r := &DiskResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"identifier": str(diskTestID)})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if calls.update != nil {
		t.Errorf("disk.update was called with %#v", calls.update)
	}
	var got DiskResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if !got.Pool.IsNull() {
		t.Errorf("pool = %v, want null for a disk in no pool", got.Pool)
	}
}

// An empty description is a value to write, not an unset one.
func TestDiskResource_updateRequest(t *testing.T) {
	r := &DiskResource{}

	req := r.updateRequest(&DiskResourceModel{
		HDDStandby:   types.StringValue("5"),
		AdvPowerMgmt: types.StringValue("127"),
		Description:  types.StringValue(""),
	})
	if req.HDDStandby == nil || *req.HDDStandby != "5" ||
		req.AdvPowerMgmt == nil || *req.AdvPowerMgmt != "127" ||
		req.Description == nil || *req.Description != "" {
		t.Errorf("request = %+v", req)
	}

	req = r.updateRequest(&DiskResourceModel{
		HDDStandby:   types.StringNull(),
		AdvPowerMgmt: types.StringUnknown(),
		Description:  types.StringUnknown(),
	})
	if req.HDDStandby != nil || req.AdvPowerMgmt != nil || req.Description != nil {
		t.Errorf("unset attributes were sent: %+v", req)
	}
}

func TestDiskResource_UpdateSendsOnlyChangedSettings(t *testing.T) {
	ctx := context.Background()
	c, calls := diskClient(t, diskBody())
	r := &DiskResource{client: c}
	sch := schemaOf(t, ctx, r)

	vals := map[string]tftypes.Value{
		"id":          str(diskTestID),
		"identifier":  str(diskTestID),
		"description": str("bay 3"),
		"hddstandby":  diskUnknown(),
	}
	st := stateFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if len(calls.update) != 1 || calls.update["description"] != "bay 3" {
		t.Errorf("update body = %#v, want only description", calls.update)
	}
}

// Create adopts a disk and cannot make one.
func TestDiskResource_CreateUnknownDiskIsADiagnostic(t *testing.T) {
	ctx := context.Background()
	r := &DiskResource{client: diskErrorClient(ctx, t, diskNotFound)}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"identifier": str("{serial}nope")})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Create accepted an identifier no disk has")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Disk Not Found" {
		t.Errorf("summary = %q, want Disk Not Found", got)
	}
}

// A disk that is gone must drop out of state, not fail the refresh.
func TestDiskResource_ReadRemovesMissingDisk(t *testing.T) {
	ctx := context.Background()
	r := &DiskResource{client: diskErrorClient(ctx, t, diskNotFound)}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str(diskTestID), "identifier": str(diskTestID),
	})

	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a disk TrueNAS no longer has stayed in state")
	}
}

func TestDiskResource_ReadRefreshesState(t *testing.T) {
	ctx := context.Background()
	body := diskBody()
	body["name"] = "sdb" // the device name moved after a reboot
	c, _ := diskClient(t, body)
	r := &DiskResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str(diskTestID), "identifier": str(diskTestID), "name": str("sda"),
	})

	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var got DiskResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if got.Name.ValueString() != "sdb" {
		t.Errorf("name = %q, want the refreshed sdb", got.Name.ValueString())
	}
}

// A failed read or write has to surface, not leave state claiming settings
// the server never accepted.
func TestDiskResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	vals := map[string]tftypes.Value{
		"id": str(diskTestID), "identifier": str(diskTestID), "hddstandby": str("60"),
	}

	t.Run("every call fails", func(t *testing.T) {
		r := &DiskResource{client: diskErrorClient(ctx, t, diskRejected)}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, vals)
		plan := planFromValues(t, ctx, sch, vals)

		cResp := &resource.CreateResponse{State: st}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		if !cResp.Diagnostics.HasError() {
			t.Error("Create reported success against a server that failed the read")
		}
		rResp := &resource.ReadResponse{State: st}
		r.Read(ctx, resource.ReadRequest{State: st}, rResp)
		if !rResp.Diagnostics.HasError() {
			t.Error("Read reported success against a server that failed")
		}
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("Update reported success against a server that rejected the write")
		}
	})

	t.Run("the write fails but the read works", func(t *testing.T) {
		c := newWSTestClient(ctx, t,
			func(_ context.Context, method string, _ []interface{}) (interface{}, *wsclient.RPCError) {
				if method == "disk.update" {
					return nil, diskRejected
				}
				return diskBody(), nil
			})
		r := &DiskResource{client: c}
		sch := schemaOf(t, ctx, r)
		plan := planFromValues(t, ctx, sch, vals)

		resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Create reported success against a server that rejected the write")
		}
	})

	t.Run("the read after the write fails", func(t *testing.T) {
		reads := 0
		c := newWSTestClient(ctx, t,
			func(_ context.Context, method string, _ []interface{}) (interface{}, *wsclient.RPCError) {
				if method == "disk.get_instance" {
					reads++
					if reads > 1 {
						return nil, diskRejected
					}
				}
				return diskBody(), nil
			})
		r := &DiskResource{client: c}
		sch := schemaOf(t, ctx, r)
		plan := planFromValues(t, ctx, sch, vals)

		resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Create reported success although the disk could not be read back")
		}
	})
}

// Removing the resource must not touch the server: no write of any kind,
// and above all nothing that wipes or reconfigures a drive.
func TestDiskResource_DeleteSendsNothing(t *testing.T) {
	ctx := context.Background()
	c, calls := diskClient(t, diskBody())
	r := &DiskResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str(diskTestID), "identifier": str(diskTestID), "hddstandby": str("60"),
	})

	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", resp.Diagnostics)
	}
	if len(calls.methods) != 0 {
		t.Errorf("Delete made API calls: %v", calls.methods)
	}
}

// Both the id and the identifier come from the import ID, so the first
// Read can find the disk.
func TestDiskResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &DiskResource{}
	sch := schemaOf(t, ctx, r)

	resp := &resource.ImportStateResponse{State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{})}
	r.ImportState(ctx, resource.ImportStateRequest{ID: diskTestID}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "identifier"} {
		var got types.String
		if diags := resp.State.GetAttribute(ctx, path.Root(name), &got); diags.HasError() || got.ValueString() != diskTestID {
			t.Errorf("%s = %v (%v)", name, got, diags)
		}
	}
}

// A plan or state that fails to decode must stop the handler.
func TestDiskResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &DiskResource{client: diskErrorClient(ctx, t, diskRejected)}
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

// The enumerations are the API model's literals. TrueNAS reads them back
// upper-cased, so a different spelling would plan a change on every run, and
// a value outside the list is rejected by the server.
func TestDiskResource_EnumValidators(t *testing.T) {
	ctx := context.Background()
	attrs := schemaOf(t, ctx, &DiskResource{}).Schema.Attributes

	cases := map[string]struct {
		valid   []string
		invalid []string
	}{
		"hddstandby": {
			valid:   []string{"ALWAYS ON", "5", "10", "20", "30", "60", "120", "180", "240", "300", "330"},
			invalid: []string{"", "0", "15", "360", "Always On", "always on", "NEVER"},
		},
		"advpowermgmt": {
			valid:   []string{"DISABLED", "1", "64", "127", "128", "192", "254"},
			invalid: []string{"", "0", "2", "255", "Disabled", "disabled"},
		},
	}
	for name, tc := range cases {
		attr, ok := attrs[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is %T, want a string attribute", name, attrs[name])
		}
		if len(attr.Validators) == 0 {
			t.Fatalf("%s has no validators", name)
		}
		check := func(value string) bool {
			resp := &validator.StringResponse{}
			for _, v := range attr.Validators {
				v.ValidateString(ctx, validator.StringRequest{
					Path:        path.Root(name),
					ConfigValue: types.StringValue(value),
				}, resp)
			}
			return resp.Diagnostics.HasError()
		}
		for _, v := range tc.valid {
			if check(v) {
				t.Errorf("%s: %q was rejected", name, v)
			}
		}
		for _, v := range tc.invalid {
			if !check(v) {
				t.Errorf("%s: %q was accepted", name, v)
			}
		}
	}
}

func TestDiskResource_IdentifierRejectsEmpty(t *testing.T) {
	ctx := context.Background()
	attr, ok := schemaOf(t, ctx, &DiskResource{}).Schema.Attributes["identifier"].(schema.StringAttribute)
	if !ok || len(attr.Validators) == 0 {
		t.Fatal("identifier has no validators")
	}
	resp := &validator.StringResponse{}
	for _, v := range attr.Validators {
		v.ValidateString(ctx, validator.StringRequest{Path: path.Root("identifier"), ConfigValue: types.StringValue("")}, resp)
	}
	if !resp.Diagnostics.HasError() {
		t.Error("an empty identifier was accepted")
	}
}
