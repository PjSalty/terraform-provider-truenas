package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// newTestServerClient creates an httptest server with the given handler
// and a *wsclient.Client pointing at it. Returns the server so tests can
// close it.
func newTestServerClient(t *testing.T, handler http.HandlerFunc) (*wsclient.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c, err := wsclient.NewWithOptions(srv.URL, "test-api-key", true)
	if err != nil {
		srv.Close()
		t.Fatalf("wsclient.New: %v", err)
	}
	return c, srv
}

// newWSConfigServerClient builds a wsclient.TestServer that serves the
// canonical config-singleton method pair — "<svc>.config" returns the
// supplied object, "<svc>.update" increments *updateCalls and returns
// the same object — and a connected *wsclient.Client. This is the
// JSON-RPC equivalent of the REST-era GET/PUT httptest handler, used
// to un-skip the config CRUD unit tests after the v2.0 WS cutover.
func newWSConfigServerClient(t *testing.T, svc string, resp map[string]interface{}, updateCalls *int) *wsclient.Client {
	t.Helper()
	ts := wsclient.NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case svc + ".config":
			return resp, nil
		case svc + ".update":
			if updateCalls != nil {
				*updateCalls++
			}
			return resp, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(context.Background())
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c
}

// newWSEntityServerClient builds a wsclient.TestServer for the
// id-addressed entity pattern: "<ns>.get_instance" and "<ns>.query"
// return the supplied object, "<ns>.delete" returns true. JSON-RPC
// equivalent of the REST-era GET-by-id/DELETE httptest handler.
func newWSEntityServerClient(t *testing.T, ns string, obj map[string]interface{}) *wsclient.Client {
	t.Helper()
	ts := wsclient.NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case ns + ".get_instance":
			return obj, nil
		case ns + ".query":
			return []interface{}{obj}, nil
		case ns + ".delete":
			return true, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(context.Background())
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c
}

// primedState builds a tfsdk.State with the given schema and a Raw
// tftypes.Value where every attribute is null. The resulting state is
// safely decodable into a struct via State.Get(), and the framework's
// subsequent State.Set() replaces it wholesale.
func primedState(t *testing.T, ctx context.Context, schemaRes resource.SchemaResponse) tfsdk.State {
	t.Helper()
	typ := schemaRes.Schema.Type().TerraformType(ctx)
	objType, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is not an object: %T", typ)
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(at, nil)
	}
	return tfsdk.State{Schema: schemaRes.Schema, Raw: tftypes.NewValue(objType, vals)}
}

// --- FTPConfig CRUD roundtrip ---

func TestFTPConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	// Fake TrueNAS response for /ftp
	resp := map[string]interface{}{
		"id":            1,
		"port":          21,
		"clients":       32,
		"ipconnections": 8,
		"loginattempt":  3,
		"timeout":       120,
		"onlyanonymous": false,
		"onlylocal":     false,
		"banner":        "TrueNAS FTP",
		"filemask":      "077",
		"dirmask":       "077",
		"fxp":           false,
		"resume":        false,
		"defaultroot":   true,
		"tls":           false,
	}

	var updateCalls int
	c := newWSConfigServerClient(t, "ftp", resp, &updateCalls)

	r := &FTPConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	t.Run("read", func(t *testing.T) {
		state := primedState(t, ctx, *sch)
		readResp := &resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, readResp)
		if readResp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", readResp.Diagnostics)
		}
	})
}

// --- SMBConfig CRUD roundtrip ---

func TestSMBConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":              1,
		"netbiosname":     "TRUENAS",
		"workgroup":       "WORKGROUP",
		"description":     "TrueNAS SMB",
		"enable_smb1":     false,
		"unixcharset":     "UTF-8",
		"aapl_extensions": false,
		"guest":           "nobody",
		"filemask":        "0775",
		"dirmask":         "0775",
	}
	c := newWSConfigServerClient(t, "smb", resp, nil)

	r := &SMBConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// --- SNMPConfig CRUD roundtrip ---

func TestSNMPConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":                1,
		"community":         "public",
		"contact":           "admin",
		"location":          "dc1",
		"v3":                false,
		"v3_username":       "",
		"v3_authtype":       "",
		"v3_password":       "",
		"v3_privproto":      nil,
		"v3_privpassphrase": nil,
	}
	c := newWSConfigServerClient(t, "snmp", resp, nil)

	r := &SNMPConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// --- UPSConfig CRUD roundtrip ---

func TestUPSConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":            1,
		"mode":          "MASTER",
		"identifier":    "ups",
		"driver":        "usbhid-ups",
		"port":          "auto",
		"remotehost":    "",
		"remoteport":    3493,
		"shutdown":      "BATT",
		"shutdowntimer": 30,
		"description":   "primary",
	}
	c := newWSConfigServerClient(t, "ups", resp, nil)

	r := &UPSConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// --- MailConfig CRUD roundtrip ---

func TestMailConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":             1,
		"fromemail":      "admin@example.com",
		"fromname":       "TrueNAS",
		"outgoingserver": "smtp.example.com",
		"port":           587,
		"security":       "TLS",
		"smtp":           true,
		"user":           "admin",
		"pass":           "",
	}
	c := newWSConfigServerClient(t, "mail", resp, nil)

	r := &MailConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// --- SSHConfig CRUD roundtrip ---

func TestSSHConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":                1,
		"tcpport":           22,
		"passwordauth":      true,
		"kerberosauth":      false,
		"tcpfwd":            true,
		"compression":       false,
		"sftp_log_level":    "",
		"sftp_log_facility": "",
		"weak_ciphers":      []string{},
	}
	c := newWSConfigServerClient(t, "ssh", resp, nil)

	r := &SSHConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// --- NFSConfig CRUD roundtrip ---

func TestNFSConfigResource_CRUD(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":            1,
		"servers":       4,
		"allow_nonroot": false,
		"protocols":     []string{"NFSV3", "NFSV4"},
		"v4_krb":        false,
		"v4_domain":     "",
		"bindip":        []string{},
		"mountd_port":   nil,
		"rpcstatd_port": nil,
		"rpclockd_port": nil,
	}
	c := newWSConfigServerClient(t, "nfs", resp, nil)

	r := &NFSConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// primedStateWithID builds a tfsdk.State with the id attribute set to the
// given string value. All other attributes remain null. This is used for
// ID-based resources whose Read/Update/Delete handlers parse state.ID.
func primedStateWithID(t *testing.T, ctx context.Context, schemaRes resource.SchemaResponse, id string) tfsdk.State {
	t.Helper()
	typ := schemaRes.Schema.Type().TerraformType(ctx)
	objType := typ.(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		if name == "id" {
			vals[name] = tftypes.NewValue(at, id)
			continue
		}
		vals[name] = tftypes.NewValue(at, nil)
	}
	return tfsdk.State{Schema: schemaRes.Schema, Raw: tftypes.NewValue(objType, vals)}
}

// primedPlan builds a tfsdk.Plan mirroring primedState. Singleton Update
// handlers read the plan and call the API — a null-attribute plan lets the
// Update path run through without requiring field-level data.
func primedPlan(t *testing.T, ctx context.Context, schemaRes resource.SchemaResponse) tfsdk.Plan {
	t.Helper()
	typ := schemaRes.Schema.Type().TerraformType(ctx)
	objType := typ.(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(at, nil)
	}
	return tfsdk.Plan{Schema: schemaRes.Schema, Raw: tftypes.NewValue(objType, vals)}
}

// Singleton Update/Delete handlers — drive them with the same test server
// so we get update+delete coverage on top of the Read coverage above.

func TestFTPConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "port": 21, "clients": 32, "ipconnections": 8, "loginattempt": 3,
		"timeout": 120, "onlyanonymous": false, "onlylocal": false, "banner": "x",
		"filemask": "077", "dirmask": "077", "fxp": false, "resume": false,
		"defaultroot": true, "tls": false,
	}
	c := newWSConfigServerClient(t, "ftp", resp, nil)

	r := &FTPConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	t.Run("update", func(t *testing.T) {
		state := primedState(t, ctx, *sch)
		plan := primedPlan(t, ctx, *sch)
		uResp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
		if uResp.Diagnostics.HasError() {
			t.Errorf("Update: %v", uResp.Diagnostics)
		}
	})
	t.Run("delete", func(t *testing.T) {
		state := primedState(t, ctx, *sch)
		dResp := &resource.DeleteResponse{State: state}
		r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
		if dResp.Diagnostics.HasError() {
			t.Errorf("Delete: %v", dResp.Diagnostics)
		}
	})
}

func TestSMBConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "netbiosname": "NAS", "workgroup": "WG",
		"description": "", "enable_smb1": false, "unixcharset": "UTF-8",
		"aapl_extensions": false, "guest": "nobody", "filemask": "0775", "dirmask": "0775",
	}
	c := newWSConfigServerClient(t, "smb", resp, nil)

	r := &SMBConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	// Delete may be a no-op; just ensure it doesn't panic.
	_ = dResp
}

func TestSNMPConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "community": "public", "contact": "a", "location": "l",
		"v3": false, "v3_username": "", "v3_authtype": "", "v3_password": "",
		"v3_privproto": nil, "v3_privpassphrase": nil,
	}
	c := newWSConfigServerClient(t, "snmp", resp, nil)

	r := &SNMPConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp
}

func TestUPSConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "mode": "MASTER", "identifier": "ups", "driver": "d",
		"port": "auto", "remotehost": "", "remoteport": 3493,
		"shutdown": "BATT", "shutdowntimer": 30, "description": "",
	}
	c := newWSConfigServerClient(t, "ups", resp, nil)

	r := &UPSConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp
}

func TestMailConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "fromemail": "a@b.c", "fromname": "N", "outgoingserver": "smtp",
		"port": 587, "security": "TLS", "smtp": true, "user": "u", "pass": "",
	}
	c := newWSConfigServerClient(t, "mail", resp, nil)

	r := &MailConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp
}

func TestSSHConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "tcpport": 22, "passwordauth": true, "kerberosauth": false,
		"tcpfwd": true, "compression": false, "sftp_log_level": "",
		"sftp_log_facility": "", "weak_ciphers": []string{},
	}
	c := newWSConfigServerClient(t, "ssh", resp, nil)

	r := &SSHConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp
}

func TestNFSConfigResource_UpdateDelete(t *testing.T) {
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "servers": 4, "allow_nonroot": false,
		"protocols": []string{"NFSV3"}, "v4_krb": false, "v4_domain": "",
		"bindip": []string{}, "mountd_port": nil, "rpcstatd_port": nil, "rpclockd_port": nil,
	}
	c := newWSConfigServerClient(t, "nfs", resp, nil)

	r := &NFSConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp
}

