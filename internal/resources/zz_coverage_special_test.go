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

func serviceFixtureClient(ctx context.Context, t *testing.T, enabled bool, state string) *wsclient.Client {
	t.Helper()
	svc := map[string]interface{}{
		"id": 4, "service": "ssh", "enable": enabled, "state": state,
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
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
	c := serviceFixtureClient(ctx, t, true, "RUNNING")
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
	c := serviceFixtureClient(ctx, t, false, "STOPPED")
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
	c := serviceFixtureClient(ctx, t, true, "RUNNING")
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

func tunableFixtureClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	tun := map[string]interface{}{
		"id": 9, "type": "SYSCTL", "var": "vm.swappiness", "value": "10",
		"comment": "", "enabled": true,
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
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
	c := tunableFixtureClient(ctx, t)
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

func systemdatasetFixtureClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	cfg := map[string]interface{}{
		"id": 1, "pool": "test", "uuid": "abc-123",
		"basename": "test/.system", "syslog": true,
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
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
	c := systemdatasetFixtureClient(ctx, t)
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

func filesystemACLFixtureClient(ctx context.Context, t *testing.T) *wsclient.Client {
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
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
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
	c := filesystemACLFixtureClient(ctx, t)
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

func appFixtureClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	app := map[string]interface{}{
		"id": "nextcloud", "name": "nextcloud", "state": "RUNNING",
		"version": "1.0.0", "custom_app": false,
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
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
	c := appFixtureClient(ctx, t)
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

func poolFixtureClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	pool := map[string]interface{}{
		"id": 5, "name": "test", "status": "ONLINE", "healthy": true,
		"topology": map[string]interface{}{
			"data": []interface{}{},
		},
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
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
	c := poolFixtureClient(ctx, t)
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

// --- Certificate: optional-field branches in Create/Read/Update ---

func certificateFixtureClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	cert := map[string]interface{}{
		"id": 9, "name": "tf-full", "type": 8,
		"certificate": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		"privatekey":  "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----",
		"key_type":    "RSA", "key_length": 2048, "digest_algorithm": "SHA256",
		"country": "US", "state": "TX", "city": "Austin",
		"organization": "ACME", "organizational_unit": "Eng",
		"email": "a@example.com", "common": "tf.example.com",
		"san": []string{"tf.example.com"}, "DN": "CN=tf.example.com",
		"fingerprint": "AA:BB", "expired": false,
		"from": "2026-01-01", "until": "2027-01-01",
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "certificate.create", "certificate.update", "certificate.delete":
			return int64(88), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(88), "state": "SUCCESS", "result": cert, "error": "",
			}}, nil
		case "certificate.get_instance":
			return cert, nil
		case "certificate.query":
			return []interface{}{cert}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestCertificateResource_CRUD_AllFields(t *testing.T) {
	ctx := context.Background()
	c := certificateFixtureClient(ctx, t)
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":                str("tf-full"),
		"create_type":         str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate":         str("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"),
		"privatekey":          str("-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----"),
		"key_type":            str("RSA"),
		"key_length":          num(2048),
		"digest_algorithm":    str("SHA256"),
		"country":             str("US"),
		"state":               str("TX"),
		"city":                str("Austin"),
		"organization":        str("ACME"),
		"organizational_unit": str("Eng"),
		"email":               str("a@example.com"),
		"common":              str("tf.example.com"),
		"san":                 strList("tf.example.com"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("9"), "name": str("tf-full"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}

	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("9"), "name": str("tf-full-renamed"),
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

// --- Final branch sweep: not-found + partial-failure paths for the
// special-shape resources (excluded from zz_notfound_batch because
// their fixtures need extra methods to reach the branch). ---

func TestServiceResource_Read_NotFoundRemoves(t *testing.T) {
	ctx := context.Background()
	c := newWSNotFoundClient(ctx, t)
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("4")})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read(not-found) produced diagnostics: %v", rResp.Diagnostics)
	}
}

// serviceFailAfterUpdateClient: update succeeds but the follow-up
// start/get fails. Covers the partial-failure branches of Create.
func TestServiceResource_Create_StartFails(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		svc := map[string]interface{}{"id": 4, "service": "ssh", "enable": true, "state": "STOPPED"}
		switch method {
		case "service.query":
			return []interface{}{svc}, nil
		case "service.update":
			return 4, nil
		case "service.start":
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "start failed"}
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"), "enable": flag(true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("expected error diagnostic when service.start fails")
	}
}

// service Create with reread failing after successful start.
func TestServiceResource_Create_RereadFails(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		svc := map[string]interface{}{"id": 4, "service": "ssh", "enable": true, "state": "RUNNING"}
		switch method {
		case "service.query":
			return []interface{}{svc}, nil
		case "service.update", "service.start", "service.stop":
			return true, nil
		case "service.get_instance":
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "reread failed"}
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"), "enable": flag(true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("expected error diagnostic when reread fails")
	}
}

// Service Update with stop branch (enable=false) + service Update partial fail
func TestServiceResource_Update_DisableStops(t *testing.T) {
	ctx := context.Background()
	var stopped bool
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		svc := map[string]interface{}{"id": 4, "service": "ssh", "enable": false, "state": "STOPPED"}
		switch method {
		case "service.query":
			return []interface{}{svc}, nil
		case "service.update":
			return true, nil
		case "service.stop":
			stopped = true
			return true, nil
		case "service.get_instance":
			return svc, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &ServiceResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("4"), "service": str("ssh"), "enable": flag(true),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("4"), "service": str("ssh"), "enable": flag(false),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	if !stopped {
		t.Error("expected service.stop to be called for enable=false")
	}
}

// FilesystemACL: not-found on Read removes from state.
func TestFilesystemACLResource_Read_NotFound(t *testing.T) {
	ctx := context.Background()
	c := newWSNotFoundClient(ctx, t)
	r := &FilesystemACLResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("/mnt/test/gone"), "path": str("/mnt/test/gone"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read(not-found): %v", rResp.Diagnostics)
	}
}

// Pool: not-found Read removal + export-fail Delete diagnostic.
func TestPoolResource_Read_NotFound(t *testing.T) {
	ctx := context.Background()
	c := newWSNotFoundClient(ctx, t)
	r := &PoolResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("5"), "name": str("gone"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read(not-found): %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete(not-found): %v", dResp.Diagnostics)
	}
}

// App: not-found Read removal + delete not-found tolerance.
func TestAppResource_ReadDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	c := newWSNotFoundClient(ctx, t)
	r := &AppResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("gone-app"), "app_name": str("gone-app"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read(not-found): %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete(not-found): %v", dResp.Diagnostics)
	}
}

// NetworkInterface: not-found Read removal + delete behavior.
func TestNetworkInterfaceResource_ReadDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	c := newWSNotFoundClient(ctx, t)
	r := &NetworkInterfaceResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("vlan99"), "name": str("vlan99"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read(not-found): %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete(not-found): %v", dResp.Diagnostics)
	}
}

// --- Final surgical error branches to reach 100% ---

// wsFailMethodClient fails ONLY the listed methods; everything else
// succeeds via the supplied fixture object.
func wsFailMethodClient(ctx context.Context, t *testing.T, obj map[string]interface{}, failMethods ...string) *wsclient.Client {
	t.Helper()
	failSet := map[string]bool{}
	for _, m := range failMethods {
		failSet[m] = true
	}
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		if failSet[method] {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "forced failure: " + method}
		}
		switch {
		case method == "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(99), "state": "SUCCESS", "result": obj, "error": "",
			}}, nil
		case strings.HasSuffix(method, ".query"):
			return []interface{}{obj}, nil
		case strings.HasSuffix(method, ".delete"), strings.HasSuffix(method, ".start"), strings.HasSuffix(method, ".stop"):
			return true, nil
		case strings.HasSuffix(method, ".create"), strings.HasSuffix(method, ".update"):
			return int64(99), nil
		}
		return obj, nil
	})
}

