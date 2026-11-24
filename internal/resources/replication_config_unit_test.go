package resources

import (
	"context"
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

func replicationConfigBody(limit interface{}) map[string]interface{} {
	return map[string]interface{}{"id": 1, "max_parallel_replication_tasks": limit}
}

func replicationConfigClient(ctx context.Context, t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return body, nil
		})
}

// failingReplicationConfigClient answers every call with an error, so the
// CRUD handlers take their diagnostic branches.
func failingReplicationConfigClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

func TestReplicationConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	c := replicationConfigClient(ctx, t, replicationConfigBody(3))
	r := &ReplicationConfigResource{client: c}
	crudDrive(t, r, c, replicationConfigSingletonID, map[string]tftypes.Value{
		"max_parallel_replication_tasks": num(3),
	})
}

// State has to hold what TrueNAS returned, and a null limit has to read
// back as 0 so that a plan with `0` settles.
func TestReplicationConfigResource_StoresServerValues(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		body interface{}
		want int64
	}{
		"a limit":  {5, 5},
		"no limit": {nil, 0},
	} {
		t.Run(name, func(t *testing.T) {
			r := &ReplicationConfigResource{client: replicationConfigClient(ctx, t, replicationConfigBody(tc.body))}
			sch := schemaOf(t, ctx, r)

			plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{})
			resp := &resource.CreateResponse{State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{})}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Create: %v", resp.Diagnostics)
			}

			var got ReplicationConfigResourceModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("state: %v", diags)
			}
			if got.ID.ValueString() != replicationConfigSingletonID {
				t.Errorf("id = %q", got.ID.ValueString())
			}
			if got.MaxParallelReplicationTasks.IsNull() || got.MaxParallelReplicationTasks.ValueInt64() != tc.want {
				t.Errorf("limit = %v, want %d", got.MaxParallelReplicationTasks, tc.want)
			}
		})
	}
}

// updateRequest decides between leaving the key out, sending a value and
// sending null. 0 is how a configuration says "no limit".
func TestReplicationConfigResource_updateRequest(t *testing.T) {
	r := &ReplicationConfigResource{}

	for name, limit := range map[string]types.Int64{
		"null":    types.Int64Null(),
		"unknown": types.Int64Unknown(),
	} {
		t.Run(name+" is left out", func(t *testing.T) {
			req := r.updateRequest(&ReplicationConfigResourceModel{MaxParallelReplicationTasks: limit})
			if !req.MaxParallelReplicationTasks.IsZero() {
				t.Error("an unset limit was sent")
			}
		})
	}

	t.Run("a limit is sent", func(t *testing.T) {
		req := r.updateRequest(&ReplicationConfigResourceModel{MaxParallelReplicationTasks: types.Int64Value(3)})
		raw, err := req.MaxParallelReplicationTasks.MarshalJSON()
		if req.MaxParallelReplicationTasks.IsZero() || err != nil || string(raw) != "3" {
			t.Errorf("limit marshals to %s (%v)", raw, err)
		}
	})

	// Without this the limit could be set but never removed.
	t.Run("0 is sent as null", func(t *testing.T) {
		req := r.updateRequest(&ReplicationConfigResourceModel{MaxParallelReplicationTasks: types.Int64Value(0)})
		raw, err := req.MaxParallelReplicationTasks.MarshalJSON()
		if req.MaxParallelReplicationTasks.IsZero() || err != nil || string(raw) != "null" {
			t.Errorf("0 marshals to %s (%v), want null", raw, err)
		}
	})
}

func TestReplicationConfigResource_LimitValidator(t *testing.T) {
	ctx := context.Background()
	a, ok := schemaOf(t, ctx, &ReplicationConfigResource{}).Schema.Attributes["max_parallel_replication_tasks"].(schema.Int64Attribute)
	if !ok || len(a.Validators) == 0 {
		t.Fatal("max_parallel_replication_tasks is not an int64 attribute with validators")
	}
	for v, wantErr := range map[int64]bool{-1: true, 0: false, 1: false, 5: false, 100: false} {
		resp := &validator.Int64Response{}
		for _, val := range a.Validators {
			val.ValidateInt64(ctx, validator.Int64Request{
				Path:        path.Root("max_parallel_replication_tasks"),
				ConfigValue: types.Int64Value(v),
			}, resp)
		}
		if got := resp.Diagnostics.HasError(); got != wantErr {
			t.Errorf("limit %d: HasError = %v, want %v", v, got, wantErr)
		}
	}
}

func TestReplicationConfigResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &ReplicationConfigResource{client: replicationConfigClient(ctx, t, replicationConfigBody(5))}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{})

	ok := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: replicationConfigSingletonID}, ok)
	if ok.Diagnostics.HasError() {
		t.Errorf("the canonical import ID was rejected: %v", ok.Diagnostics)
	}
	var id types.String
	if diags := ok.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() || id.ValueString() != replicationConfigSingletonID {
		t.Errorf("imported id = %v (%v)", id, diags)
	}

	bad := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("ImportState accepted an ID other than the singleton's")
	}
}

// A failed write or read has to surface as a diagnostic.
func TestReplicationConfigResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &ReplicationConfigResource{client: failingReplicationConfigClient(ctx, t)}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"id":                             str(replicationConfigSingletonID),
		"max_parallel_replication_tasks": num(3),
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

// Removing the resource must not touch the server.
func TestReplicationConfigResource_DeleteSendsNothing(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			calls++
			return replicationConfigBody(5), nil
		})
	r := &ReplicationConfigResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(replicationConfigSingletonID)})

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
func TestReplicationConfigResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &ReplicationConfigResource{client: failingReplicationConfigClient(ctx, t)}
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
