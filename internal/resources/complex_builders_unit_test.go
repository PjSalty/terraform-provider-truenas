package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --- AlertClasses ---

func TestAlertClassesResource_MapResponseToModel(t *testing.T) {
	r := &AlertClassesResource{}
	ctx := context.Background()
	proactiveTrue := true
	cases := []*client.AlertClassesConfig{
		{ID: 1, Classes: map[string]client.AlertClassEntry{
			"SystemPoolCritical": {Level: "CRITICAL", Policy: "IMMEDIATELY", ProactiveSupport: &proactiveTrue},
			"DataPoolOffline":    {Level: "WARNING", Policy: "HOURLY"},
		}},
		{ID: 1, Classes: map[string]client.AlertClassEntry{
			"UPSOnBattery": {Level: "INFO", Policy: "IMMEDIATELY"},
		}},
		{ID: 1, Classes: map[string]client.AlertClassEntry{}},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m AlertClassesResourceModel
			r.mapResponseToModel(ctx, cfg, &m, nil)
			if m.ID.ValueString() != "alertclasses" {
				t.Errorf("case %d: ID should be alertclasses", i)
			}
			if got := len(m.Classes.Elements()); got != len(cfg.Classes) {
				t.Errorf("case %d: Classes len = %d, want %d", i, got, len(cfg.Classes))
			}
		})
	}
}

func TestAlertClassesResource_BuildUpdateRequest(t *testing.T) {
	r := &AlertClassesResource{}
	ctx := context.Background()
	t.Run("with classes", func(t *testing.T) {
		entry, d := types.ObjectValue(alertClassEntryAttrTypes, map[string]attr.Value{
			"level":             types.StringValue("WARNING"),
			"policy":            types.StringValue("HOURLY"),
			"proactive_support": types.BoolValue(true),
		})
		if d.HasError() {
			t.Fatalf("ObjectValue: %v", d)
		}
		m, d := types.MapValue(types.ObjectType{AttrTypes: alertClassEntryAttrTypes}, map[string]attr.Value{
			"DataPoolOffline": entry,
		})
		if d.HasError() {
			t.Fatalf("MapValue: %v", d)
		}
		plan := &AlertClassesResourceModel{Classes: m}
		req := r.buildUpdateRequest(ctx, plan)
		if len(req.Classes) != 1 {
			t.Errorf("Classes len = %d, want 1", len(req.Classes))
		}
		if c, ok := req.Classes["DataPoolOffline"]; !ok || c.Level != "WARNING" {
			t.Errorf("DataPoolOffline entry mismatch: %+v", c)
		}
	})
	t.Run("null classes", func(t *testing.T) {
		plan := &AlertClassesResourceModel{
			Classes: types.MapNull(types.ObjectType{AttrTypes: alertClassEntryAttrTypes}),
		}
		req := r.buildUpdateRequest(ctx, plan)
		if len(req.Classes) != 0 {
			t.Errorf("expected empty classes")
		}
	})
	t.Run("empty map", func(t *testing.T) {
		m, _ := types.MapValue(types.ObjectType{AttrTypes: alertClassEntryAttrTypes}, map[string]attr.Value{})
		plan := &AlertClassesResourceModel{Classes: m}
		req := r.buildUpdateRequest(ctx, plan)
		if len(req.Classes) != 0 {
			t.Errorf("expected empty classes")
		}
	})
}

// --- FilesystemACL mapResponseToModel ---

func TestFilesystemACLResource_MapResponseToModel(t *testing.T) {
	r := &FilesystemACLResource{}
	ctx := context.Background()
	who := "root"
	cases := []*client.FilesystemACL{
		{
			Path: "/mnt/tank/data", UID: 0, GID: 0, ACLType: "POSIX1E",
			User: &who, Group: &who,
			ACL: []client.ACLEntry{
				{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: true, Execute: true}},
				{Tag: "GROUP_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: false, Execute: true}},
				{Tag: "OTHER", ID: -1, Perms: client.ACLPerms{Read: true}},
			},
		},
		{
			Path: "/mnt/tank/empty", UID: 1000, GID: 1000, ACLType: "POSIX1E",
		},
		{
			Path: "/mnt/tank/nfs4", UID: 100, GID: 100, ACLType: "NFS4",
			ACL: []client.ACLEntry{
				{Tag: "everyone@", ID: -1, Perms: client.ACLPerms{Read: true}, Default: false},
			},
		},
	}
	for i, acl := range cases {
		t.Run("case", func(t *testing.T) {
			var m FilesystemACLResourceModel
			var d diag.Diagnostics
			r.mapResponseToModel(ctx, acl, &m, &d)
			if m.Path.ValueString() != acl.Path {
				t.Errorf("case %d: Path mismatch", i)
			}
			if m.ACLType.ValueString() != acl.ACLType {
				t.Errorf("case %d: ACLType mismatch", i)
			}
			if m.UID.ValueInt64() != int64(acl.UID) {
				t.Errorf("case %d: UID mismatch", i)
			}
			if got := len(m.DACL.Elements()); got != len(acl.ACL) {
				t.Errorf("case %d: DACL len = %d, want %d", i, got, len(acl.ACL))
			}
		})
	}
}