func TestNVMetGlobalResource_UpdateDelete(t *testing.T) {
	skipWSCutover(t)
	ctx := context.Background()
	resp := map[string]interface{}{
		"id": 1, "basenqn": "nqn.x", "kernel": true, "ana": false,
		"rdma": false, "xport_referral": true,
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()

	r := &NVMetGlobalResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	plan := primedPlan(t, ctx, *sch)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Errorf("Update: %v", uResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	_ = dResp
}

// --- ID-based resources: Read + Delete drive paths through client API ---

func TestISCSIAuthResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "iscsi.auth", map[string]interface{}{
		"id": 1, "tag": 1, "user": "chap", "secret": "[REDACTED]",
		"peeruser": "", "peersecret": "", "discovery_auth": "NONE",
	})
	r := &ISCSIAuthResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestISCSIExtentResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "iscsi.extent", map[string]interface{}{
		"id": 1, "name": "e1", "type": "FILE", "path": "/mnt/tank/e1",
		"blocksize": 512, "enabled": true, "comment": "",
		"ro": false, "xen": false, "insecure_tpc": false, "filesize": "0",
	})
	r := &ISCSIExtentResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestISCSIInitiatorResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "iscsi.initiator", map[string]interface{}{
		"id": 1, "initiators": []string{}, "comment": "all",
	})
	r := &ISCSIInitiatorResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestISCSIPortalResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "iscsi.portal", map[string]interface{}{
		"id": 1, "tag": 1, "comment": "",
		"listen": []map[string]interface{}{{"ip": "0.0.0.0", "port": 3260}},
	})
	r := &ISCSIPortalResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestISCSITargetResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "iscsi.target", map[string]interface{}{
		"id": 1, "name": "tgt1", "alias": "", "mode": "ISCSI", "groups": []interface{}{},
	})
	r := &ISCSITargetResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestISCSITargetExtentResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "iscsi.targetextent", map[string]interface{}{
		"id": 1, "target": 1, "extent": 1, "lunid": 0,
	})
	r := &ISCSITargetExtentResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestUserResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "user", map[string]interface{}{
		"id": 1, "uid": 1000, "username": "alice", "full_name": "Alice",
		"email": nil, "home": "/home/alice", "shell": "/bin/bash",
		"locked": false, "smb": true, "group": map[string]interface{}{"id": 100, "bsdgrp_gid": 100},
		"groups": []int{}, "sudo_commands": []string{}, "sshpubkey": nil,
	})
	r := &UserResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestGroupResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "group", map[string]interface{}{
		"id": 1, "gid": 1000, "group": "users", "name": "users",
		"builtin": false, "smb": false, "sudo_commands": []string{},
	})
	r := &GroupResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestNFSShareResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "sharing.nfs", map[string]interface{}{
		"id": 1, "path": "/mnt/tank", "comment": "",
		"hosts": []string{}, "networks": []string{}, "security": []string{},
		"readonly": false, "enabled": true,
		"maproot_user": "", "maproot_group": "", "mapall_user": "", "mapall_group": "",
	})
	r := &NFSShareResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestSMBShareResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "sharing.smb", map[string]interface{}{
		"id": 1, "path": "/mnt/tank", "name": "tank", "comment": "",
		"purpose": "NO_PRESET", "browsable": true, "readonly": false,
		"abe": false, "enabled": true, "hostsallow": []string{}, "hostsdeny": []string{},
	})
	r := &SMBShareResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestNVMetHostResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "nvmet.host", map[string]interface{}{
		"id": 1, "hostnqn": "nqn.x",
	})
	r := &NVMetHostResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

func TestNVMetSubsysResource_ReadDelete(t *testing.T) {
	ctx := context.Background()
	c := newWSEntityServerClient(t, "nvmet.subsys", map[string]interface{}{
		"id": 1, "name": "tgt", "allow_any_host": false, "serial": "SN",
	})
	r := &NVMetSubsysResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	st := primedStateWithID(t, ctx, *sch, "1")
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Errorf("Read: %v", rResp.Diagnostics)
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
}

// --- NVMetGlobal CRUD roundtrip ---

func TestNVMetGlobalResource_CRUD(t *testing.T) {
	skipWSCutover(t)
	ctx := context.Background()
	resp := map[string]interface{}{
		"id":             1,
		"basenqn":        "nqn.2020-01.truenas",
		"kernel":         true,
		"ana":            false,
		"rdma":           false,
		"xport_referral": true,
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()

	r := &NVMetGlobalResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
}

// skipWSCutover skips unit tests that historically mocked the REST
// transport via httptest. The v2.0 cutover to JSON-RPC over WebSocket
// retired the REST path that these tests bind against; equivalent
// typed-call coverage now lives in internal/wsclient/*_test.go (which
// uses internal/wsclient/testserver.go for the WS fixture). Rewriting
// the resource-layer mocks to bind against wsclient's testserver is
// tracked as v2.x polish — the acc suite already exercises the WS
// path end-to-end against a live TrueNAS for every resource.
func skipWSCutover(t *testing.T) {
	t.Helper()
	t.Skip("v2.0 WS cutover: REST httptest fixtures no longer valid; equivalent typed-call coverage at internal/wsclient/*_test.go; resource-layer wsclient testserver rewrite tracked as v2.x polish")
}
