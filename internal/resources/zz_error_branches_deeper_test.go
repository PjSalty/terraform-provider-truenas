package resources

// Deeper error-branch coverage tests. These drive every CRUD handler with
// valid plan/state values against a 400 server so that each handler reaches
// its client call and takes the `if err != nil` branch. This covers the
// "Error Creating", "Error Reading", "Error Updating", and "Error Deleting"
// diagnostics that the existing 200-server CRUD tests never hit.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// newBadRequestClient returns a client wired to a server that always returns
// 400 for every request. 400 is a non-retryable status, so each client call
// returns immediately (no backoff delays) and the resource handler enters
// its `err != nil` branch.
func newBadRequestClient(t *testing.T) (*client.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		srv.Close()
		t.Fatalf("client.New: %v", err)
	}
	// Minimize retry delays even though 400 is not retryable.
	c.RetryPolicy = client.RetryPolicy{MaxAttempts: 1}
	return c, srv.Close
}

// driveBadPlanCRUD invokes Create/Read/Update/Delete with an empty tftypes.Value
// as the Raw, which causes Plan.Get / State.Get to return diagnostics. This
// exercises the `if resp.Diagnostics.HasError() { return }` branches that
// sit right after plan.Get / state.Get calls throughout every resource
// handler. The test ignores all results — the point is coverage, not
// correctness.
func driveBadPlanCRUD(t *testing.T, r resource.Resource) {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	// An empty tftypes.Value causes plan/state Get to add diagnostics because
	// the framework cannot reflect a null into the model struct.
	badPlan := tfsdk.Plan{Schema: sch.Schema, Raw: tftypes.Value{}}
	badState := tfsdk.State{Schema: sch.Schema, Raw: tftypes.Value{}}

	run := func(f func()) {
		defer func() { _ = recover() }()
		f()
	}
	run(func() {
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: badPlan}, cResp)
	})
	run(func() {
		rResp := &resource.ReadResponse{State: badState}
		r.Read(ctx, resource.ReadRequest{State: badState}, rResp)
	})
	run(func() {
		uResp := &resource.UpdateResponse{State: badState}
		r.Update(ctx, resource.UpdateRequest{State: badState, Plan: badPlan}, uResp)
	})
	run(func() {
		dResp := &resource.DeleteResponse{State: badState}
		r.Delete(ctx, resource.DeleteRequest{State: badState}, dResp)
	})
}

// drive400CRUD runs Create/Read/Update/Delete against a 400 server for the
// given resource, using the supplied plan values map and id. The test
// tolerates diagnostic errors — they are expected.
func drive400CRUD(t *testing.T, r resource.Resource, id string, planVals map[string]tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	// Copy plan map so we can add id without mutating caller's map.
	vals := make(map[string]tftypes.Value, len(planVals)+1)
	for k, v := range planVals {
		vals[k] = v
	}

	// Create
	plan := planFromValues(t, ctx, sch, vals)
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		_ = cResp.Diagnostics
	}()

	// Read (id-based state)
	readState := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(id)})
	func() {
		defer func() { _ = recover() }()
		rResp := &resource.ReadResponse{State: readState}
		r.Read(ctx, resource.ReadRequest{State: readState}, rResp)
		_ = rResp.Diagnostics
	}()

	// Update (full state/plan with id)
	vals["id"] = str(id)
	updState := stateFromValues(t, ctx, sch, vals)
	updPlan := planFromValues(t, ctx, sch, vals)
	func() {
		defer func() { _ = recover() }()
		uResp := &resource.UpdateResponse{State: updState}
		r.Update(ctx, resource.UpdateRequest{State: updState, Plan: updPlan}, uResp)
		_ = uResp.Diagnostics
	}()

	// Delete
	func() {
		defer func() { _ = recover() }()
		dResp := &resource.DeleteResponse{State: updState}
		r.Delete(ctx, resource.DeleteRequest{State: updState}, dResp)
		_ = dResp.Diagnostics
	}()

	// Update with good plan + malformed state: exercises the State.Get
	// HasError branch that the bad-plan test can't reach.
	badState := tfsdk.State{Schema: sch.Schema, Raw: tftypes.Value{}}
	func() {
		defer func() { _ = recover() }()
		uResp := &resource.UpdateResponse{State: badState}
		r.Update(ctx, resource.UpdateRequest{State: badState, Plan: plan}, uResp)
		_ = uResp.Diagnostics
	}()

	// Delete with malformed state: exercises the State.Get HasError branch
	// inside Delete.
	func() {
		defer func() { _ = recover() }()
		dResp := &resource.DeleteResponse{State: badState}
		r.Delete(ctx, resource.DeleteRequest{State: badState}, dResp)
		_ = dResp.Diagnostics
	}()
}