// --- DirectoryServices buildUpdateRequest ---

func TestDirectoryServicesResource_BuildUpdateRequest(t *testing.T) {
	r := &DirectoryServicesResource{}
	t.Run("populated ad", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &DirectoryServicesResourceModel{
			ServiceType:        types.StringValue("ACTIVEDIRECTORY"),
			Enable:             types.BoolValue(true),
			EnableAccountCache: types.BoolValue(true),
			EnableDNSUpdates:   types.BoolValue(false),
			Timeout:            types.Int64Value(60),
			KerberosRealm:      types.StringValue("CORP.EXAMPLE.COM"),
			CredentialJSON:     types.StringValue(`{"credential_type":"KERBEROS_USER","username":"admin","password":"x"}`),
			ConfigurationJSON:  types.StringValue(`{"hostname":"dc1.corp.example.com","domain":"corp.example.com"}`),
			Force:              types.BoolValue(false),
		}
		req := r.buildUpdateRequest(plan, &d)
		if d.HasError() {
			t.Fatalf("diags: %v", d)
		}
		if req.ServiceType == nil || *req.ServiceType != "ACTIVEDIRECTORY" {
			t.Errorf("ServiceType mismatch")
		}
		if req.Enable == nil || !*req.Enable {
			t.Errorf("Enable mismatch")
		}
		if req.Timeout == nil || *req.Timeout != 60 {
			t.Errorf("Timeout mismatch")
		}
		if req.KerberosRealm == nil || *req.KerberosRealm != "CORP.EXAMPLE.COM" {
			t.Errorf("KerberosRealm mismatch")
		}
		if req.Credential == nil {
			t.Errorf("Credential should be set")
		}
		if req.Configuration == nil {
			t.Errorf("Configuration should be set")
		}
	})
	t.Run("all null", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &DirectoryServicesResourceModel{
			ServiceType:       types.StringNull(),
			Enable:            types.BoolNull(),
			CredentialJSON:    types.StringNull(),
			ConfigurationJSON: types.StringNull(),
		}
		req := r.buildUpdateRequest(plan, &d)
		if req.ServiceType != nil || req.Enable != nil || req.Credential != nil {
			t.Errorf("null fields should be skipped")
		}
	})
	t.Run("empty realm string skipped", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &DirectoryServicesResourceModel{
			KerberosRealm: types.StringValue(""),
		}
		req := r.buildUpdateRequest(plan, &d)
		if req.KerberosRealm != nil {
			t.Errorf("empty realm should be skipped")
		}
	})
	t.Run("invalid credential JSON produces diagnostic", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &DirectoryServicesResourceModel{
			CredentialJSON: types.StringValue(`{bad`),
		}
		r.buildUpdateRequest(plan, &d)
		if !d.HasError() {
			t.Errorf("expected error for invalid JSON")
		}
	})
	t.Run("invalid configuration JSON produces diagnostic", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &DirectoryServicesResourceModel{
			ConfigurationJSON: types.StringValue(`{bad`),
		}
		r.buildUpdateRequest(plan, &d)
		if !d.HasError() {
			t.Errorf("expected error for invalid JSON")
		}
	})
}

// --- NetworkConfig buildUpdateRequest ---

func TestNetworkConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &NetworkConfigResource{}
	ctx := context.Background()
	t.Run("all fields populated", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &NetworkConfigResourceModel{
			Hostname:    types.StringValue("truenas"),
			Domain:      types.StringValue("corp.example.com"),
			IPv4Gateway: types.StringValue("10.0.0.1"),
			IPv6Gateway: types.StringValue("fe80::1"),
			Nameserver1: types.StringValue("1.1.1.1"),
			Nameserver2: types.StringValue("8.8.8.8"),
			Nameserver3: types.StringValue("9.9.9.9"),
			HTTPProxy:   types.StringValue("http://proxy:8080"),
			Hosts:       stringListValue(t, []string{"10.0.0.1 gw", "10.0.0.2 switch"}),
		}
		req := r.buildUpdateRequest(ctx, plan, &d)
		if req.Hostname == nil || *req.Hostname != "truenas" {
			t.Errorf("Hostname mismatch")
		}
		if req.Domain == nil || *req.Domain != "corp.example.com" {
			t.Errorf("Domain mismatch")
		}
		if len(req.Hosts) != 2 {
			t.Errorf("Hosts len = %d", len(req.Hosts))
		}
	})
	t.Run("all null", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &NetworkConfigResourceModel{
			Hostname:    types.StringNull(),
			Domain:      types.StringNull(),
			IPv4Gateway: types.StringNull(),
			Hosts:       types.ListNull(types.StringType),
		}
		req := r.buildUpdateRequest(ctx, plan, &d)
		if req.Hostname != nil || req.Domain != nil || req.Hosts != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}

// --- FilesystemACL buildSetRequest ---

func TestFilesystemACLResource_BuildSetRequest(t *testing.T) {
	r := &FilesystemACLResource{}
	ctx := context.Background()
	t.Run("populated dacl", func(t *testing.T) {
		entry1, d := types.ObjectValue(aclEntryAttrTypes, map[string]attr.Value{
			"tag":          types.StringValue("USER_OBJ"),
			"id":           types.Int64Value(-1),
			"perm_read":    types.BoolValue(true),
			"perm_write":   types.BoolValue(true),
			"perm_execute": types.BoolValue(true),
			"default":      types.BoolValue(false),
		})
		if d.HasError() {
			t.Fatalf("ObjectValue: %v", d)
		}
		entry2, _ := types.ObjectValue(aclEntryAttrTypes, map[string]attr.Value{
			"tag":          types.StringValue("OTHER"),
			"id":           types.Int64Value(-1),
			"perm_read":    types.BoolValue(true),
			"perm_write":   types.BoolValue(false),
			"perm_execute": types.BoolValue(false),
			"default":      types.BoolValue(false),
		})
		dacl, _ := types.ListValue(types.ObjectType{AttrTypes: aclEntryAttrTypes}, []attr.Value{entry1, entry2})
		plan := &FilesystemACLResourceModel{
			Path:    types.StringValue("/mnt/tank/data"),
			ACLType: types.StringValue("POSIX1E"),
			UID:     types.Int64Value(1000),
			GID:     types.Int64Value(1000),
			DACL:    dacl,
		}
		req, _ := r.buildSetRequest(ctx, plan)
		if req == nil {
			t.Fatalf("nil request")
		}
		if req.Path != "/mnt/tank/data" {
			t.Errorf("Path mismatch")
		}
		if req.UID == nil || *req.UID != 1000 {
			t.Errorf("UID mismatch")
		}
		if len(req.DACL) != 2 {
			t.Errorf("DACL len = %d, want 2", len(req.DACL))
		}
		if req.DACL[0].Tag != "USER_OBJ" || !req.DACL[0].Perms.Read {
			t.Errorf("DACL[0] mismatch: %+v", req.DACL[0])
		}
	})
	t.Run("null uid/gid", func(t *testing.T) {
		empty, _ := types.ListValue(types.ObjectType{AttrTypes: aclEntryAttrTypes}, []attr.Value{})
		plan := &FilesystemACLResourceModel{
			Path:    types.StringValue("/mnt/tank/none"),
			ACLType: types.StringValue("POSIX1E"),
			UID:     types.Int64Null(),
			GID:     types.Int64Null(),
			DACL:    empty,
		}
		req, _ := r.buildSetRequest(ctx, plan)
		if req.UID != nil || req.GID != nil {
			t.Errorf("UID/GID should be nil for null")
		}
		if len(req.DACL) != 0 {
			t.Errorf("DACL should be empty")
		}
	})
}

// --- network_interface aliasesFromList ---

