package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// These tests drive the directory resource Create/Read/Update/Delete
// handlers against tailored mock WS servers and assert both the happy-path
// state AND the error-diagnostic branches, matching the per-resource
// coverage the other resources carry.

func dirStatBody(mode int) map[string]interface{} {
	return map[string]interface{}{
		"realpath": "/mnt/tank/d", "type": "DIRECTORY", "size": 4096,
		"mode": mode, "uid": 1000, "gid": 1000, "acl": false, "is_mountpoint": false,
	}
}

func dirPlanVals(path, mode string, parents bool) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"path":           str(path),
		"mode":           str(mode),
		"create_parents": flag(parents),
		"uid":            num(1000),
		"gid":            num(1000),
	}
}

func newDirResource(t *testing.T, handler wsclient.TestHandler) *DirectoryResource {
	t.Helper()
	ts := wsclient.NewTestServer(t, handler)
	c, err := ts.NewClient(context.Background())
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return &DirectoryResource{client: c}
}

// jobOK answers a setperm/* job call chain with SUCCESS.
func dirHandlerOK(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
	switch method {
	case "filesystem.mkdir", "filesystem.stat":
		return dirStatBody(0o40755), nil
	case "filesystem.setperm":
		return int64(1), nil
	case "core.get_jobs":
		return []interface{}{map[string]interface{}{"id": int64(1), "state": "SUCCESS", "result": nil, "error": ""}}, nil
	}
	return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
}

func TestDirectoryResource_Create_Success(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, dirHandlerOK)
	sch := schemaOf(t, ctx, r)
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, dirPlanVals("/mnt/tank/d", "755", false))}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	var m DirectoryResourceModel
	resp.State.Get(ctx, &m)
	if m.ID.ValueString() != "/mnt/tank/d" || m.Mode.ValueString() != "755" || m.UID.ValueInt64() != 1000 || m.GID.ValueInt64() != 1000 {
		t.Errorf("state: id=%q mode=%q uid=%d gid=%d", m.ID.ValueString(), m.Mode.ValueString(), m.UID.ValueInt64(), m.GID.ValueInt64())
	}
}

func TestDirectoryResource_Create_WithParents(t *testing.T) {
	ctx := context.Background()
	existing := map[string]bool{"/mnt/tank": true}
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "filesystem.stat":
			p, _ := params[0].(string)
			if existing[p] {
				return dirStatBody(0o40755), nil
			}
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Data: json.RawMessage(`{"errname":"ENOENT"}`)}
		case "filesystem.mkdir":
			obj, _ := params[0].(map[string]interface{})
			p, _ := obj["path"].(string)
			existing[p] = true
			return dirStatBody(0o40755), nil
		case "filesystem.setperm":
			return int64(1), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": int64(1), "state": "SUCCESS"}}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	sch := schemaOf(t, ctx, r)
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, dirPlanVals("/mnt/tank/a/b/d", "755", true))}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create(create_parents): %v", resp.Diagnostics)
	}
	if !existing["/mnt/tank/a"] || !existing["/mnt/tank/a/b"] {
		t.Errorf("parents not created: %v", existing)
	}
}

func TestDirectoryResource_Create_MkdirError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		if method == "filesystem.mkdir" {
			return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
		}
		return dirStatBody(0o40755), nil
	})
	sch := schemaOf(t, ctx, r)
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, dirPlanVals("/mnt/tank/d", "755", false))}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Create should error when mkdir fails")
	}
}

func TestDirectoryResource_Create_ParentsError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		// every ancestor stat reports a hard (non-ENOENT) error
		if method == "filesystem.stat" {
			return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
		}
		return dirStatBody(0o40755), nil
	})
	sch := schemaOf(t, ctx, r)
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, dirPlanVals("/mnt/tank/a/b/d", "755", true))}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Create should error when an ancestor stat fails")
	}
}

func TestDirectoryResource_Update_Success(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, dirHandlerOK)
	sch := schemaOf(t, ctx, r)
	vals := dirPlanVals("/mnt/tank/d", "750", false)
	vals["id"] = str("/mnt/tank/d")
	st := stateFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
}

func TestDirectoryResource_Update_SetpermError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		if method == "filesystem.setperm" {
			return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
		}
		return dirStatBody(0o40755), nil
	})
	sch := schemaOf(t, ctx, r)
	vals := dirPlanVals("/mnt/tank/d", "750", false)
	vals["id"] = str("/mnt/tank/d")
	st := stateFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Update should error when setperm fails")
	}
}

func TestDirectoryResource_Read_Success(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, dirHandlerOK)
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("/mnt/tank/d"), "path": str("/mnt/tank/d")})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var m DirectoryResourceModel
	resp.State.Get(ctx, &m)
	if m.Mode.ValueString() != "755" {
		t.Errorf("read mode = %q, want 755", m.Mode.ValueString())
	}
}

func TestDirectoryResource_Read_ServerError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		// a hard (non-ENOENT) error must surface as a diagnostic, not a silent remove
		return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
	})
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("/mnt/tank/d"), "path": str("/mnt/tank/d")})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Read should surface a non-not-found error as a diagnostic")
	}
}

func TestDirectoryResource_Delete_ExistsAndGone(t *testing.T) {
	ctx := context.Background()
	sch := func(r *DirectoryResource) resource.SchemaResponse { return schemaOf(t, ctx, r) }

	// directory still on disk: Delete is state-only, no error, no removal.
	r1 := newDirResource(t, dirHandlerOK)
	s1 := sch(r1)
	st1 := stateFromValues(t, ctx, s1, map[string]tftypes.Value{"id": str("/mnt/tank/d"), "path": str("/mnt/tank/d")})
	d1 := &resource.DeleteResponse{State: st1}
	r1.Delete(ctx, resource.DeleteRequest{State: st1}, d1)
	if d1.Diagnostics.HasError() {
		t.Fatalf("Delete(exists): %v", d1.Diagnostics)
	}

	// directory already gone upstream: IsNotFound short-circuits cleanly.
	r2 := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Data: json.RawMessage(`{"errname":"ENOENT"}`)}
	})
	s2 := sch(r2)
	st2 := stateFromValues(t, ctx, s2, map[string]tftypes.Value{"id": str("/mnt/tank/d"), "path": str("/mnt/tank/d")})
	d2 := &resource.DeleteResponse{State: st2}
	r2.Delete(ctx, resource.DeleteRequest{State: st2}, d2)
	if d2.Diagnostics.HasError() {
		t.Fatalf("Delete(gone): %v", d2.Diagnostics)
	}
}

// statFailHandler succeeds mkdir + the setperm job but fails the read-back
// stat, exercising the post-mutation "Error Reading Directory" branch.
func statFailHandler(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
	switch method {
	case "filesystem.mkdir":
		return dirStatBody(0o40755), nil
	case "filesystem.setperm":
		return int64(1), nil
	case "core.get_jobs":
		return []interface{}{map[string]interface{}{"id": int64(1), "state": "SUCCESS"}}, nil
	case "filesystem.stat":
		return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
	}
	return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
}

func TestDirectoryResource_Create_StatError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, statFailHandler)
	sch := schemaOf(t, ctx, r)
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, dirPlanVals("/mnt/tank/d", "755", false))}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Create should error when the read-back stat fails")
	}
}

func TestDirectoryResource_Update_StatError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, statFailHandler)
	sch := schemaOf(t, ctx, r)
	vals := dirPlanVals("/mnt/tank/d", "750", false)
	vals["id"] = str("/mnt/tank/d")
	st := stateFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Update should error when the read-back stat fails")
	}
}