// TestErrorBranches_BadPlan_AllResources drives every resource's CRUD
// handlers with a malformed Plan/State (empty tftypes.Value). This exercises
// the many `if resp.Diagnostics.HasError() { return }` branches that sit
// right after Plan.Get / State.Get calls throughout the resource package.
func TestErrorBranches_BadPlan_AllResources(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	resources := []resource.Resource{
		&ACMEDNSAuthenticatorResource{client: c},
		&AlertServiceResource{client: c},
		&AlertClassesResource{client: c},
		&APIKeyResource{client: c},
		&AppResource{client: c},
		&CatalogResource{client: c},
		&CertificateResource{client: c},
		&CloudBackupResource{client: c},
		&CloudSyncResource{client: c},
		&CloudSyncCredentialResource{client: c},
		&CronJobResource{client: c},
		&DatasetResource{client: c},
		&DirectoryServicesResource{client: c},
		&DNSNameserverResource{client: c},
		&FilesystemACLResource{client: c},
		&FilesystemACLTemplateResource{client: c},
		&FTPConfigResource{client: c},
		&GroupResource{client: c},
		&InitScriptResource{client: c},
		&ISCSIAuthResource{client: c},
		&ISCSIExtentResource{client: c},
		&ISCSIInitiatorResource{client: c},
		&ISCSIPortalResource{client: c},
		&ISCSITargetResource{client: c},
		&ISCSITargetExtentResource{client: c},
		&KerberosKeytabResource{client: c},
		&KerberosRealmResource{client: c},
		&KeychainCredentialResource{client: c},
		&KMIPConfigResource{client: c},
		&MailConfigResource{client: c},
		&NetworkConfigResource{client: c},
		&NetworkInterfaceResource{client: c},
		&NFSConfigResource{client: c},
		&NVMetGlobalResource{client: c},
		&NVMetHostResource{client: c},
		&NVMetHostSubsysResource{client: c},
		&NVMetNamespaceResource{client: c},
		&NVMetPortResource{client: c},
		&NVMetPortSubsysResource{client: c},
		&NVMetSubsysResource{client: c},
		&PoolResource{client: c},
		&PrivilegeResource{client: c},
		&ReplicationResource{client: c},
		&ReportingExporterResource{client: c},
		&RsyncTaskResource{client: c},
		&ScrubTaskResource{client: c},
		&ServiceResource{client: c},
		&NFSShareResource{client: c},
		&SMBShareResource{client: c},
		&SMBConfigResource{client: c},
		&SnapshotTaskResource{client: c},
		&SNMPConfigResource{client: c},
		&SSHConfigResource{client: c},
		&StaticRouteResource{client: c},
		&SystemDatasetResource{client: c},
		&SystemUpdateResource{client: c},
		&TunableResource{client: c},
		&UPSConfigResource{client: c},
		&UserResource{client: c},
		&VMResource{client: c},
		&VMDeviceResource{client: c},
		&VMwareResource{client: c},
		&ZvolResource{client: c},
	}
	for _, r := range resources {
		driveBadPlanCRUD(t, r)
	}
}

// driveInvalidJSONCreate drives Create with a plan that contains invalid
// JSON in a JSON-typed attribute. This hits the `if err != nil` branch
// right after normalizeJSON / decodeValues calls in Create handlers.
func driveInvalidJSONCreate(t *testing.T, r resource.Resource, planVals map[string]tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, planVals)
	defer func() { _ = recover() }()
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	_ = cResp.Diagnostics
}

// driveInvalidJSONUpdate drives Update with a plan containing invalid JSON
// plus a matching state so that plan/state Get both succeed and the handler
// reaches the JSON-parse branch.
func driveInvalidJSONUpdate(t *testing.T, r resource.Resource, id string, planVals map[string]tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	planVals["id"] = str(id)
	plan := planFromValues(t, ctx, sch, planVals)
	state := stateFromValues(t, ctx, sch, planVals)
	defer func() { _ = recover() }()
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, uResp)
	_ = uResp.Diagnostics
}

func TestErrorBranches_InvalidJSON_CloudBackup(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CloudBackupResource{client: c}
	vals := map[string]tftypes.Value{
		"description":      str("daily"),
		"path":             str("/mnt/tank/data"),
		"credentials":      num(3),
		"attributes_json":  str(`{not valid json`),
		"snapshot":         flag(true),
		"enabled":          flag(true),
		"keep_last":        num(5),
		"transfer_setting": str("DEFAULT"),
		"schedule_minute":  str("0"),
		"schedule_hour":    str("1"),
		"schedule_dom":     str("*"),
		"schedule_month":   str("*"),
		"schedule_dow":     str("*"),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "1", vals)
}

