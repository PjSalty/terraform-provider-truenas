package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --- AlertService ---

func TestAlertServiceResource_MapResponseToModel_Cases(t *testing.T) {
	r := &AlertServiceResource{}
	cases := []struct {
		name     string
		svc      *client.AlertService
		wantType string
	}{
		{name: "mail via legacy type", svc: &client.AlertService{ID: 1, Name: "mail", Type: "Mail", Enabled: true, Level: "WARNING", Settings: map[string]interface{}{"email": "foo@bar"}}, wantType: "Mail"},
		{name: "pushover via attributes type", svc: &client.AlertService{ID: 2, Name: "po", Enabled: true, Level: "INFO", Settings: map[string]interface{}{"type": "Pushover", "api_key": "k"}}, wantType: "Pushover"},
		{name: "slack disabled", svc: &client.AlertService{ID: 3, Name: "slack", Enabled: false, Level: "CRITICAL", Settings: map[string]interface{}{"type": "Slack"}}, wantType: "Slack"},
		{name: "nil settings", svc: &client.AlertService{ID: 4, Name: "empty", Type: "Custom", Enabled: true, Level: "NOTICE"}, wantType: "Custom"},
		{name: "telegram via attributes", svc: &client.AlertService{ID: 5, Name: "tg", Enabled: true, Level: "ALERT", Settings: map[string]interface{}{"type": "Telegram", "bot_token": "t", "chat_ids": []interface{}{float64(123)}}}, wantType: "Telegram"},
		{name: "VictorOps attr", svc: &client.AlertService{ID: 6, Name: "vo", Enabled: true, Level: "CRITICAL", Settings: map[string]interface{}{"type": "VictorOps", "api_key": "k", "routing_key": "r"}}, wantType: "VictorOps"},
		{name: "PagerDuty attr", svc: &client.AlertService{ID: 7, Name: "pd", Enabled: true, Level: "ALERT", Settings: map[string]interface{}{"type": "PagerDuty", "service_key": "s", "client_name": "c"}}, wantType: "PagerDuty"},
		{name: "SNMP trap legacy type", svc: &client.AlertService{ID: 8, Name: "snmp", Type: "SNMPTrap", Enabled: false, Level: "WARNING"}, wantType: "SNMPTrap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m AlertServiceResourceModel
			r.mapResponseToModel(tc.svc, &m)
			if m.Name.ValueString() != tc.svc.Name {
				t.Errorf("Name mismatch")
			}
			if m.Type.ValueString() != tc.wantType {
				t.Errorf("Type = %q, want %q", m.Type.ValueString(), tc.wantType)
			}
			if m.Enabled.ValueBool() != tc.svc.Enabled {
				t.Errorf("Enabled mismatch")
			}
			if m.Level.ValueString() != tc.svc.Level {
				t.Errorf("Level mismatch")
			}
		})
	}
}

func TestAlertServiceResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewAlertServiceResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "type", "enabled", "level", "settings_json"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- CloudSync ---

func TestCloudSyncResource_MapResponseToModel_Cases(t *testing.T) {
	r := &CloudSyncResource{}
	cases := []struct {
		name string
		cs   *client.CloudSync
	}{
		{name: "push", cs: &client.CloudSync{ID: 1, Description: "backup", Path: "/mnt/tank", Credentials: 2, Direction: "PUSH", TransferMode: "COPY", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "1", Dom: "*", Month: "*", Dow: "*"}, Attributes: map[string]interface{}{"bucket": "b"}}},
		{name: "pull sync", cs: &client.CloudSync{ID: 2, Path: "/a", Credentials: 3, Direction: "PULL", TransferMode: "SYNC", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "move direction", cs: &client.CloudSync{ID: 3, Path: "/x", Credentials: 1, Direction: "PUSH", TransferMode: "MOVE", Schedule: client.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "minimal", cs: &client.CloudSync{ID: 4, Path: "/p", Credentials: 1, Direction: "PUSH", TransferMode: "COPY"}},
		{name: "with description", cs: &client.CloudSync{ID: 5, Description: "backup task", Path: "/a", Credentials: 2, Direction: "PUSH", TransferMode: "COPY"}},
		{name: "pull copy", cs: &client.CloudSync{ID: 6, Path: "/b", Credentials: 3, Direction: "PULL", TransferMode: "COPY"}},
		{name: "pull move", cs: &client.CloudSync{ID: 7, Path: "/c", Credentials: 1, Direction: "PULL", TransferMode: "MOVE"}},
		{name: "with attrs", cs: &client.CloudSync{ID: 8, Path: "/d", Credentials: 1, Direction: "PUSH", TransferMode: "COPY", Attributes: map[string]interface{}{"bucket": "b", "prefix": "p"}, Enabled: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m CloudSyncResourceModel
			r.mapResponseToModel(tc.cs, &m)
			if m.Path.ValueString() != tc.cs.Path {
				t.Errorf("Path mismatch")
			}
			if m.Credentials.ValueInt64() != int64(tc.cs.Credentials) {
				t.Errorf("Credentials mismatch")
			}
			if m.Direction.ValueString() != tc.cs.Direction {
				t.Errorf("Direction mismatch")
			}
			if m.TransferMode.ValueString() != tc.cs.TransferMode {
				t.Errorf("TransferMode mismatch")
			}
		})
	}
}

func TestCloudSyncResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewCloudSyncResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "path", "credentials", "direction", "transfer_mode"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
}