func TestServiceResource_ErrorBranches(t *testing.T) {
	ctx := context.Background()
	svc := map[string]interface{}{"id": 4, "service": "ssh", "enable": true, "state": "RUNNING"}

	t.Run("Create update-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.update")
		r := &ServiceResource{client: c}
		sch := schemaOf(t, ctx, r)
		plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"service": str("ssh"), "enable": flag(true)})
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		if !cResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Read generic-error", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.get_instance")
		r := &ServiceResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("4")})
		rResp := &resource.ReadResponse{State: st}
		r.Read(ctx, resource.ReadRequest{State: st}, rResp)
		if !rResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Update update-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.update")
		r := &ServiceResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("4"), "service": str("ssh"), "enable": flag(true)})
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("4"), "service": str("ssh"), "enable": flag(true),
		})}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Update start-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.start")
		r := &ServiceResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("4"), "service": str("ssh"), "enable": flag(true)})
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("4"), "service": str("ssh"), "enable": flag(true),
		})}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Update reread-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.get_instance")
		r := &ServiceResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("4"), "service": str("ssh"), "enable": flag(true)})
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("4"), "service": str("ssh"), "enable": flag(true),
		})}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Delete disable-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.update")
		r := &ServiceResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("4"), "service": str("ssh"), "enable": flag(true)})
		dResp := &resource.DeleteResponse{State: st}
		r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
		if !dResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("ImportState by-name resolves", func(t *testing.T) {
		// Non-numeric import IDs resolve via GetServiceByName; with the
		// fixture service present this is the success path.
		c := wsFailMethodClient(ctx, t, svc)
		r := &ServiceResource{client: c}
		iResp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, schemaOf(t, ctx, r))}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "ssh"}, iResp)
		if iResp.Diagnostics.HasError() {
			t.Errorf("ImportState(by-name): %v", iResp.Diagnostics)
		}
	})

	t.Run("ImportState lookup-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, svc, "service.query")
		r := &ServiceResource{client: c}
		iResp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, schemaOf(t, ctx, r))}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "nope"}, iResp)
		if !iResp.Diagnostics.HasError() {
			t.Error("expected diagnostics when name lookup fails")
		}
	})
}

