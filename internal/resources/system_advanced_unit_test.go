package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// systemAdvancedBody is what system.advanced.config returns on a fresh
// install, plus one remote UDP server.
func systemAdvancedBody() map[string]interface{} {
	return map[string]interface{}{
		"id": 1, "consolemenu": false, "serialconsole": false, "serialport": "ttyS0",
		"serialspeed": "9600", "powerdaemon": false, "overprovision": nil,
		"traceback": true, "debugkernel": false, "uploadcrash": true, "anonstats": true,
		"motd": "Welcome to TrueNAS", "login_banner": "", "boot_scrub": 7,
		"fqdn_syslog": false, "sysloglevel": "F_INFO", "syslog_audit": false,
		"kdump_enabled": false, "kernel_extra_options": "",
		"syslogservers": []map[string]interface{}{
			{"host": "192.0.2.10", "transport": "UDP", "tls_certificate": nil},
		},
	}
}

func systemAdvancedClient(t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return body, nil
		})
}

func failingSystemAdvancedClient(t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

var systemAdvancedServerTfType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"host": tftypes.String, "transport": tftypes.String, "tls_certificate": tftypes.Number,
}}

// sysServer builds one syslogservers element for a plan or config.
func sysServer(host, transport string, cert interface{}) tftypes.Value {
	return tftypes.NewValue(systemAdvancedServerTfType, map[string]tftypes.Value{
		"host":            tftypes.NewValue(tftypes.String, host),
		"transport":       tftypes.NewValue(tftypes.String, transport),
		"tls_certificate": tftypes.NewValue(tftypes.Number, cert),
	})
}

func sysServers(servers ...tftypes.Value) tftypes.Value {
	return tftypes.NewValue(tftypes.List{ElementType: systemAdvancedServerTfType}, servers)
}

func sysServerModels(servers ...SystemAdvancedSyslogServerModel) types.List {
	vals := make([]attr.Value, 0, len(servers))
	for _, s := range servers {
		vals = append(vals, types.ObjectValueMust(systemAdvancedSyslogServerAttrTypes, map[string]attr.Value{
			"host": s.Host, "transport": s.Transport, "tls_certificate": s.TLSCertificate,
		}))
	}
	return types.ListValueMust(types.ObjectType{AttrTypes: systemAdvancedSyslogServerAttrTypes}, vals)
}

func TestSystemAdvancedResource_CRUD(t *testing.T) {
	c := systemAdvancedClient(t, systemAdvancedBody())
	r := &SystemAdvancedResource{client: c}
	crudDrive(t, r, c, systemAdvancedSingletonID, map[string]tftypes.Value{
		"motd":          str("Welcome to TrueNAS"),
		"syslogservers": sysServers(sysServer("192.0.2.10", "UDP", nil)),
	})
}

// State has to hold what TrueNAS returned, not what the plan asked for.
func TestSystemAdvancedResource_CreateStoresServerValues(t *testing.T) {
	ctx := context.Background()
	body := systemAdvancedBody()
	body["sysloglevel"] = "f_debug" // an older row can be lower case
	body["overprovision"] = 10
	body["syslogservers"] = []map[string]interface{}{
		{"host": "log.example.com:6514", "transport": "TLS", "tls_certificate": 4},
		{"host": "192.0.2.10", "transport": "UDP", "tls_certificate": nil},
	}
	r := &SystemAdvancedResource{client: systemAdvancedClient(t, body)}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"motd": str("x")})
	resp := &resource.CreateResponse{State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{})}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var got SystemAdvancedResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if got.ID.ValueString() != systemAdvancedSingletonID || got.Motd.ValueString() != "Welcome to TrueNAS" {
		t.Errorf("state = %+v", got)
	}
	if got.Sysloglevel.ValueString() != "F_DEBUG" {
		t.Errorf("sysloglevel = %q, want it upper-cased", got.Sysloglevel.ValueString())
	}
	if got.Overprovision.ValueInt64() != 10 || got.BootScrub.ValueInt64() != 7 {
		t.Errorf("overprovision = %v, boot_scrub = %v", got.Overprovision, got.BootScrub)
	}
	want := sysServerModels(
		SystemAdvancedSyslogServerModel{Host: types.StringValue("log.example.com:6514"), Transport: types.StringValue("TLS"), TLSCertificate: types.Int64Value(4)},
		SystemAdvancedSyslogServerModel{Host: types.StringValue("192.0.2.10"), Transport: types.StringValue("UDP"), TLSCertificate: types.Int64Null()},
	)
	if !got.Syslogservers.Equal(want) {
		t.Errorf("syslogservers = %v, want %v (server order is kept)", got.Syslogservers, want)
	}
}