// --- KeychainCredential ---

func TestKeychainCredentialResource_MapResponseToModel_Cases(t *testing.T) {
	r := &KeychainCredentialResource{}
	ctx := context.Background()
	cases := []struct {
		name string
		cred *client.KeychainCredential
	}{
		{name: "ssh key pair", cred: &client.KeychainCredential{ID: 1, Name: "key1", Type: "SSH_KEY_PAIR", Attributes: map[string]interface{}{"public_key": "ssh-rsa AAAA"}}},
		{name: "ssh credentials", cred: &client.KeychainCredential{ID: 2, Name: "remote", Type: "SSH_CREDENTIALS", Attributes: map[string]interface{}{"host": "remote", "port": float64(22)}}},
		{name: "empty attributes", cred: &client.KeychainCredential{ID: 3, Name: "blank", Type: "SSH_KEY_PAIR"}},
		{name: "multiple attrs", cred: &client.KeychainCredential{ID: 4, Name: "multi", Type: "SSH_CREDENTIALS", Attributes: map[string]interface{}{"host": "h", "username": "u"}}},
		{name: "ssh with port", cred: &client.KeychainCredential{ID: 5, Name: "ssh-port", Type: "SSH_CREDENTIALS", Attributes: map[string]interface{}{"host": "remote.example.com", "port": float64(2222), "username": "backup"}}},
		{name: "many attributes", cred: &client.KeychainCredential{ID: 6, Name: "full", Type: "SSH_CREDENTIALS", Attributes: map[string]interface{}{"host": "h", "username": "u", "port": float64(22), "private_key": "k", "remote_host_key": "rhk"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m KeychainCredentialResourceModel
			r.mapResponseToModel(ctx, tc.cred, &m)
			if m.Name.ValueString() != tc.cred.Name {
				t.Errorf("Name mismatch")
			}
			if m.Type.ValueString() != tc.cred.Type {
				t.Errorf("Type mismatch")
			}
			if got := len(m.Attributes.Elements()); got != len(tc.cred.Attributes) {
				t.Errorf("Attributes len = %d, want %d", got, len(tc.cred.Attributes))
			}
		})
	}
}

func TestKeychainCredentialResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewKeychainCredentialResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "type", "attributes"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- Certificate ---

func TestCertificateResource_MapResponseToModel_Cases(t *testing.T) {
	r := &CertificateResource{}
	ctx := context.Background()
	cases := []struct {
		name string
		cert *client.Certificate
	}{
		{name: "basic cert", cert: &client.Certificate{ID: 1, Name: "c1", CertificateData: "-----BEGIN-----", KeyType: "RSA", KeyLength: 4096, DigestAlgorithm: "SHA256", Lifetime: 365, Common: "example.com", SAN: []string{"example.com", "www.example.com"}}},
		{name: "ec cert", cert: &client.Certificate{ID: 2, Name: "ec", CertificateData: "data", KeyType: "EC", KeyLength: 256, DigestAlgorithm: "SHA256", Lifetime: 90, Country: "US", State: "CA", Organization: "Acme"}},
		{name: "expired cert", cert: &client.Certificate{ID: 3, Name: "old", CertificateData: "d", KeyType: "RSA", KeyLength: 2048, Expired: true, DigestAlgorithm: "SHA256"}},
		{name: "no san", cert: &client.Certificate{ID: 4, Name: "nosan", CertificateData: "d", KeyType: "RSA", KeyLength: 2048, DigestAlgorithm: "SHA256"}},
		{name: "wildcard cert", cert: &client.Certificate{ID: 5, Name: "wild", CertificateData: "d", KeyType: "RSA", KeyLength: 2048, DigestAlgorithm: "SHA256", Common: "*.example.com", SAN: []string{"*.example.com", "example.com"}}},
		{name: "self-signed CA", cert: &client.Certificate{ID: 6, Name: "ca", CertificateData: "d", KeyType: "RSA", KeyLength: 4096, DigestAlgorithm: "SHA512", Lifetime: 3650, Common: "Internal CA", Country: "US", State: "CA", Organization: "Home"}},
		{name: "with all fields", cert: &client.Certificate{ID: 7, Name: "full", CertificateData: "d", KeyType: "EC", KeyLength: 384, DigestAlgorithm: "SHA384", Common: "api.example.com", SAN: []string{"api.example.com", "api-v2.example.com", "api-v3.example.com"}, Country: "US", State: "NY", Organization: "Example", Email: "admin@example.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m CertificateResourceModel
			r.mapResponseToModel(ctx, tc.cert, &m)
			if m.Name.ValueString() != tc.cert.Name {
				t.Errorf("Name mismatch")
			}
			if m.KeyType.ValueString() != tc.cert.KeyType {
				t.Errorf("KeyType mismatch")
			}
			if m.KeyLength.ValueInt64() != int64(tc.cert.KeyLength) {
				t.Errorf("KeyLength mismatch")
			}
			if m.Expired.ValueBool() != tc.cert.Expired {
				t.Errorf("Expired mismatch")
			}
			if got := len(m.SAN.Elements()); got != len(tc.cert.SAN) {
				t.Errorf("SAN len = %d, want %d", got, len(tc.cert.SAN))
			}
		})
	}
}

func TestCertificateResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewCertificateResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "create_type", "key_type", "key_length", "san"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- APIKey (schema only - mapping is inline in Create/Read) ---

func TestAPIKeyResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewAPIKeyResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "username", "key"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
	if !attrs["username"].IsRequired() {
		t.Error("username should be required")
	}
	if !attrs["key"].IsComputed() {
		t.Error("key should be computed")
	}
	if !attrs["key"].IsSensitive() {
		t.Error("key should be sensitive")
	}
}

