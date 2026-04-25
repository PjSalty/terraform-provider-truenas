package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// Config resources share an identical pattern: buildUpdateRequest maps a
// TF model to a client update request, skipping null/unknown values so
// unwritten fields aren't clobbered. These tests exercise the populated
// path and the null/unknown path for each singleton config builder.

// --- FTP Config ---

func TestFTPConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &FTPConfigResource{}
	t.Run("all fields populated", func(t *testing.T) {
		plan := &FTPConfigResourceModel{
			Port:          types.Int64Value(21),
			Clients:       types.Int64Value(32),
			IPConnections: types.Int64Value(8),
			LoginAttempt:  types.Int64Value(3),
			Timeout:       types.Int64Value(120),
			OnlyAnonymous: types.BoolValue(false),
			OnlyLocal:     types.BoolValue(false),
			Banner:        types.StringValue("FTP Server"),
			Filemask:      types.StringValue("077"),
			Dirmask:       types.StringValue("077"),
			FXP:           types.BoolValue(false),
			Resume:        types.BoolValue(true),
			DefaultRoot:   types.BoolValue(false),
			TLS:           types.BoolValue(true),
		}
		req := r.buildUpdateRequest(plan)
		if req.Port == nil || *req.Port != 21 {
			t.Errorf("Port mismatch")
		}
		if req.Clients == nil || *req.Clients != 32 {
			t.Errorf("Clients mismatch")
		}
		if req.Banner == nil || *req.Banner != "FTP Server" {
			t.Errorf("Banner mismatch")
		}
		if req.TLS == nil || !*req.TLS {
			t.Errorf("TLS mismatch")
		}
		if req.Resume == nil || !*req.Resume {
			t.Errorf("Resume mismatch")
		}
	})
	t.Run("all null/unknown omitted", func(t *testing.T) {
		plan := &FTPConfigResourceModel{
			Port:          types.Int64Null(),
			Clients:       types.Int64Unknown(),
			IPConnections: types.Int64Null(),
			LoginAttempt:  types.Int64Null(),
			Timeout:       types.Int64Null(),
			OnlyAnonymous: types.BoolNull(),
			OnlyLocal:     types.BoolUnknown(),
			Banner:        types.StringNull(),
			Filemask:      types.StringNull(),
			Dirmask:       types.StringNull(),
			FXP:           types.BoolNull(),
			Resume:        types.BoolNull(),
			DefaultRoot:   types.BoolNull(),
			TLS:           types.BoolNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Port != nil || req.Clients != nil || req.Banner != nil || req.TLS != nil {
			t.Errorf("null/unknown fields should not be set: %+v", req)
		}
	})
}

// --- SMB Config ---

func TestSMBConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &SMBConfigResource{}
	t.Run("all fields populated", func(t *testing.T) {
		plan := &SMBConfigResourceModel{
			NetbiosName:    types.StringValue("TRUENAS"),
			Workgroup:      types.StringValue("WORKGROUP"),
			Description:    types.StringValue("TrueNAS SMB"),
			EnableSMB1:     types.BoolValue(false),
			UnixCharset:    types.StringValue("UTF-8"),
			AAPLExtensions: types.BoolValue(true),
			Guest:          types.StringValue("nobody"),
			Filemask:       types.StringValue("0775"),
			Dirmask:        types.StringValue("0775"),
		}
		req := r.buildUpdateRequest(plan)
		if req.NetbiosName == nil || *req.NetbiosName != "TRUENAS" {
			t.Errorf("NetbiosName mismatch")
		}
		if req.AAPLExtensions == nil || !*req.AAPLExtensions {
			t.Errorf("AAPLExtensions mismatch")
		}
		if req.Workgroup == nil || *req.Workgroup != "WORKGROUP" {
			t.Errorf("Workgroup mismatch")
		}
	})
	t.Run("null fields omitted", func(t *testing.T) {
		plan := &SMBConfigResourceModel{
			NetbiosName: types.StringNull(),
			Workgroup:   types.StringNull(),
			EnableSMB1:  types.BoolNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.NetbiosName != nil || req.Workgroup != nil || req.EnableSMB1 != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}

// --- SNMP Config ---

func TestSNMPConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &SNMPConfigResource{}
	t.Run("v3 disabled", func(t *testing.T) {
		plan := &SNMPConfigResourceModel{
			Community: types.StringValue("public"),
			Contact:   types.StringValue("admin@example.com"),
			Location:  types.StringValue("DataCenter1"),
			V3:        types.BoolValue(false),
		}
		req := r.buildUpdateRequest(plan)
		if req.Community == nil || *req.Community != "public" {
			t.Errorf("Community mismatch")
		}
		if req.V3 == nil || *req.V3 {
			t.Errorf("V3 should be false")
		}
	})
	t.Run("v3 enabled", func(t *testing.T) {
		plan := &SNMPConfigResourceModel{
			Community:        types.StringValue("public"),
			V3:               types.BoolValue(true),
			V3Username:       types.StringValue("snmpuser"),
			V3AuthType:       types.StringValue("SHA"),
			V3Password:       types.StringValue("pw"),
			V3PrivProto:      types.StringValue("AES"),
			V3PrivPassphrase: types.StringValue("priv"),
		}
		req := r.buildUpdateRequest(plan)
		if req.V3 == nil || !*req.V3 {
			t.Errorf("V3 should be true")
		}
		if req.V3Username == nil || *req.V3Username != "snmpuser" {
			t.Errorf("V3Username mismatch")
		}
		if req.V3AuthType == nil || *req.V3AuthType != "SHA" {
			t.Errorf("V3AuthType mismatch")
		}
	})
	t.Run("all null", func(t *testing.T) {
		plan := &SNMPConfigResourceModel{
			Community: types.StringNull(),
			V3:        types.BoolNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Community != nil || req.V3 != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}

// --- UPS Config ---

func TestUPSConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &UPSConfigResource{}
	t.Run("master mode", func(t *testing.T) {
		plan := &UPSConfigResourceModel{
			Mode:          types.StringValue("MASTER"),
			Identifier:    types.StringValue("ups"),
			Driver:        types.StringValue("usbhid-ups"),
			Port:          types.StringValue("auto"),
			Shutdown:      types.StringValue("BATT"),
			ShutdownTimer: types.Int64Value(30),
			Description:   types.StringValue("primary UPS"),
		}
		req := r.buildUpdateRequest(plan)
		if req.Mode == nil || *req.Mode != "MASTER" {
			t.Errorf("Mode mismatch")
		}
		if req.ShutdownTimer == nil || *req.ShutdownTimer != 30 {
			t.Errorf("ShutdownTimer mismatch")
		}
	})
	t.Run("slave mode", func(t *testing.T) {
		plan := &UPSConfigResourceModel{
			Mode:       types.StringValue("SLAVE"),
			RemoteHost: types.StringValue("10.0.0.1"),
			RemotePort: types.Int64Value(3493),
		}
		req := r.buildUpdateRequest(plan)
		if req.Mode == nil || *req.Mode != "SLAVE" {
			t.Errorf("Mode mismatch")
		}
		if req.RemoteHost == nil || *req.RemoteHost != "10.0.0.1" {
			t.Errorf("RemoteHost mismatch")
		}
	})
	t.Run("all null", func(t *testing.T) {
		plan := &UPSConfigResourceModel{Mode: types.StringNull()}
		req := r.buildUpdateRequest(plan)
		if req.Mode != nil {
			t.Errorf("null field should be skipped")
		}
	})
}

// --- Mail Config ---

func TestMailConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &MailConfigResource{}
	t.Run("plain smtp", func(t *testing.T) {
		plan := &MailConfigResourceModel{
			FromEmail:      types.StringValue("admin@example.com"),
			FromName:       types.StringValue("TrueNAS"),
			OutgoingServer: types.StringValue("smtp.example.com"),
			Port:           types.Int64Value(587),
			Security:       types.StringValue("TLS"),
			SMTP:           types.BoolValue(true),
			User:           types.StringValue("admin"),
			Pass:           types.StringValue("p"),
		}
		req := r.buildUpdateRequest(plan)
		if req.FromEmail == nil || *req.FromEmail != "admin@example.com" {
			t.Errorf("FromEmail mismatch")
		}
		if req.Port == nil || *req.Port != 587 {
			t.Errorf("Port mismatch")
		}
		if req.SMTP == nil || !*req.SMTP {
			t.Errorf("SMTP mismatch")
		}
	})
	t.Run("all null", func(t *testing.T) {
		plan := &MailConfigResourceModel{
			Port: types.Int64Null(),
			SMTP: types.BoolNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Port != nil || req.SMTP != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}

// --- KMIP Config ---

func TestKMIPConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &KMIPConfigResource{}
	t.Run("enabled with SED", func(t *testing.T) {
		plan := &KMIPConfigResourceModel{
			Enabled:              types.BoolValue(true),
			ManageSEDDisks:       types.BoolValue(true),
			ManageZFSKeys:        types.BoolValue(false),
			Certificate:          types.Int64Value(1),
			CertificateAuthority: types.Int64Value(2),
			Port:                 types.Int64Value(5696),
			Server:               types.StringValue("kmip.example.com"),
			SSLVersion:           types.StringValue("PROTOCOL_TLSv1_2"),
			ChangeServer:         types.BoolValue(false),
			Validate:             types.BoolValue(true),
			ForceClear:           types.BoolValue(false),
		}
		req := r.buildUpdateRequest(plan)
		if req.Enabled == nil || !*req.Enabled {
			t.Errorf("Enabled mismatch")
		}
		if req.Server == nil || *req.Server != "kmip.example.com" {
			t.Errorf("Server mismatch")
		}
		if req.Port == nil || *req.Port != 5696 {
			t.Errorf("Port mismatch")
		}
	})
	t.Run("null certs omitted, others always present", func(t *testing.T) {
		// KMIP builder always populates Enabled/Port/Server/etc from TF model
		// (null => zero value in request). Certificate and CertificateAuthority
		// are the only two that get skipped when null/zero.
		plan := &KMIPConfigResourceModel{
			Enabled:              types.BoolValue(false),
			Port:                 types.Int64Value(5696),
			Server:               types.StringValue(""),
			Certificate:          types.Int64Null(),
			CertificateAuthority: types.Int64Null(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Certificate != nil {
			t.Errorf("Certificate should be nil for null")
		}
		if req.CertificateAuthority != nil {
			t.Errorf("CertificateAuthority should be nil for null")
		}
		if req.Enabled == nil || *req.Enabled {
			t.Errorf("Enabled should be set to false explicitly")
		}
	})
}

// --- NVMet Global ---

func TestNVMetGlobalResource_BuildUpdateRequest(t *testing.T) {
	r := &NVMetGlobalResource{}
	t.Run("all populated", func(t *testing.T) {
		plan := &NVMetGlobalResourceModel{
			Basenqn:       types.StringValue("nqn.2020-01.org.nvmexpress"),
			Kernel:        types.BoolValue(true),
			Ana:           types.BoolValue(false),
			Rdma:          types.BoolValue(true),
			XportReferral: types.BoolValue(true),
		}
		req := r.buildUpdateRequest(plan)
		if req.Basenqn == nil || *req.Basenqn != "nqn.2020-01.org.nvmexpress" {
			t.Errorf("Basenqn mismatch")
		}
		if req.Kernel == nil || !*req.Kernel {
			t.Errorf("Kernel mismatch")
		}
		if req.Rdma == nil || !*req.Rdma {
			t.Errorf("Rdma mismatch")
		}
	})
	t.Run("all null", func(t *testing.T) {
		plan := &NVMetGlobalResourceModel{
			Basenqn: types.StringNull(),
			Kernel:  types.BoolNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Basenqn != nil || req.Kernel != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}

// --- mapResponseToModel tests for singleton configs ---

func TestFTPConfigResource_MapResponseToModel(t *testing.T) {
	r := &FTPConfigResource{}
	cases := []*client.FTPConfig{
		{Port: 21, Clients: 32, Banner: "TrueNAS FTP"},
		{Port: 21, TLS: true, Banner: "FTPS"},
		{Port: 2121, OnlyAnonymous: true},
		{Port: 21, Banner: "Welcome to production file transfer", DefaultRoot: true},
		{Port: 21, FXP: true, Resume: true, Filemask: "077", Dirmask: "077"},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m FTPConfigResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Port.ValueInt64() != int64(cfg.Port) {
				t.Errorf("case %d: Port mismatch", i)
			}
			if m.Banner.ValueString() != cfg.Banner {
				t.Errorf("case %d: Banner mismatch", i)
			}
			if m.OnlyAnonymous.ValueBool() != cfg.OnlyAnonymous {
				t.Errorf("case %d: OnlyAnonymous mismatch", i)
			}
			if m.TLS.ValueBool() != cfg.TLS {
				t.Errorf("case %d: TLS mismatch", i)
			}
			if m.ID.ValueString() == "" {
				t.Errorf("case %d: ID should be set", i)
			}
		})
	}
}

func TestSMBConfigResource_MapResponseToModel(t *testing.T) {
	r := &SMBConfigResource{}
	cases := []*client.SMBConfig{
		{NetbiosName: "TRUENAS", Workgroup: "WORKGROUP"},
		{NetbiosName: "FS01", Workgroup: "CORP", Description: "file server"},
		{NetbiosName: "LEGACY", Workgroup: "OLD", EnableSMB1: true},
		{NetbiosName: "MAC", Workgroup: "WG", AAPLExtensions: true, UnixCharset: "UTF-8"},
		{NetbiosName: "GUEST", Workgroup: "WG", Guest: "nobody", Filemask: "0775", Dirmask: "0775"},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m SMBConfigResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.NetbiosName.ValueString() != cfg.NetbiosName {
				t.Errorf("case %d: NetbiosName mismatch", i)
			}
			if m.Workgroup.ValueString() != cfg.Workgroup {
				t.Errorf("case %d: Workgroup mismatch", i)
			}
			if m.EnableSMB1.ValueBool() != cfg.EnableSMB1 {
				t.Errorf("case %d: EnableSMB1 mismatch", i)
			}
			if m.AAPLExtensions.ValueBool() != cfg.AAPLExtensions {
				t.Errorf("case %d: AAPLExtensions mismatch", i)
			}
		})
	}
}

// --- SSH Config ---

func TestSSHConfigResource_BuildUpdateRequest(t *testing.T) {
	r := &SSHConfigResource{}
	ctx := testCtx()
	t.Run("populated", func(t *testing.T) {
		var d diagDiagnostics
		plan := &SSHConfigResourceModel{
			TCPPort:         types.Int64Value(22),
			PasswordAuth:    types.BoolValue(false),
			KerberosAuth:    types.BoolValue(false),
			TCPFwd:          types.BoolValue(false),
			Compression:     types.BoolValue(false),
			SFTPLogLevel:    types.StringValue("VERBOSE"),
			SFTPLogFacility: types.StringValue("LOCAL3"),
			WeakCiphers:     stringListValue(t, []string{"AES128-CBC"}),
		}
		req := r.buildUpdateRequest(ctx, plan, &d.Diagnostics)
		if req.TCPPort == nil || *req.TCPPort != 22 {
			t.Errorf("TCPPort mismatch")
		}
		if req.SFTPLogLevel == nil || *req.SFTPLogLevel != "VERBOSE" {
			t.Errorf("SFTPLogLevel mismatch")
		}
		if req.WeakCiphers == nil || len(*req.WeakCiphers) != 1 {
			t.Errorf("WeakCiphers mismatch")
		}
	})
	t.Run("all null", func(t *testing.T) {
		var d diagDiagnostics
		plan := &SSHConfigResourceModel{
			TCPPort:         types.Int64Null(),
			PasswordAuth:    types.BoolNull(),
			KerberosAuth:    types.BoolNull(),
			TCPFwd:          types.BoolNull(),
			Compression:     types.BoolNull(),
			SFTPLogLevel:    types.StringNull(),
			SFTPLogFacility: types.StringNull(),
			WeakCiphers:     types.ListNull(types.StringType),
		}
		req := r.buildUpdateRequest(ctx, plan, &d.Diagnostics)
		if req.TCPPort != nil || req.PasswordAuth != nil {
			t.Errorf("null fields should be skipped")
		}
	})
}

func TestSSHConfigResource_MapResponseToModel(t *testing.T) {
	r := &SSHConfigResource{}
	ctx := testCtx()
	cases := []*client.SSHConfig{
		{TCPPort: 22, PasswordAuth: true, TCPFwd: true},
		{TCPPort: 2222, PasswordAuth: false, KerberosAuth: false, Compression: true},
		{TCPPort: 22, SFTPLogLevel: "INFO", SFTPLogFacility: "AUTH"},
		{TCPPort: 22, WeakCiphers: []string{"AES128-CBC", "3DES-CBC"}},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m SSHConfigResourceModel
			r.mapResponseToModel(ctx, cfg, &m)
			if m.TCPPort.ValueInt64() != int64(cfg.TCPPort) {
				t.Errorf("case %d: TCPPort mismatch", i)
			}
			if m.PasswordAuth.ValueBool() != cfg.PasswordAuth {
				t.Errorf("case %d: PasswordAuth mismatch", i)
			}
			if len(m.WeakCiphers.Elements()) != len(cfg.WeakCiphers) {
				t.Errorf("case %d: WeakCiphers len mismatch", i)
			}
		})
	}
}

// --- DNS Nameserver ---

func TestDNSNameserverResource_BuildUpdateRequest(t *testing.T) {
	r := &DNSNameserverResource{}
	t.Run("three nameservers", func(t *testing.T) {
		plan := &DNSNameserverResourceModel{
			Nameserver1: types.StringValue("1.1.1.1"),
			Nameserver2: types.StringValue("8.8.8.8"),
			Nameserver3: types.StringValue("9.9.9.9"),
		}
		req := r.buildUpdateRequest(plan)
		if req.Nameserver1 == nil || *req.Nameserver1 != "1.1.1.1" {
			t.Errorf("NS1 mismatch")
		}
		if req.Nameserver2 == nil || *req.Nameserver2 != "8.8.8.8" {
			t.Errorf("NS2 mismatch")
		}
		if req.Nameserver3 == nil || *req.Nameserver3 != "9.9.9.9" {
			t.Errorf("NS3 mismatch")
		}
	})
	t.Run("one nameserver others null", func(t *testing.T) {
		plan := &DNSNameserverResourceModel{
			Nameserver1: types.StringValue("1.1.1.1"),
			Nameserver2: types.StringNull(),
			Nameserver3: types.StringNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Nameserver1 == nil {
			t.Errorf("NS1 should be set")
		}
		if req.Nameserver2 != nil || req.Nameserver3 != nil {
			t.Errorf("NS2/3 should be nil")
		}
	})
	t.Run("all null", func(t *testing.T) {
		plan := &DNSNameserverResourceModel{
			Nameserver1: types.StringNull(),
			Nameserver2: types.StringNull(),
			Nameserver3: types.StringNull(),
		}
		req := r.buildUpdateRequest(plan)
		if req.Nameserver1 != nil || req.Nameserver2 != nil || req.Nameserver3 != nil {
			t.Errorf("all null should produce empty request")
		}
	})
}

func TestDNSNameserverResource_MapResponseToModel(t *testing.T) {
	r := &DNSNameserverResource{}
	cases := []*client.NetworkConfig{
		{Nameserver1: "1.1.1.1", Nameserver2: "", Nameserver3: ""},
		{Nameserver1: "1.1.1.1", Nameserver2: "8.8.8.8", Nameserver3: ""},
		{Nameserver1: "1.1.1.1", Nameserver2: "8.8.8.8", Nameserver3: "9.9.9.9"},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m DNSNameserverResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Nameserver1.ValueString() != cfg.Nameserver1 {
				t.Errorf("case %d: NS1 mismatch", i)
			}
			if m.Nameserver2.ValueString() != cfg.Nameserver2 {
				t.Errorf("case %d: NS2 mismatch", i)
			}
			if m.Nameserver3.ValueString() != cfg.Nameserver3 {
				t.Errorf("case %d: NS3 mismatch", i)
			}
		})
	}
}

func TestNVMetGlobalResource_MapResponseToModel(t *testing.T) {
	r := &NVMetGlobalResource{}
	cases := []*client.NVMetGlobal{
		{Basenqn: "nqn.base"},
		{Basenqn: "nqn.rdma", Rdma: true},
		{Basenqn: "nqn.ak", Ana: true, Kernel: true},
		{Basenqn: "nqn.all", Kernel: true, Ana: true, Rdma: true, XportReferral: true},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m NVMetGlobalResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Basenqn.ValueString() != cfg.Basenqn {
				t.Errorf("case %d: Basenqn mismatch", i)
			}
			if m.Kernel.ValueBool() != cfg.Kernel {
				t.Errorf("case %d: Kernel mismatch", i)
			}
			if m.Ana.ValueBool() != cfg.Ana {
				t.Errorf("case %d: Ana mismatch", i)
			}
			if m.Rdma.ValueBool() != cfg.Rdma {
				t.Errorf("case %d: Rdma mismatch", i)
			}
		})
	}
}