func TestSystemAdvancedResource_ReadNullOverprovision(t *testing.T) {
	ctx := context.Background()
	r := &SystemAdvancedResource{client: systemAdvancedClient(t, systemAdvancedBody())}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(systemAdvancedSingletonID)})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	var got SystemAdvancedResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	if !got.Overprovision.IsNull() {
		t.Errorf("overprovision = %v, want null as TrueNAS returns it", got.Overprovision)
	}
}

// updateRequest must send only what the plan set, and must send an empty
// server list, because that is how the servers are removed.
func TestSystemAdvancedResource_updateRequest(t *testing.T) {
	ctx := context.Background()
	r := &SystemAdvancedResource{}
	null := SystemAdvancedResourceModel{
		Syslogservers: types.ListNull(types.ObjectType{AttrTypes: systemAdvancedSyslogServerAttrTypes}),
	}

	t.Run("unset attributes are left out", func(t *testing.T) {
		var diags diag.Diagnostics
		plan := null
		plan.Motd = types.StringValue("hi")
		plan.BootScrub = types.Int64Unknown()
		plan.FQDNSyslog = types.BoolNull()
		req := r.updateRequest(ctx, &plan, &diags)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if req.Motd == nil || *req.Motd != "hi" {
			t.Errorf("motd = %v", req.Motd)
		}
		if req.BootScrub != nil || req.FQDNSyslog != nil || req.Syslogservers != nil || req.Overprovision != nil {
			t.Errorf("unset attributes were sent: %+v", req)
		}
	})

	t.Run("unknown servers are left out", func(t *testing.T) {
		var diags diag.Diagnostics
		plan := null
		plan.Syslogservers = types.ListUnknown(types.ObjectType{AttrTypes: systemAdvancedSyslogServerAttrTypes})
		if req := r.updateRequest(ctx, &plan, &diags); req.Syslogservers != nil {
			t.Errorf("unknown servers were sent: %v", *req.Syslogservers)
		}
	})

	t.Run("false and zero are real values", func(t *testing.T) {
		var diags diag.Diagnostics
		plan := null
		plan.FQDNSyslog = types.BoolValue(false)
		plan.Overprovision = types.Int64Value(0)
		req := r.updateRequest(ctx, &plan, &diags)
		if req.FQDNSyslog == nil || *req.FQDNSyslog || req.Overprovision == nil || *req.Overprovision != 0 {
			t.Errorf("request = %+v, want pointers to false and 0", req)
		}
	})

	t.Run("every attribute is mapped to its own key", func(t *testing.T) {
		var diags diag.Diagnostics
		plan := SystemAdvancedResourceModel{
			Syslogservers:      sysServerModels(),
			Sysloglevel:        types.StringValue("F_ERR"),
			FQDNSyslog:         types.BoolValue(true),
			SyslogAudit:        types.BoolValue(true),
			Motd:               types.StringValue("m"),
			LoginBanner:        types.StringValue("b"),
			Consolemenu:        types.BoolValue(true),
			Serialconsole:      types.BoolValue(true),
			Serialport:         types.StringValue("ttyS1"),
			Serialspeed:        types.StringValue("115200"),
			BootScrub:          types.Int64Value(3),
			KernelExtraOptions: types.StringValue("quiet"),
			KdumpEnabled:       types.BoolValue(true),
			Debugkernel:        types.BoolValue(true),
			Powerdaemon:        types.BoolValue(true),
			Traceback:          types.BoolValue(false),
			Uploadcrash:        types.BoolValue(false),
			Anonstats:          types.BoolValue(false),
			Overprovision:      types.Int64Value(9),
		}
		req := r.updateRequest(ctx, &plan, &diags)
		if *req.Sysloglevel != "F_ERR" || !*req.FQDNSyslog || !*req.SyslogAudit || *req.Motd != "m" ||
			*req.LoginBanner != "b" || !*req.Consolemenu || !*req.Serialconsole || *req.Serialport != "ttyS1" ||
			*req.Serialspeed != "115200" || *req.BootScrub != 3 || *req.KernelExtraOptions != "quiet" ||
			!*req.KdumpEnabled || !*req.Debugkernel || !*req.Powerdaemon || *req.Traceback ||
			*req.Uploadcrash || *req.Anonstats || *req.Overprovision != 9 {
			t.Errorf("request = %+v", req)
		}
		if req.Syslogservers == nil || len(*req.Syslogservers) != 0 {
			t.Errorf("an empty server list must be sent as an empty list, got %v", req.Syslogservers)
		}
	})

	t.Run("servers keep their order and a non-TLS certificate is null", func(t *testing.T) {
		var diags diag.Diagnostics
		plan := null
		plan.Syslogservers = sysServerModels(
			SystemAdvancedSyslogServerModel{Host: types.StringValue("b.example.com"), Transport: types.StringValue("TLS"), TLSCertificate: types.Int64Value(4)},
			SystemAdvancedSyslogServerModel{Host: types.StringValue("a.example.com"), Transport: types.StringValue("UDP"), TLSCertificate: types.Int64Null()},
		)
		req := r.updateRequest(ctx, &plan, &diags)
		got := *req.Syslogservers
		if len(got) != 2 || got[0].Host != "b.example.com" || got[1].Host != "a.example.com" {
			t.Fatalf("servers = %+v, want the configured order", got)
		}
		if got[0].TLSCertificate == nil || *got[0].TLSCertificate != 4 || got[1].TLSCertificate != nil {
			t.Errorf("certificates = %v, %v", got[0].TLSCertificate, got[1].TLSCertificate)
		}
	})
}