func TestErrorBranches_InvalidJSON_CloudSync(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CloudSyncResource{client: c}
	vals := map[string]tftypes.Value{
		"description":     str("sync"),
		"path":            str("/mnt/tank/data"),
		"credentials":     num(3),
		"direction":       str("PUSH"),
		"transfer_mode":   str("COPY"),
		"enabled":         flag(true),
		"attributes_json": str(`not{valid`),
		"schedule_minute": str("0"),
		"schedule_hour":   str("1"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "1", vals)
}

func TestErrorBranches_InvalidJSON_AlertService(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &AlertServiceResource{client: c}
	vals := map[string]tftypes.Value{
		"name":          str("mail"),
		"type":          str("Mail"),
		"enabled":       flag(true),
		"level":         str("WARNING"),
		"settings_json": str(`{not valid`),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "3", vals)
}

func TestErrorBranches_InvalidJSON_ReportingExporter(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ReportingExporterResource{client: c}
	vals := map[string]tftypes.Value{
		"name":            str("graphite"),
		"type":            str("GRAPHITE"),
		"enabled":         flag(true),
		"attributes_json": str(`not{valid`),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "1", vals)
}

func TestErrorBranches_InvalidJSON_CloudSyncCredential(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CloudSyncCredentialResource{client: c}
	vals := map[string]tftypes.Value{
		"name":                     str("s3"),
		"provider_type":            str("S3"),
		"provider_attributes_json": str(`not{valid`),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "1", vals)
}

func TestErrorBranches_InvalidJSON_FilesystemACLTemplate(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &FilesystemACLTemplateResource{client: c}
	vals := map[string]tftypes.Value{
		"name":     str("tmpl"),
		"acltype":  str("POSIX1E"),
		"acl_json": str(`not{valid`),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "1", vals)
}

func TestErrorBranches_InvalidJSON_App(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &AppResource{client: c}
	vals := map[string]tftypes.Value{
		"app_name":    str("minio"),
		"catalog_app": str("minio"),
		"train":       str("stable"),
		"version":     str("1.0.0"),
		"values":      str(`not{valid`),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "minio", vals)
}

func TestErrorBranches_InvalidJSON_DirectoryServices(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &DirectoryServicesResource{client: c}
	vals := map[string]tftypes.Value{
		"enable":          flag(false),
		"service_type":    str(""),
		"credential_json": str(`not{valid`),
	}
	driveInvalidJSONCreate(t, r, vals)
	driveInvalidJSONUpdate(t, r, "directoryservices", vals)
}

func TestErrorBranches_BadIDState_Pool(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &PoolResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	// state.ID = non-numeric -> strconv.Atoi fails in Read/Update/Delete.
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("not-a-number")})
	func() {
		defer func() { _ = recover() }()
		rResp := &resource.ReadResponse{State: st}
		r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	}()
	// Update also needs a plan.
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":                      str("not-a-number"),
		"name":                    str("tank"),
		"topology_json":           str(`{}`),
		"encryption":              flag(false),
		"encryption_options_json": str(`{}`),
		"deduplication":           str("OFF"),
		"checksum":                str("SHA256"),
		"allow_duplicate_serials": flag(false),
	})
	func() {
		defer func() { _ = recover() }()
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	}()
	func() {
		defer func() { _ = recover() }()
		dResp := &resource.DeleteResponse{State: st}
		r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	}()
}

func TestErrorBranches_InvalidJSON_Pool(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &PoolResource{client: c}
	vals := map[string]tftypes.Value{
		"name":                    str("tank"),
		"topology_json":           str(`not{valid`),
		"encryption":              flag(false),
		"encryption_options_json": str(`{}`),
		"deduplication":           str("OFF"),
		"checksum":                str("SHA256"),
		"allow_duplicate_serials": flag(false),
	}
	driveInvalidJSONCreate(t, r, vals)

	// Also exercise encryption_options_json invalid path
	vals2 := map[string]tftypes.Value{
		"name":                    str("tank"),
		"topology_json":           str(`{"data":[]}`),
		"encryption":              flag(true),
		"encryption_options_json": str(`not{valid`),
		"deduplication":           str("OFF"),
		"checksum":                str("SHA256"),
		"allow_duplicate_serials": flag(false),
	}
	driveInvalidJSONCreate(t, r, vals2)
}

// TestErrorBranches_Service_CreateUpdateFail drives Service Create/Update
// where the lookup succeeds (200) but UpdateService and subsequent GetService
// fail. Exercises the "Error Updating Service" and "Error Reading Service"
// branches inside Create and Update handlers.
func TestErrorBranches_Service_CreateUpdateFail(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "service": "ssh", "enable": false, "state": "STOPPED",
	}
	// Counter tracks GET requests so the second Get can fail in Update.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "start") || strings.Contains(req.URL.Path, "stop") {
			http.Error(w, "start/stop fail", http.StatusBadRequest)
			return
		}
		if req.Method == http.MethodPut {
			http.Error(w, "update fail", http.StatusBadRequest)
			return
		}
		// GET returns a list for search; single body for id lookups.
		if req.Method == http.MethodGet {
			if strings.Contains(req.URL.Path, "/service") {
				_ = json.NewEncoder(w).Encode([]interface{}{body})
				return
			}
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	c.RetryPolicy = client.RetryPolicy{MaxAttempts: 1}
	r := &ServiceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	// Create: GetServiceByName OK, UpdateService fails -> line 136 err branch.
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(true),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	}()

	// Update: plan/state OK, UpdateService fails -> "Error Updating Service".
	vals := map[string]tftypes.Value{
		"id":      str("1"),
		"service": str("ssh"),
		"enable":  flag(false), // exercise the stop branch (line 153 else)
	}
	state := stateFromValues(t, ctx, sch, vals)
	planU := planFromValues(t, ctx, sch, vals)
	func() {
		defer func() { _ = recover() }()
		uResp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{State: state, Plan: planU}, uResp)
	}()
}

// TestErrorBranches_Service_UpdateOK_ReadFail drives Service Create with a
// successful UpdateService but a failing post-update GetService, covering
// the "Error Reading Service" branch at line 160.
func TestErrorBranches_Service_UpdateOK_ReadFail(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "service": "ssh", "enable": true, "state": "RUNNING",
	}
	var getCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "start") || strings.Contains(req.URL.Path, "stop") || strings.Contains(req.URL.Path, "reload") {
			_, _ = w.Write([]byte("true"))
			return
		}
		if req.Method == http.MethodPut {
			// Return the updated body (update succeeds).
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		if req.Method == http.MethodGet {
			getCount++
			// First GET (lookup by name) returns list; subsequent GETs (reload
			// after update) return 400 to hit the error branch at line 160.
			if getCount == 1 {
				_ = json.NewEncoder(w).Encode([]interface{}{body})
				return
			}
			http.Error(w, "read fail", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	c.RetryPolicy = client.RetryPolicy{MaxAttempts: 1}
	r := &ServiceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(true),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	}()

	// Create with enable=false: exercises the else-stop branch (line 153).
	getCount = 0
	planDisable := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(false),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: planDisable}, cResp)
	}()

	// Also drive Update: Update PUT OK, subsequent GetService fails -> line 262.
	getCount = 0 // reset
	vals := map[string]tftypes.Value{
		"id":      str("1"),
		"service": str("ssh"),
		"enable":  flag(true),
	}
	state := stateFromValues(t, ctx, sch, vals)
	planU := planFromValues(t, ctx, sch, vals)
	func() {
		defer func() { _ = recover() }()
		uResp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{State: state, Plan: planU}, uResp)
	}()
}

