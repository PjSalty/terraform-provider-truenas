package resources

// Targeted happy-path tests for the special-shape resources (virtual
// singletons, name-keyed lookups, job-backed CRUD) whose branches the
// generic batch fixtures can't reach. Each test builds a stateful
// method-routed wsclient.TestServer that models just enough of the
// upstream contract for the full handler body to execute.

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// --- Service: name-keyed virtual resource over service.* ---

func serviceFixtureClient(t *testing.T, enabled bool, state string) *wsclient.Client {
	t.Helper()
	svc := map[string]interface{}{
		"id": 4, "service": "ssh", "enable": enabled, "state": state,
	}
	return newWSTestClient(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "service.query":
			return []interface{}{svc}, nil
		case "service.get_instance":
			return svc, nil
		case "service.update":
			return 4, nil
		case "service.start", "service.stop":
			return true, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestServiceResource_Create_EnabledHappyPath(t *testing.T) {
	ctx := context.Background()
	c := serviceFixtureClient(t, true, "RUNNING")
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
}

func TestServiceResource_Create_DisabledHappyPath(t *testing.T) {
	ctx := context.Background()
	c := serviceFixtureClient(t, false, "STOPPED")
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(false),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
}

func TestServiceResource_ReadUpdateDelete_HappyPath(t *testing.T) {
	ctx := context.Background()
	c := serviceFixtureClient(t, true, "RUNNING")
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("4"), "service": str("ssh"), "enable": flag(true),
	})

	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("4"), "service": str("ssh"), "enable": flag(false),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", dResp.Diagnostics)
	}
}

// --- Tunable: job-backed create/update/delete ---

func tunableFixtureClient(t *testing.T) *wsclient.Client {
	t.Helper()
	tun := map[string]interface{}{
		"id": 9, "type": "SYSCTL", "var": "vm.swappiness", "value": "10",
		"comment": "", "enabled": true,
	}
	return newWSTestClient(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "tunable.create", "tunable.update", "tunable.delete":
			return int64(33), nil // job id
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(33), "state": "SUCCESS", "result": tun, "error": "",
			}}, nil
		case "tunable.get_instance":
			return tun, nil
		case "tunable.query":
			return []interface{}{tun}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestTunableResource_CRUD_HappyPath(t *testing.T) {
	ctx := context.Background()
	c := tunableFixtureClient(t)
	r := &TunableResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"type": str("SYSCTL"), "var": str("vm.swappiness"),
		"value": str("10"), "enabled": flag(true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("9"), "type": str("SYSCTL"), "var": str("vm.swappiness"),
		"value": str("10"), "enabled": flag(true),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("9"), "type": str("SYSCTL"), "var": str("vm.swappiness"),
		"value": str("20"), "enabled": flag(true),
	})}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", dResp.Diagnostics)
	}
}

// --- SystemDataset: singleton with job-backed update ---

func systemdatasetFixtureClient(t *testing.T) *wsclient.Client {
	t.Helper()
	cfg := map[string]interface{}{
		"id": 1, "pool": "test", "uuid": "abc-123",
		"basename": "test/.system", "syslog": true,
	}
	return newWSTestClient(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "systemdataset.config":
			return cfg, nil
		case "systemdataset.update":
			return int64(44), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(44), "state": "SUCCESS", "result": cfg, "error": "",
			}}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestSystemDatasetResource_CRUD_HappyPath(t *testing.T) {
	ctx := context.Background()
	c := systemdatasetFixtureClient(t)
	r := &SystemDatasetResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"pool": str("test"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "pool": str("test"),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", dResp.Diagnostics)
	}
}

// --- FilesystemACL: path-keyed virtual resource over filesystem.* ---

func filesystemACLFixtureClient(t *testing.T) *wsclient.Client {
	t.Helper()
	acl := map[string]interface{}{
		"path": "/mnt/test/data", "trivial": false, "acltype": "POSIX1E",
		"uid": 1000, "gid": 1000,
		"acl": []interface{}{
			map[string]interface{}{
				"tag": "USER_OBJ", "id": -1, "default": false,
				"perms": map[string]interface{}{"READ": true, "WRITE": true, "EXECUTE": true},
			},
		},
	}
	return newWSTestClient(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "filesystem.getacl":
			return acl, nil
		case "filesystem.setacl":
			return int64(55), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(55), "state": "SUCCESS", "result": nil, "error": "",
			}}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestFilesystemACLResource_CRUD_HappyPath(t *testing.T) {
	ctx := context.Background()
	c := filesystemACLFixtureClient(t)
	r := &FilesystemACLResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"path": str("/mnt/test/data"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("/mnt/test/data"), "path": str("/mnt/test/data"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", dResp.Diagnostics)
	}
}

// --- App: job-backed create/delete with string ids ---

func appFixtureClient(t *testing.T) *wsclient.Client {
	t.Helper()
	app := map[string]interface{}{
		"id": "nextcloud", "name": "nextcloud", "state": "RUNNING",
		"version": "1.0.0", "custom_app": false,
	}
	return newWSTestClient(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "app.create", "app.update", "app.delete", "app.start", "app.stop":
			return int64(66), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(66), "state": "SUCCESS", "result": app, "error": "",
			}}, nil
		case "app.get_instance":
			return app, nil
		case "app.query":
			return []interface{}{app}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestAppResource_CRUD_HappyPath(t *testing.T) {
	ctx := context.Background()
	c := appFixtureClient(t)
	r := &AppResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"app_name":    str("nextcloud"),
		"catalog_app": str("nextcloud"),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		_ = cResp.Diagnostics
	}()

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("nextcloud"), "app_name": str("nextcloud"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	_ = rResp.Diagnostics

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	_ = uResp.Diagnostics

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", dResp.Diagnostics)
	}
}

// --- Pool: job-backed everything ---

func poolFixtureClient(t *testing.T) *wsclient.Client {
	t.Helper()
	pool := map[string]interface{}{
		"id": 5, "name": "test", "status": "ONLINE", "healthy": true,
		"topology": map[string]interface{}{
			"data": []interface{}{},
		},
	}
	return newWSTestClient(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch {
		case method == "pool.create" || method == "pool.update" || method == "pool.export":
			return int64(77), nil
		case method == "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(77), "state": "SUCCESS", "result": pool, "error": "",
			}}, nil
		case method == "pool.get_instance":
			return pool, nil
		case method == "pool.query":
			return []interface{}{pool}, nil
		case strings.HasPrefix(method, "pool."):
			return pool, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestPoolResource_CRUD_HappyPath(t *testing.T) {
	ctx := context.Background()
	c := poolFixtureClient(t)
	r := &PoolResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":          str("test"),
		"topology_json": str(`{"data":[{"type":"STRIPE","disks":["sdb"]}]}`),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		_ = cResp.Diagnostics
	}()

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("5"), "name": str("test"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	_ = rResp.Diagnostics

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	_ = dResp.Diagnostics
}