// syslogHostProblem applies the check system.advanced.update makes, and rejects the values
// that make it crash or misread them.
func TestSyslogHostProblem(t *testing.T) {
	cases := []struct {
		host    string
		problem string // substring, "" for valid
	}{
		{"192.0.2.10", ""},
		{"192.0.2.10:5514", ""},
		{"logs.example.com", ""},
		{"logs.example.com:65535", ""},
		{"[2001:db8::10]", ""},
		{"[2001:db8::10]:5514", ""},
		{"logs.example.com:65536", "range"},
		{"logs.example.com:abc", "number"},
		{"logs.example.com:", "number"},
		{"[2001:db8::10", "brackets"},
		{"2001:db8::10", "brackets"},
		{"", "invalid syslog server format"},
		{"has space", "invalid syslog server format"},
		{"host/path", "invalid syslog server format"},
		{"[::1]:", "invalid syslog server format"},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			got := syslogHostProblem(tc.host)
			if tc.problem == "" && got != "" {
				t.Errorf("rejected a valid host: %s", got)
			}
			if tc.problem != "" && !strings.Contains(got, tc.problem) {
				t.Errorf("problem = %q, want it to mention %q", got, tc.problem)
			}
		})
	}
}

func TestSystemAdvancedResource_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &SystemAdvancedResource{}
	sch := schemaOf(t, ctx, r)

	validate := func(vals map[string]tftypes.Value) diag.Diagnostics {
		cfg := tfsdk.Config{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, vals)}
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
		return resp.Diagnostics
	}

	cases := []struct {
		name    string
		vals    map[string]tftypes.Value
		wantErr string
	}{
		{"nothing configured", nil, ""},
		{"valid servers", map[string]tftypes.Value{"syslogservers": sysServers(
			sysServer("192.0.2.10", "UDP", nil), sysServer("log.example.com:6514", "TLS", 3))}, ""},
		{"TLS without a certificate is one-way TLS", map[string]tftypes.Value{"syslogservers": sysServers(
			sysServer("log.example.com", "TLS", nil))}, ""},
		{"empty list", map[string]tftypes.Value{"syslogservers": sysServers()}, ""},
		{"duplicate host", map[string]tftypes.Value{"syslogservers": sysServers(
			sysServer("192.0.2.10", "UDP", nil), sysServer("192.0.2.10", "TCP", nil))}, "listed more than once"},
		{"bad host", map[string]tftypes.Value{"syslogservers": sysServers(
			sysServer("2001:db8::10", "UDP", nil))}, "brackets"},
		{"certificate on UDP", map[string]tftypes.Value{"syslogservers": sysServers(
			sysServer("192.0.2.10", "UDP", 3))}, "only be set when transport"},
		{"unknown host and transport are not judged", map[string]tftypes.Value{"syslogservers": sysServers(
			tftypes.NewValue(systemAdvancedServerTfType, map[string]tftypes.Value{
				"host":            tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				"transport":       tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				"tls_certificate": tftypes.NewValue(tftypes.Number, 3),
			}))}, ""},
		{"unknown server list", map[string]tftypes.Value{"syslogservers": tftypes.NewValue(
			tftypes.List{ElementType: systemAdvancedServerTfType}, tftypes.UnknownValue)}, ""},
		{"kernel options", map[string]tftypes.Value{"kernel_extra_options": str("quiet splash")}, ""},
		{"kernel options with a quote", map[string]tftypes.Value{"kernel_extra_options": str(`a="b"`)}, "double quote"},
		{"kernel options with a newline", map[string]tftypes.Value{"kernel_extra_options": str("a\nb")}, "newline"},
		{"cgroup hierarchy", map[string]tftypes.Value{"kernel_extra_options": str("systemd.unified_cgroup_hierarchy=0")}, "unified_cgroup_hierarchy"},
		{"unknown kernel options", map[string]tftypes.Value{"kernel_extra_options": tftypes.NewValue(tftypes.String, tftypes.UnknownValue)}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := validate(tc.vals)
			if tc.wantErr == "" {
				if diags.HasError() {
					t.Errorf("unexpected error: %v", diags)
				}
				return
			}
			if !diags.HasError() || !strings.Contains(diags.Errors()[0].Detail(), tc.wantErr) {
				t.Errorf("diags = %v, want an error mentioning %q", diags, tc.wantErr)
			}
		})
	}

	t.Run("an undecodable config stops", func(t *testing.T) {
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{
			Config: tfsdk.Config{Schema: sch.Schema, Raw: tftypes.NewValue(tftypes.String, "x")},
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ValidateConfig accepted a config it could not decode")
		}
	})
}