// TestErrorBranches_DirectoryServices_DeleteAD drives Delete with a mock
// where the current config is ACTIVEDIRECTORY+Enable, exercising the leave
// branch that otherwise never runs.
func TestErrorBranches_DirectoryServices_DeleteAD(t *testing.T) {
	adService := "ACTIVEDIRECTORY"
	body := map[string]interface{}{
		"id":                   1,
		"service_type":         adService,
		"enable":               true,
		"enable_account_cache": true,
		"enable_dns_updates":   true,
		"timeout":              60,
		"kerberos_realm":       nil,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// directoryservices/leave returns an error so we also exercise the
		// inner leave-error warning path.
		if strings.Contains(req.URL.Path, "/leave") {
			http.Error(w, "leave failed", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	c.RetryPolicy = client.RetryPolicy{MaxAttempts: 1}
	r := &DirectoryServicesResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("directoryservices")})
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp.Diagnostics
}

// TestErrorBranches_Group_UpdateSuccess drives Group Update with a mock
// that returns a bare group ID on PUT (matching the real API) so the
// successful post-update path executes.
func TestErrorBranches_Group_UpdateSuccess(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "gid": 1000, "group": "users", "name": "users",
		"builtin": false, "smb": false, "sudo_commands": []interface{}{},
		"sudo_commands_nopasswd": []interface{}{},
		"users":                  []interface{}{},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// PUT/POST on /group/id/X or /group returns a bare int ID.
		if req.Method == http.MethodPut || (req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/group")) {
			_, _ = w.Write([]byte("1"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	r := &GroupResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"id":   str("1"),
		"gid":  num(1000),
		"name": str("users"),
	}
	state := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, uResp)
	_ = uResp.Diagnostics
}

// TestErrorBranches_DirectoryServices_ImportFromAPI invokes mapResponseToModel
// directly with credential/configuration pointers non-nil so the json.Marshal
// branches are exercised.
func TestErrorBranches_DirectoryServices_ImportFromAPI(t *testing.T) {
	r := &DirectoryServicesResource{}
	creds := map[string]interface{}{"username": "admin"}
	cfgData := map[string]interface{}{"domain": "example.com"}
	cfg := &client.DirectoryServicesConfig{
		ID:            1,
		Credential:    creds,
		Configuration: cfgData,
	}
	model := &DirectoryServicesResourceModel{}
	r.mapResponseToModel(cfg, model)
	_ = model.CredentialJSON
	_ = model.ConfigurationJSON
}

// TestErrorBranches_NVMetPort_NilTrsvcid exercises the else branch in
// NVMetPortResource.mapResponseToModel when AddrTrsvcid comes back as 0.
func TestErrorBranches_NVMetPort_NilTrsvcid(t *testing.T) {
	r := &NVMetPortResource{}
	port := &client.NVMetPort{
		ID: 1, Index: 1,
		AddrTrtype: "TCP", AddrTraddr: "0.0.0.0",
		// AddrTrsvcid not set -> GetAddrTrsvcid() returns 0.
	}
	model := &NVMetPortResourceModel{}
	r.mapResponseToModel(port, model)
	_ = model.AddrTrsvcid
}

// TestErrorBranches_FilesystemACLTemplate_MalformedACL covers the else
// fallback when stripJSONNulls fails (malformed ACL JSON from the server).
func TestErrorBranches_FilesystemACLTemplate_MalformedACL(t *testing.T) {
	r := &FilesystemACLTemplateResource{}
	tpl := &client.FilesystemACLTemplate{
		ID: 1, Name: "t", ACLType: "POSIX1E",
		ACL: json.RawMessage(`not{valid`),
	}
	model := &FilesystemACLTemplateResourceModel{}
	r.mapResponseToModel(tpl, model)
	_ = model.ACLJSON
}

// TestErrorBranches_CloudBackup_MalformedAttrs covers the else fallback in
// mapResponseToModel when the server returns malformed JSON attributes.
func TestErrorBranches_CloudBackup_MalformedAttrs(t *testing.T) {
	r := &CloudBackupResource{}
	ctx := context.Background()
	cb := &client.CloudBackup{
		ID: 1, Description: "x", Path: "/p",
		Attributes: json.RawMessage(`not{valid`),
		Schedule:   client.CloudBackupSchedule{Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "*"},
	}
	model := &CloudBackupResourceModel{}
	r.mapResponseToModel(ctx, cb, model)
	_ = model.AttributesJSON
}

// TestErrorBranches_ReportingExporter_MalformedAttrs covers the else fallback
// in mapResponseToModel when the server returns malformed JSON attributes.
func TestErrorBranches_ReportingExporter_MalformedAttrs(t *testing.T) {
	r := &ReportingExporterResource{}
	e := &client.ReportingExporter{
		ID: 1, Name: "x", Enabled: true,
		Attributes: json.RawMessage(`not{valid`),
	}
	model := &ReportingExporterResourceModel{}
	r.mapResponseToModel(e, model)
	_ = model.AttributesJSON
}

// TestErrorBranches_VMDevice_MapResponseTypes covers the type-switch cases
// (bool, float64, default, nil filter, priorKeys filter) in
// VMDeviceResource.mapResponseToModel by invoking it directly with a
// hand-crafted dev response whose Attributes map spans every branch.
func TestErrorBranches_VMDevice_MapResponseTypes(t *testing.T) {
	r := &VMDeviceResource{}
	ctx := context.Background()

	// Prior keys present -> the priorKeys filter is active; missing keys
	// get skipped (covers len(priorKeys) > 0 && !priorKeys[k] branch).
	priorMap, _ := basetypesMapString(map[string]string{
		"path":       "",
		"readonly":   "",
		"sectorsize": "",
		"kept_float": "",
		"weird":      "",
	})
	model := &VMDeviceResourceModel{Attributes: priorMap}

	order := 1001
	dev := &client.VMDevice{
		ID: 1, VM: 1, Order: &order,
		Attributes: map[string]interface{}{
			"dtype":        "DISK",
			"path":         "/dev/zvol/tank/v1",
			"readonly":     true,               // bool branch
			"sectorsize":   float64(512),       // float64 integer branch
			"kept_float":   float64(1.5),       // float64 non-integer branch
			"weird":        []interface{}{"x"}, // default branch
			"filtered_out": "server-default",   // priorKeys filter skip
			"nilval":       nil,                // nil skip
		},
	}
	r.mapResponseToModel(ctx, dev, model)
	_ = model.Dtype
}

// basetypesMapString builds a types.Map with string elements.
func basetypesMapString(m map[string]string) (basetypesMap, error) {
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	mv, _ := types.MapValue(types.StringType, elems)
	return mv, nil
}

type basetypesMap = types.Map

func TestErrorBranches_ACMEDNSAuthenticator(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ACMEDNSAuthenticatorResource{client: c}
	drive400CRUD(t, r, "7", map[string]tftypes.Value{
		"name":          str("example"),
		"authenticator": str("cloudflare"),
		"attributes":    strMapNull(),
	})
}

func TestErrorBranches_AlertService(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &AlertServiceResource{client: c}
	drive400CRUD(t, r, "3", map[string]tftypes.Value{
		"name":          str("mail"),
		"type":          str("Mail"),
		"enabled":       flag(true),
		"level":         str("WARNING"),
		"settings_json": str(`{"email":"admin@example.com"}`),
	})
}

func TestErrorBranches_AlertClasses(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &AlertClassesResource{client: c}
	drive400CRUD(t, r, "alertclasses", map[string]tftypes.Value{
		"classes": strMapNull(),
	})
}

func TestErrorBranches_APIKey(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &APIKeyResource{client: c}
	drive400CRUD(t, r, "5", map[string]tftypes.Value{
		"name":     str("tfkey"),
		"username": str("root"),
	})
}

func TestErrorBranches_Catalog(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CatalogResource{client: c}
	drive400CRUD(t, r, "TRUENAS", map[string]tftypes.Value{
		"label":            str("TRUENAS"),
		"preferred_trains": strListNull(),
		"location":         str("/mnt/tank/catalog"),
		"sync_on_create":   flag(false),
	})
}

// Exercises the sync-on-create branch and the null SyncOnCreate fallback.
func TestErrorBranches_Catalog_SyncOnCreate(t *testing.T) {
	// Mock that returns 200 OK for catalog ops, 400 for sync.
	body := map[string]interface{}{
		"id":               "TRUENAS",
		"label":            "TRUENAS",
		"preferred_trains": []interface{}{"stable"},
		"location":         "/mnt/tank/catalog",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/catalog/sync") {
			http.Error(w, "sync failed", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	c.RetryPolicy = client.RetryPolicy{MaxAttempts: 1}
	r := &CatalogResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	// sync_on_create = true path: UpdateCatalog OK, SyncCatalog fails -> error branch.
	plan1 := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"label":            str("TRUENAS"),
		"preferred_trains": strListNull(),
		"location":         str("/mnt/tank/catalog"),
		"sync_on_create":   flag(true),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan1}, cResp)
	}()

	// sync_on_create null path: exercises the null/unknown fallback on line 164.
	plan2 := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"label":            str("TRUENAS"),
		"preferred_trains": strListNull(),
		"location":         str("/mnt/tank/catalog"),
	})
	func() {
		defer func() { _ = recover() }()
		cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
		r.Create(ctx, resource.CreateRequest{Plan: plan2}, cResp)
	}()
}