func TestAliasesFromList(t *testing.T) {
	ctx := context.Background()
	ot := aliasObjectType()
	t.Run("null list returns ok=false", func(t *testing.T) {
		var diags diag.Diagnostics
		_, ok := aliasesFromList(ctx, types.ListNull(ot), &diags)
		if ok {
			t.Error("null list should return ok=false")
		}
	})
	t.Run("unknown list returns ok=false", func(t *testing.T) {
		var diags diag.Diagnostics
		_, ok := aliasesFromList(ctx, types.ListUnknown(ot), &diags)
		if ok {
			t.Error("unknown list should return ok=false")
		}
	})
	t.Run("populated list", func(t *testing.T) {
		var diags diag.Diagnostics
		obj1, _ := types.ObjectValue(ot.AttrTypes, map[string]attr.Value{
			"type":    types.StringValue("INET"),
			"address": types.StringValue("10.0.0.5"),
			"netmask": types.Int64Value(24),
		})
		obj2, _ := types.ObjectValue(ot.AttrTypes, map[string]attr.Value{
			"type":    types.StringValue("INET6"),
			"address": types.StringValue("fe80::1"),
			"netmask": types.Int64Value(64),
		})
		list, _ := types.ListValue(ot, []attr.Value{obj1, obj2})
		result, ok := aliasesFromList(ctx, list, &diags)
		if !ok {
			t.Fatalf("expected ok=true")
		}
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].Type != "INET" || result[0].Address != "10.0.0.5" || result[0].Netmask != 24 {
			t.Errorf("result[0] mismatch: %+v", result[0])
		}
		if result[1].Type != "INET6" || result[1].Netmask != 64 {
			t.Errorf("result[1] mismatch: %+v", result[1])
		}
	})
	t.Run("empty list returns empty slice", func(t *testing.T) {
		var diags diag.Diagnostics
		list, _ := types.ListValue(ot, []attr.Value{})
		result, ok := aliasesFromList(ctx, list, &diags)
		if !ok {
			t.Fatalf("expected ok=true")
		}
		if len(result) != 0 {
			t.Errorf("expected empty")
		}
	})
}

// --- NFSConfig buildUpdateRequest ---

func TestNFSConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &NFSConfigResource{}
	ctx := context.Background()
	t.Run("populated", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &NFSConfigResourceModel{
			Servers:      types.Int64Value(8),
			AllowNonroot: types.BoolValue(false),
			Protocols:    stringListValue(t, []string{"NFSV3", "NFSV4"}),
			V4Krb:        types.BoolValue(true),
			V4Domain:     types.StringValue("CORP.EXAMPLE.COM"),
			BindIP:       stringListValue(t, []string{"10.0.0.5", "10.0.0.6"}),
			MountdPort:   types.Int64Value(600),
			RpcstatdPort: types.Int64Value(601),
			RpclockdPort: types.Int64Value(602),
		}
		req := r.buildUpdateRequest(ctx, plan, &d)
		if req.Servers == nil || *req.Servers != 8 {
			t.Errorf("Servers mismatch")
		}
		if req.V4Krb == nil || !*req.V4Krb {
			t.Errorf("V4Krb mismatch")
		}
		if req.V4Domain == nil || *req.V4Domain != "CORP.EXAMPLE.COM" {
			t.Errorf("V4Domain mismatch")
		}
		if len(req.Protocols) != 2 {
			t.Errorf("Protocols len = %d", len(req.Protocols))
		}
		if len(req.BindIP) != 2 {
			t.Errorf("BindIP len = %d", len(req.BindIP))
		}
		if req.MountdPort == nil || *req.MountdPort != 600 {
			t.Errorf("MountdPort mismatch")
		}
	})
	t.Run("all null", func(t *testing.T) {
		var d diag.Diagnostics
		plan := &NFSConfigResourceModel{
			Servers:      types.Int64Null(),
			AllowNonroot: types.BoolNull(),
			Protocols:    types.ListNull(types.StringType),
			V4Krb:        types.BoolNull(),
			V4Domain:     types.StringNull(),
			BindIP:       types.ListNull(types.StringType),
			MountdPort:   types.Int64Null(),
			RpcstatdPort: types.Int64Null(),
			RpclockdPort: types.Int64Null(),
		}
		req := r.buildUpdateRequest(ctx, plan, &d)
		if req.Servers != nil || req.V4Krb != nil || req.V4Domain != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}