// The schema validators keep out what TrueNAS rejects, so the error comes at
// plan time. Each case sets one attribute to a value TrueNAS refuses.
func TestSystemAdvancedResource_SchemaValidators(t *testing.T) {
	ctx := context.Background()
	attrs := schemaOf(t, ctx, &SystemAdvancedResource{}).Schema.Attributes

	stringCases := []struct {
		attr    string
		value   string
		wantErr bool
	}{
		{"sysloglevel", "F_INFO", false},
		{"sysloglevel", "F_DEBUG", false},
		{"sysloglevel", "info", true},
		{"sysloglevel", "F_IS_DEBUG", true},
		{"serialspeed", "115200", false},
		{"serialspeed", "4800", true},
		{"motd", strings.Repeat("a", 1024), false},
		{"motd", strings.Repeat("a", 1025), true},
		{"login_banner", strings.Repeat("a", 4096), false},
		{"login_banner", strings.Repeat("a", 4097), true},
		{"serialport", strings.Repeat("a", 1025), true},
		{"kernel_extra_options", strings.Repeat("a", 1025), true},
	}
	for _, tc := range stringCases {
		t.Run(tc.attr+" "+tc.value[:min(len(tc.value), 12)], func(t *testing.T) {
			a, ok := attrs[tc.attr].(schema.StringAttribute)
			if !ok || len(a.Validators) == 0 {
				t.Fatalf("%s: %T with %d validators", tc.attr, attrs[tc.attr], len(a.Validators))
			}
			resp := &validator.StringResponse{}
			for _, v := range a.Validators {
				v.ValidateString(ctx, validator.StringRequest{Path: path.Root(tc.attr), ConfigValue: types.StringValue(tc.value)}, resp)
			}
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError = %v, want %v", got, tc.wantErr)
			}
		})
	}

	intCases := []struct {
		attr    string
		value   int64
		wantErr bool
	}{
		{"boot_scrub", 1, false},
		{"boot_scrub", 0, true}, // PositiveInt
		{"overprovision", 0, false},
		{"overprovision", -1, true},
	}
	for _, tc := range intCases {
		t.Run(tc.attr, func(t *testing.T) {
			a, ok := attrs[tc.attr].(schema.Int64Attribute)
			if !ok {
				t.Fatalf("%s is %T", tc.attr, attrs[tc.attr])
			}
			resp := &validator.Int64Response{}
			for _, v := range a.Validators {
				v.ValidateInt64(ctx, validator.Int64Request{Path: path.Root(tc.attr), ConfigValue: types.Int64Value(tc.value)}, resp)
			}
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError = %v, want %v", got, tc.wantErr)
			}
		})
	}

	list, ok := attrs["syslogservers"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("syslogservers is %T", attrs["syslogservers"])
	}
	one := SystemAdvancedSyslogServerModel{Host: types.StringValue("a"), Transport: types.StringValue("UDP"), TLSCertificate: types.Int64Null()}
	for n, wantErr := range map[int]bool{0: false, 2: false, 3: true} {
		servers := make([]SystemAdvancedSyslogServerModel, n)
		for i := range servers {
			servers[i] = one
		}
		resp := &validator.ListResponse{}
		for _, v := range list.Validators {
			v.ValidateList(ctx, validator.ListRequest{Path: path.Root("syslogservers"), ConfigValue: sysServerModels(servers...)}, resp)
		}
		if got := resp.Diagnostics.HasError(); got != wantErr {
			t.Errorf("%d servers: HasError = %v, want %v", n, got, wantErr)
		}
	}

	nested, ok := list.NestedObject.Attributes["transport"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("transport is %T", list.NestedObject.Attributes["transport"])
	}
	for value, wantErr := range map[string]bool{"UDP": false, "TCP": false, "TLS": false, "udp": true, "RELP": true} {
		resp := &validator.StringResponse{}
		for _, v := range nested.Validators {
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("transport"), ConfigValue: types.StringValue(value)}, resp)
		}
		if got := resp.Diagnostics.HasError(); got != wantErr {
			t.Errorf("transport %q: HasError = %v, want %v", value, got, wantErr)
		}
	}
	host, ok := list.NestedObject.Attributes["host"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("host is %T", list.NestedObject.Attributes["host"])
	}
	resp := &validator.StringResponse{}
	for _, v := range host.Validators {
		v.ValidateString(ctx, validator.StringRequest{Path: path.Root("host"), ConfigValue: types.StringValue("")}, resp)
	}
	if !resp.Diagnostics.HasError() {
		t.Error("an empty host was accepted")
	}
}

func TestSystemAdvancedResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &SystemAdvancedResource{client: systemAdvancedClient(t, systemAdvancedBody())}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{})

	ok := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: systemAdvancedSingletonID}, ok)
	if ok.Diagnostics.HasError() {
		t.Errorf("the canonical import ID was rejected: %v", ok.Diagnostics)
	}
	var id types.String
	if diags := ok.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() || id.ValueString() != systemAdvancedSingletonID {
		t.Errorf("imported id = %v (%v)", id, diags)
	}

	bad := &resource.ImportStateResponse{State: st}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("ImportState accepted an ID other than the singleton's")
	}
}

// A failed write or read has to surface as a diagnostic.
func TestSystemAdvancedResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &SystemAdvancedResource{client: failingSystemAdvancedClient(t)}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{"id": str(systemAdvancedSingletonID), "motd": str("x")}
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
func TestSystemAdvancedResource_DeleteSendsNothing(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := newWSTestClient(ctx, t,
		func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			calls++
			return systemAdvancedBody(), nil
		})
	r := &SystemAdvancedResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(systemAdvancedSingletonID)})

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
func TestSystemAdvancedResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &SystemAdvancedResource{client: failingSystemAdvancedClient(t)}
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