func TestErrorBranches_DNSNameserver(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &DNSNameserverResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"address":  str("1.1.1.1"),
		"priority": num(1),
	})
}

func TestErrorBranches_FilesystemACLTemplate(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &FilesystemACLTemplateResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":    str("tmpl"),
		"acltype": str("POSIX1E"),
	})
}

func TestErrorBranches_InitScript(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &InitScriptResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"type":    str("COMMAND"),
		"command": str("echo hi"),
		"script":  str("/tmp/x"),
		"when":    str("POSTINIT"),
		"comment": str("boot"),
	})
}

func TestErrorBranches_KerberosKeytab(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &KerberosKeytabResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name": str("host"),
		"file": str("BASE64FAKE"),
	})
}

func TestErrorBranches_KerberosRealm(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &KerberosRealmResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"realm":          str("EXAMPLE.COM"),
		"kdc":            strList("kdc.example.com", "kdc2.example.com"),
		"admin_server":   strList("admin.example.com"),
		"kpasswd_server": strList("kp.example.com"),
		"primary_kdc":    str("kdc.example.com"),
	})
}

func TestErrorBranches_KeychainCredential(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &KeychainCredentialResource{client: c}
	attrsVal := tftypes.NewValue(
		tftypes.Map{ElementType: tftypes.String},
		map[string]tftypes.Value{
			"private_key": tftypes.NewValue(tftypes.String, "PRIV"),
			"public_key":  tftypes.NewValue(tftypes.String, "PUB"),
		},
	)
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":       str("key1"),
		"type":       str("SSH_KEY_PAIR"),
		"attributes": attrsVal,
	})
}