// --- Privilege ---

func TestPrivilegeResource_MapResponseToModel_Cases(t *testing.T) {
	r := &PrivilegeResource{}
	ctx := context.Background()
	cases := []struct {
		name string
		priv *client.Privilege
	}{
		{name: "basic admin", priv: &client.Privilege{ID: 1, Name: "admins", LocalGroups: []client.PrivilegeGroup{{ID: 1, GID: 545}}, Roles: []string{"FULL_ADMIN"}, WebShell: true}},
		{name: "with ds groups", priv: &client.Privilege{ID: 2, Name: "ops", LocalGroups: []client.PrivilegeGroup{}, DSGroups: []interface{}{"S-1-5-21-1"}, Roles: []string{"READONLY_ADMIN"}}},
		{name: "multiple roles", priv: &client.Privilege{ID: 3, Name: "multi", LocalGroups: []client.PrivilegeGroup{{ID: 2, GID: 546}}, Roles: []string{"SHARING_ADMIN", "DATASET_WRITE"}}},
		{name: "empty privilege", priv: &client.Privilege{ID: 4, Name: "empty"}},
		{name: "no web shell", priv: &client.Privilege{ID: 5, Name: "noui", LocalGroups: []client.PrivilegeGroup{{ID: 10, GID: 1000}}, Roles: []string{"READONLY_ADMIN"}, WebShell: false}},
		{name: "multiple local groups", priv: &client.Privilege{ID: 6, Name: "multi_grp", LocalGroups: []client.PrivilegeGroup{{ID: 1, GID: 100}, {ID: 2, GID: 200}, {ID: 3, GID: 300}}, Roles: []string{"FULL_ADMIN"}}},
		{name: "ds groups multiple", priv: &client.Privilege{ID: 7, Name: "ds_multi", DSGroups: []interface{}{"S-1-5-21-1", "S-1-5-21-2", "S-1-5-21-3"}, Roles: []string{"FULL_ADMIN"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m PrivilegeResourceModel
			var d diag.Diagnostics
			r.mapResponseToModel(ctx, tc.priv, &m, &d)
			if d.HasError() {
				t.Fatalf("diags: %v", d)
			}
			if m.Name.ValueString() != tc.priv.Name {
				t.Errorf("Name mismatch")
			}
			if m.WebShell.ValueBool() != tc.priv.WebShell {
				t.Errorf("WebShell mismatch")
			}
			if got := len(m.LocalGroups.Elements()); got != len(tc.priv.LocalGroups) {
				t.Errorf("LocalGroups len = %d, want %d", got, len(tc.priv.LocalGroups))
			}
			if got := len(m.Roles.Elements()); got != len(tc.priv.Roles) {
				t.Errorf("Roles len = %d, want %d", got, len(tc.priv.Roles))
			}
		})
	}
}

func TestPrivilegeResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewPrivilegeResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "local_groups", "ds_groups", "roles", "web_shell"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- ReportingExporter ---

func TestReportingExporterResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ReportingExporterResource{}
	cases := []struct {
		name string
		e    *client.ReportingExporter
	}{
		{name: "graphite enabled", e: &client.ReportingExporter{ID: 1, Name: "graphite", Enabled: true, Attributes: json.RawMessage(`{"host":"graphite.local","port":2003}`)}},
		{name: "disabled exporter", e: &client.ReportingExporter{ID: 2, Name: "off", Enabled: false, Attributes: json.RawMessage(`{}`)}},
		{name: "nil attributes", e: &client.ReportingExporter{ID: 3, Name: "x", Enabled: true}},
		{name: "complex attributes", e: &client.ReportingExporter{ID: 4, Name: "y", Enabled: true, Attributes: json.RawMessage(`{"host":"h","port":2003,"prefix":"tnas"}`)}},
		{name: "prometheus remote write", e: &client.ReportingExporter{ID: 5, Name: "prom", Enabled: true, Attributes: json.RawMessage(`{"url":"https://prom/api/v1/write","username":"u","password":"p"}`)}},
		{name: "influxdb v2", e: &client.ReportingExporter{ID: 6, Name: "influx", Enabled: true, Attributes: json.RawMessage(`{"org":"homelab","bucket":"tnas","token":"t","url":"https://influx"}`)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ReportingExporterResourceModel
			r.mapResponseToModel(tc.e, &m)
			if m.Name.ValueString() != tc.e.Name {
				t.Errorf("Name mismatch")
			}
			if m.Enabled.ValueBool() != tc.e.Enabled {
				t.Errorf("Enabled mismatch")
			}
		})
	}
}

func TestReportingExporterResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewReportingExporterResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "enabled", "attributes_json"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- Pool ---

func TestPoolResource_MapResponseToModel_Cases(t *testing.T) {
	r := &PoolResource{}
	cases := []struct {
		name string
		pool *client.Pool
	}{
		{name: "healthy tank", pool: &client.Pool{ID: 1, Name: "tank", GUID: "12345", Path: "/mnt/tank", Status: "ONLINE", Healthy: true}},
		{name: "degraded pool", pool: &client.Pool{ID: 2, Name: "ssd", GUID: "99999", Path: "/mnt/ssd", Status: "DEGRADED", Healthy: false}},
		{name: "fault pool", pool: &client.Pool{ID: 3, Name: "fault", GUID: "abc", Path: "/mnt/fault", Status: "FAULTED", Healthy: false}},
		{name: "minimal", pool: &client.Pool{ID: 4, Name: "min", Status: "ONLINE", Healthy: true}},
		{name: "offline pool", pool: &client.Pool{ID: 5, Name: "off", GUID: "offg", Path: "/mnt/off", Status: "OFFLINE", Healthy: false}},
		{name: "long pool name", pool: &client.Pool{ID: 6, Name: "really-long-pool-name-for-test", GUID: "longg", Path: "/mnt/rlpnft", Status: "ONLINE", Healthy: true}},
		{name: "unavail pool", pool: &client.Pool{ID: 7, Name: "unavail", Status: "UNAVAIL", Healthy: false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m PoolResourceModel
			r.mapResponseToModel(tc.pool, &m)
			if m.Name.ValueString() != tc.pool.Name {
				t.Errorf("Name mismatch")
			}
			if m.GUID.ValueString() != tc.pool.GUID {
				t.Errorf("GUID mismatch")
			}
			if m.Status.ValueString() != tc.pool.Status {
				t.Errorf("Status mismatch")
			}
			if m.Healthy.ValueBool() != tc.pool.Healthy {
				t.Errorf("Healthy mismatch")
			}
		})
	}
}

func TestPoolResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewPoolResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "topology_json", "guid", "status", "healthy"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}