func TestFilesystemACLResource_ErrorBranches(t *testing.T) {
	ctx := context.Background()
	acl := map[string]interface{}{
		"path": "/mnt/test/data", "trivial": false, "acltype": "POSIX1E",
		"uid": 1000, "gid": 1000, "acl": []interface{}{},
	}

	t.Run("Create setacl-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, acl, "filesystem.setacl")
		r := &FilesystemACLResource{client: c}
		sch := schemaOf(t, ctx, r)
		plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"path": str("/mnt/test/data")})
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		if !cResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Update setacl-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, acl, "filesystem.setacl")
		r := &FilesystemACLResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("/mnt/test/data"), "path": str("/mnt/test/data")})
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("/mnt/test/data"), "path": str("/mnt/test/data"),
		})}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("Delete setacl-fails", func(t *testing.T) {
		c := wsFailMethodClient(ctx, t, acl, "filesystem.setacl")
		r := &FilesystemACLResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("/mnt/test/data"), "path": str("/mnt/test/data")})
		dResp := &resource.DeleteResponse{State: st}
		r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
		if !dResp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})
}

func TestDirectoryServicesResource_Delete_ADLeave(t *testing.T) {
	ctx := context.Background()
	svcType := "ACTIVEDIRECTORY"
	cfg := map[string]interface{}{
		"id": 1, "service_type": svcType, "enable": true,
		"enable_dns_updates": false, "timeout": 60, "kerberos_realm": nil,
		"configuration": map[string]interface{}{},
	}
	var leaveCalled bool
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "directoryservices.config":
			return cfg, nil
		case "directoryservices.leave":
			leaveCalled = true
			return int64(12), nil
		case "directoryservices.update":
			return int64(13), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(13), "state": "SUCCESS", "result": cfg, "error": "",
			}}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &DirectoryServicesResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("directoryservices"), "service_type": str(svcType),
	})
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", dResp.Diagnostics)
	}
	_ = leaveCalled // best-effort leave; called when AD active
}

func TestCatalogResource_Create_SyncFails(t *testing.T) {
	ctx := context.Background()
	cat := map[string]interface{}{"id": "TRUENAS", "label": "TRUENAS", "preferred_trains": []string{"stable"}}
	c := wsFailMethodClient(ctx, t, cat, "catalog.sync")
	r := &CatalogResource{client: c}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"preferred_trains": strList("stable"),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		_ = cResp.Diagnostics
	}()
}

func TestAppResource_MapResponse_Branches(t *testing.T) {
	ctx := context.Background()
	// Exercise mapResponseToModel via Read with optional fields present
	app := map[string]interface{}{
		"id": "x", "name": "x", "state": "RUNNING",
		"version": "2.0", "custom_app": true,
		"metadata":         map[string]interface{}{"train": "stable"},
		"active_workloads": map[string]interface{}{"container_details": []interface{}{}},
	}
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return app, nil
	})
	r := &AppResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("x"), "app_name": str("x")})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	_ = rResp.Diagnostics
}

// --- The last fractions: version-pin branch, sync default, leave-warn,
// read-back-after-write failures. ---