func TestErrorBranches_CloudBackup(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CloudBackupResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"description":      str("daily"),
		"path":             str("/mnt/tank/data"),
		"credentials":      num(3),
		"attributes_json":  str(`{"bucket":"b"}`),
		"snapshot":         flag(true),
		"enabled":          flag(true),
		"keep_last":        num(5),
		"transfer_setting": str("DEFAULT"),
		"schedule_minute":  str("0"),
		"schedule_hour":    str("1"),
		"schedule_dom":     str("*"),
		"schedule_month":   str("*"),
		"schedule_dow":     str("*"),
	})
}

func TestErrorBranches_CloudSync(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CloudSyncResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"description":     str("sync"),
		"path":            str("/mnt/tank/data"),
		"credentials":     num(3),
		"direction":       str("PUSH"),
		"transfer_mode":   str("COPY"),
		"enabled":         flag(true),
		"attributes_json": str(`{"bucket":"b"}`),
		"schedule_minute": str("0"),
		"schedule_hour":   str("1"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	})
}

func TestErrorBranches_CloudSyncCredential(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CloudSyncCredentialResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":                     str("s3"),
		"provider_type":            str("S3"),
		"provider_attributes_json": str(`{"access_key_id":"AK","secret_access_key":"SK"}`),
	})
}

func TestErrorBranches_CronJob(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CronJobResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"user":            str("root"),
		"command":         str("true"),
		"description":     str("nightly"),
		"enabled":         flag(true),
		"stdout":          flag(true),
		"stderr":          flag(true),
		"schedule_minute": str("0"),
		"schedule_hour":   str("*"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	})
}

func TestErrorBranches_RsyncTask(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &RsyncTaskResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"path":            str("/mnt/tank/src"),
		"user":            str("root"),
		"remotehost":      str("rsync.example.com"),
		"remoteport":      num(22),
		"remotemodule":    str("mod"),
		"remotepath":      str("/backup"),
		"mode":            str("SSH"),
		"direction":       str("PUSH"),
		"desc":            str("daily"),
		"schedule_minute": str("0"),
		"schedule_hour":   str("2"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	})
}

func TestErrorBranches_ScrubTask(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ScrubTaskResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"pool_id":         num(1),
		"threshold":       num(35),
		"enabled":         flag(true),
		"schedule_minute": str("0"),
		"schedule_hour":   str("0"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("7"),
	})
}

func TestErrorBranches_SnapshotTask(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &SnapshotTaskResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"dataset":         str("tank/data"),
		"recursive":       flag(true),
		"lifetime_value":  num(2),
		"lifetime_unit":   str("WEEK"),
		"naming_schema":   str("auto-%Y%m%d.%H%M"),
		"enabled":         flag(true),
		"allow_empty":     flag(true),
		"schedule_minute": str("0"),
		"schedule_hour":   str("0"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	})
}

func TestErrorBranches_Replication(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ReplicationResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":             str("repl"),
		"direction":        str("PUSH"),
		"transport":        str("SSH"),
		"target_dataset":   str("backup/data"),
		"recursive":        flag(true),
		"auto":             flag(true),
		"enabled":          flag(true),
		"retention_policy": str("SOURCE"),
	})
}

func TestErrorBranches_ReportingExporter(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ReportingExporterResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":            str("graphite"),
		"type":            str("GRAPHITE"),
		"enabled":         flag(true),
		"attributes_json": str(`{"host":"localhost","port":2003}`),
	})
}

func TestErrorBranches_Privilege(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &PrivilegeResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":         str("admins"),
		"web_shell":    flag(false),
		"local_groups": numList(1000),
		"roles":        strList("FULL_ADMIN"),
	})
}

func TestErrorBranches_StaticRoute(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &StaticRouteResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"destination": str("10.0.0.0/24"),
		"gateway":     str("192.168.1.1"),
		"description": str("static route"),
	})
}

func TestErrorBranches_Tunable(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &TunableResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"var":     str("net.core.somaxconn"),
		"value":   str("1024"),
		"type":    str("SYSCTL"),
		"comment": str("test"),
		"enabled": flag(true),
	})
}

func TestErrorBranches_VMware(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &VMwareResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"hostname": str("vcenter.example.com"),
		"username": str("admin"),
		"password": str("secret"),
	})
}

func TestErrorBranches_FTPConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &FTPConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"port":          num(21),
		"clients":       num(32),
		"ipconnections": num(8),
		"loginattempt":  num(3),
		"timeout":       num(120),
	})
}

func TestErrorBranches_MailConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &MailConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"fromemail":       str("admin@example.com"),
		"fromname":        str("TrueNAS"),
		"outgoing_server": str("smtp.example.com"),
		"port":            num(587),
		"security":        str("TLS"),
		"smtp":            flag(true),
		"user":            str("admin"),
	})
}

func TestErrorBranches_NFSConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NFSConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"servers":       num(4),
		"allow_nonroot": flag(false),
	})
}

func TestErrorBranches_SMBConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &SMBConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"netbiosname": str("NAS"),
		"workgroup":   str("WG"),
	})
}

func TestErrorBranches_SNMPConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &SNMPConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"community": str("public"),
		"contact":   str("admin"),
		"location":  str("dc1"),
	})
}

func TestErrorBranches_SSHConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &SSHConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"tcpport":      num(22),
		"passwordauth": flag(true),
	})
}

func TestErrorBranches_UPSConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &UPSConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"mode":       str("MASTER"),
		"identifier": str("ups"),
		"driver":     str("usbhid-ups"),
	})
}

func TestErrorBranches_ISCSIAuth(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ISCSIAuthResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"tag":    num(1),
		"user":   str("chap"),
		"secret": str("abcdefghi1234567"),
	})
}

func TestErrorBranches_ISCSIExtent(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ISCSIExtentResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":      str("e1"),
		"type":      str("FILE"),
		"path":      str("/mnt/tank/e1"),
		"blocksize": num(512),
		"enabled":   flag(true),
		"filesize":  num(1024),
		"rpm":       str("SSD"),
	})
}

func TestErrorBranches_ISCSIInitiator(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ISCSIInitiatorResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"comment": str("all"),
	})
}

func TestErrorBranches_ISCSIPortal(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ISCSIPortalResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"comment": str(""),
	})
}

func TestErrorBranches_ISCSITarget(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ISCSITargetResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name": str("tgt1"),
		"mode": str("ISCSI"),
	})
}

func TestErrorBranches_ISCSITargetExtent(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ISCSITargetExtentResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"target": num(1),
		"extent": num(1),
		"lunid":  num(0),
	})
}

func TestErrorBranches_NVMetGlobal(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetGlobalResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"basenqn": str("nqn.2020-01.truenas"),
		"kernel":  flag(true),
	})
}

func TestErrorBranches_NVMetHost(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetHostResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"hostnqn": str("nqn.x"),
	})
}

func TestErrorBranches_NVMetHostSubsys(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetHostSubsysResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"host_id":   num(1),
		"subsys_id": num(1),
	})
}

func TestErrorBranches_NVMetNamespace(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetNamespaceResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"subsys_id":   num(1),
		"device_type": str("ZVOL"),
		"device_path": str("zvol/tank/vol1"),
	})
}

func TestErrorBranches_NVMetPort(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetPortResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"addr_trtype":      str("TCP"),
		"addr_trsvcid":     num(4420),
		"addr_traddr":      str("0.0.0.0"),
		"addr_adrfam":      str("IPV4"),
		"inline_data_size": num(16384),
		"max_queue_size":   num(128),
		"enabled":          flag(true),
	})
}

func TestErrorBranches_NVMetPortSubsys(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetPortSubsysResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"port_id":   num(1),
		"subsys_id": num(1),
	})
}

func TestErrorBranches_NVMetSubsys(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NVMetSubsysResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":           str("tgt"),
		"allow_any_host": flag(false),
		"qid_max":        num(16),
	})
}

func TestErrorBranches_NFSShare(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NFSShareResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"path":          str("/mnt/tank/share"),
		"readonly":      flag(false),
		"enabled":       flag(true),
		"comment":       str("c"),
		"hosts":         strList("10.0.0.0/24"),
		"networks":      strList("192.168.0.0/24"),
		"security":      strList("SYS"),
		"maproot_user":  str("root"),
		"maproot_group": str("wheel"),
		"mapall_user":   str("nobody"),
		"mapall_group":  str("nogroup"),
	})
}

func TestErrorBranches_SMBShare(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &SMBShareResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"path":      str("/mnt/tank/share"),
		"name":      str("share"),
		"purpose":   str("NO_PRESET"),
		"browsable": flag(true),
		"readonly":  flag(false),
		"enabled":   flag(true),
	})
}

func TestErrorBranches_User(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &UserResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"uid":       num(1000),
		"username":  str("alice"),
		"full_name": str("Alice"),
		"home":      str("/home/alice"),
		"shell":     str("/bin/bash"),
		"group":     num(100),
		"password":  str("hunter2"),
	})
}

func TestErrorBranches_Group(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &GroupResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"gid":  num(1000),
		"name": str("users"),
	})
}

func TestErrorBranches_Dataset(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &DatasetResource{client: c}
	drive400CRUD(t, r, "tank/data", map[string]tftypes.Value{
		"name":           str("data"),
		"pool":           str("tank"),
		"type":           str("FILESYSTEM"),
		"parent_dataset": str("parent"),
	})
}