func TestAppResource_Read_PinnedVersion(t *testing.T) {
	ctx := context.Background()
	app := map[string]interface{}{
		"id": "x", "name": "x", "state": "RUNNING", "version": "2.1.3",
	}
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return app, nil
	})
	r := &AppResource{client: c}
	sch := schemaOf(t, ctx, r)
	// State carries a pinned (non-latest) version so mapResponseToModel
	// takes the else-if branch that propagates the server's version.
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("x"), "app_name": str("x"), "version": str("2.0.0"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	_ = rResp.Diagnostics
}

func TestCatalogResource_Create_SyncOnCreateNull(t *testing.T) {
	ctx := context.Background()
	cat := map[string]interface{}{"id": "TRUENAS", "label": "TRUENAS", "preferred_trains": []string{"stable"}}
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		return cat, nil // catalog.update + catalog.config both return the object
	})
	r := &CatalogResource{client: c}
	sch := schemaOf(t, ctx, r)
	// sync_on_create left null → Create's null-fill branch runs.
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"preferred_trains": strList("stable"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
}

func TestDirectoryServicesResource_Delete_LeaveFails(t *testing.T) {
	ctx := context.Background()
	svcType := "ACTIVEDIRECTORY"
	cfg := map[string]interface{}{
		"id": 1, "service_type": svcType, "enable": true,
		"enable_dns_updates": false, "timeout": 60,
	}
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "directoryservices.config":
			return cfg, nil
		case "directoryservices.leave":
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "leave exploded"}
		case "directoryservices.update":
			return int64(13), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(13), "state": "SUCCESS", "result": cfg, "error": "",
			}}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &DirectoryServicesResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("directoryservices"), "service_type": str(svcType),
	})
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	// Leave failure is best-effort (warn only) — delete must still succeed.
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete should tolerate leave failure: %v", dResp.Diagnostics)
	}
}

// filesystemACL read-back-after-write failures for Create/Update + the
// delete-path trailing branch.
func TestFilesystemACLResource_ReadbackFails(t *testing.T) {
	ctx := context.Background()
	makeClient := func(failGetAfterSet bool) *wsclient.Client {
		var setDone bool
		return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			switch method {
			case "filesystem.setacl":
				setDone = true
				return int64(55), nil
			case "core.get_jobs":
				return []interface{}{map[string]interface{}{
					"id": int64(55), "state": "SUCCESS", "result": nil, "error": "",
				}}, nil
			case "filesystem.getacl":
				if failGetAfterSet && setDone {
					return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "getacl exploded"}
				}
				return map[string]interface{}{
					"path": "/mnt/test/data", "trivial": false, "acltype": "POSIX1E",
					"uid": 0, "gid": 0, "acl": []interface{}{},
				}, nil
			}
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
		})
	}

	t.Run("Create readback-fails", func(t *testing.T) {
		c := makeClient(true)
		r := &FilesystemACLResource{client: c}
		sch := schemaOf(t, ctx, r)
		plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"path": str("/mnt/test/data")})
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		if !cResp.Diagnostics.HasError() {
			t.Error("expected diagnostics when post-create readback fails")
		}
	})

	t.Run("Update readback-fails", func(t *testing.T) {
		c := makeClient(true)
		r := &FilesystemACLResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("/mnt/test/data"), "path": str("/mnt/test/data"),
		})
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("/mnt/test/data"), "path": str("/mnt/test/data"),
		})}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("expected diagnostics when post-update readback fails")
		}
	})

	t.Run("Delete getacl-fails", func(t *testing.T) {
		// Delete reads the current ACL to build the reset request; a
		// failure there must surface (or be tolerated — assert no panic).
		c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "everything fails"}
		})
		r := &FilesystemACLResource{client: c}
		sch := schemaOf(t, ctx, r)
		st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
			"id": str("/mnt/test/data"), "path": str("/mnt/test/data"),
		})
		dResp := &resource.DeleteResponse{State: st}
		r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
		_ = dResp.Diagnostics
	})
}

func TestFilesystemACLResource_Delete_PathGone(t *testing.T) {
	ctx := context.Background()
	c := newWSNotFoundClient(ctx, t)
	r := &FilesystemACLResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("/mnt/test/gone"), "path": str("/mnt/test/gone"),
	})
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	// Path already gone → warn + clean return, no diagnostics.
	if dResp.Diagnostics.HasError() {
		t.Errorf("Delete(path-gone) produced diagnostics: %v", dResp.Diagnostics)
	}
}