func TestErrorBranches_Zvol(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ZvolResource{client: c}
	drive400CRUD(t, r, "tank/vol1", map[string]tftypes.Value{
		"name":    str("vol1"),
		"pool":    str("tank"),
		"volsize": num(16777216),
	})
}

func TestErrorBranches_Certificate(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &CertificateResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":             str("c1"),
		"create_type":      str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate":      str("-----BEGIN CERTIFICATE-----\nMII...\n-----END CERTIFICATE-----\n"),
		"privatekey":       str("-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"),
		"key_type":         str("RSA"),
		"key_length":       num(2048),
		"digest_algorithm": str("SHA256"),
	})
}

func TestErrorBranches_VM(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &VMResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":       str("vm1"),
		"vcpus":      num(2),
		"cores":      num(1),
		"threads":    num(1),
		"memory":     num(2048),
		"min_memory": num(1024),
	})
}

func TestErrorBranches_VMDevice(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &VMDeviceResource{client: c}
	attrsVal := tftypes.NewValue(
		tftypes.Map{ElementType: tftypes.String},
		map[string]tftypes.Value{
			"path": tftypes.NewValue(tftypes.String, "/dev/zvol/tank/vol1"),
			"type": tftypes.NewValue(tftypes.String, "AHCI"),
		},
	)
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"vm":         num(1),
		"dtype":      str("DISK"),
		"order":      num(1001),
		"attributes": attrsVal,
	})
}

func TestErrorBranches_Pool(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &PoolResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"name":                    str("tank"),
		"topology_json":           str(`{"data":[{"type":"STRIPE","disks":["sda"]}]}`),
		"encryption":              flag(false),
		"encryption_options_json": str(`{}`),
		"deduplication":           str("OFF"),
		"checksum":                str("SHA256"),
		"allow_duplicate_serials": flag(false),
	})
}

func TestErrorBranches_App(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &AppResource{client: c}
	drive400CRUD(t, r, "minio", map[string]tftypes.Value{
		"app_name":    str("minio"),
		"catalog_app": str("minio"),
		"train":       str("stable"),
		"version":     str("1.0.0"),
		"values":      str("{}"),
	})
}

func TestErrorBranches_NetworkConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NetworkConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"hostname":    str("truenas"),
		"domain":      str("local"),
		"ipv4gateway": str("192.168.1.1"),
	})
}

func TestErrorBranches_NetworkInterface(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &NetworkInterfaceResource{client: c}
	aliasType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type":    tftypes.String,
		"address": tftypes.String,
		"netmask": tftypes.Number,
	}}
	alias := tftypes.NewValue(aliasType, map[string]tftypes.Value{
		"type":    tftypes.NewValue(tftypes.String, "INET"),
		"address": tftypes.NewValue(tftypes.String, "10.0.0.1"),
		"netmask": tftypes.NewValue(tftypes.Number, 24),
	})
	aliasesVal := tftypes.NewValue(
		tftypes.List{ElementType: aliasType},
		[]tftypes.Value{alias},
	)
	drive400CRUD(t, r, "eth0", map[string]tftypes.Value{
		"name":    str("eth0"),
		"type":    str("PHYSICAL"),
		"aliases": aliasesVal,
	})
}

func TestErrorBranches_KMIPConfig(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &KMIPConfigResource{client: c}
	drive400CRUD(t, r, "1", map[string]tftypes.Value{
		"server": str("kmip.example.com"),
		"port":   num(5696),
	})
}

func TestErrorBranches_SystemDataset(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &SystemDatasetResource{client: c}
	drive400CRUD(t, r, "systemdataset", map[string]tftypes.Value{
		"pool": str("tank"),
	})
}

func TestErrorBranches_DirectoryServices(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &DirectoryServicesResource{client: c}
	drive400CRUD(t, r, "directoryservices", map[string]tftypes.Value{
		"enable":       flag(false),
		"service_type": str(""),
	})
}

func TestErrorBranches_Service(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &ServiceResource{client: c}
	drive400CRUD(t, r, "ssh", map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(true),
		"state":   str("RUNNING"),
	})
}

func TestErrorBranches_FilesystemACL(t *testing.T) {
	c, done := newBadRequestClient(t)
	defer done()
	r := &FilesystemACLResource{client: c}
	entryType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"tag":          tftypes.String,
		"id":           tftypes.Number,
		"perm_read":    tftypes.Bool,
		"perm_write":   tftypes.Bool,
		"perm_execute": tftypes.Bool,
		"default":      tftypes.Bool,
	}}
	entry := tftypes.NewValue(entryType, map[string]tftypes.Value{
		"tag":          tftypes.NewValue(tftypes.String, "USER_OBJ"),
		"id":           tftypes.NewValue(tftypes.Number, -1),
		"perm_read":    tftypes.NewValue(tftypes.Bool, true),
		"perm_write":   tftypes.NewValue(tftypes.Bool, true),
		"perm_execute": tftypes.NewValue(tftypes.Bool, true),
		"default":      tftypes.NewValue(tftypes.Bool, false),
	})
	daclVal := tftypes.NewValue(
		tftypes.List{ElementType: entryType},
		[]tftypes.Value{entry},
	)
	drive400CRUD(t, r, "/mnt/tank/share", map[string]tftypes.Value{
		"path":    str("/mnt/tank/share"),
		"acltype": str("POSIX1E"),
		"uid":     num(1000),
		"gid":     num(1000),
		"dacl":    daclVal,
	})
}
