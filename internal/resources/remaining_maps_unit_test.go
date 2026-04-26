package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// These tests cover mapResponseToModel for resources that didn't yet have
// mapping tests. Each exercises 3-4 fixture cases to hit nil-pointer and
// value paths.

func TestACMEDNSAuthenticatorResource_MapResponseToModel(t *testing.T) {
	r := &ACMEDNSAuthenticatorResource{}
	ctx := context.Background()
	cases := []*client.ACMEDNSAuthenticator{
		{ID: 1, Name: "cloudflare-1", Attributes: map[string]interface{}{"authenticator": "cloudflare", "api_token": "abc"}},
		{ID: 2, Name: "route53", Attributes: map[string]interface{}{"authenticator": "route53", "access_key": "AK", "secret_key": "SK"}},
		{ID: 3, Name: "do", Attributes: map[string]interface{}{"authenticator": "digitalocean", "api_key": "x"}},
		{ID: 4, Name: "empty"},
	}
	for i, a := range cases {
		t.Run("case", func(t *testing.T) {
			var m ACMEDNSAuthenticatorResourceModel
			r.mapResponseToModel(ctx, a, &m)
			if m.Name.ValueString() != a.Name {
				t.Errorf("case %d: Name mismatch", i)
			}
			if m.ID.ValueString() == "" {
				t.Errorf("case %d: ID should be set", i)
			}
		})
	}
}

func TestAppResource_MapResponseToModel(t *testing.T) {
	r := &AppResource{}
	cases := []*client.App{
		{ID: "plex", Name: "plex", State: "RUNNING", HumanVersion: "1.0.0", UpgradeAvailable: false},
		{ID: "sonarr", Name: "sonarr", State: "STOPPED", HumanVersion: "4.0.0", UpgradeAvailable: true},
		{ID: "custom", Name: "custom", State: "RUNNING", CustomApp: true},
		{ID: "empty", Name: "empty"},
	}
	for i, a := range cases {
		t.Run("case", func(t *testing.T) {
			var m AppResourceModel
			r.mapResponseToModel(a, &m)
			if m.AppName.ValueString() != a.Name {
				t.Errorf("case %d: Name mismatch", i)
			}
			if m.State.ValueString() != a.State {
				t.Errorf("case %d: State mismatch", i)
			}
			if m.UpgradeAvailable.ValueBool() != a.UpgradeAvailable {
				t.Errorf("case %d: UpgradeAvailable mismatch", i)
			}
		})
	}
}

func TestCatalogResource_MapResponseToModel(t *testing.T) {
	r := &CatalogResource{}
	ctx := context.Background()
	cases := []*client.Catalog{
		{ID: "TRUENAS", Label: "TRUENAS", Location: "/mnt/tank/catalog", PreferredTrains: []string{"stable"}},
		{ID: "COMMUNITY", Label: "COMMUNITY", PreferredTrains: []string{"stable", "community"}},
		{ID: "EMPTY", Label: "EMPTY"},
		{ID: "MULTI", Label: "MULTI", PreferredTrains: []string{"stable", "incubator", "test", "enterprise"}},
	}
	for i, c := range cases {
		t.Run("case", func(t *testing.T) {
			var m CatalogResourceModel
			r.mapResponseToModel(ctx, c, &m)
			if m.Label.ValueString() != c.Label {
				t.Errorf("case %d: Label mismatch", i)
			}
			if got := len(m.PreferredTrains.Elements()); got != len(c.PreferredTrains) {
				t.Errorf("case %d: PreferredTrains len = %d, want %d", i, got, len(c.PreferredTrains))
			}
		})
	}
}

func TestCloudSyncCredentialResource_MapResponseToModel(t *testing.T) {
	r := &CloudSyncCredentialResource{}
	ctx := context.Background()
	cases := []*client.CloudSyncCredential{
		{ID: 1, Name: "s3-prod", Provider: map[string]interface{}{"type": "S3", "access_key_id": "k", "secret_access_key": "s"}},
		{ID: 2, Name: "b2", Provider: map[string]interface{}{"type": "B2", "account": "a"}},
		{ID: 3, Name: "blob", Provider: map[string]interface{}{"type": "AZUREBLOB", "account": "a"}},
	}
	for i, c := range cases {
		t.Run("case", func(t *testing.T) {
			var m CloudSyncCredentialResourceModel
			r.mapResponseToModel(ctx, c, &m)
			if m.Name.ValueString() != c.Name {
				t.Errorf("case %d: Name mismatch", i)
			}
			if m.ProviderType.ValueString() == "" {
				t.Errorf("case %d: ProviderType should be set", i)
			}
		})
	}
}

func TestDirectoryServicesResource_MapResponseToModel(t *testing.T) {
	r := &DirectoryServicesResource{}
	svcType := "ACTIVEDIRECTORY"
	realm := "CORP.EXAMPLE.COM"
	cases := []*client.DirectoryServicesConfig{
		{ID: 1, ServiceType: &svcType, Enable: true, EnableAccountCache: true, KerberosRealm: &realm, Timeout: 60},
		{ID: 1, Enable: false, Timeout: 60},
		{ID: 1, ServiceType: &svcType, Enable: true, EnableDNSUpdates: true, Timeout: 120},
		{ID: 1, Configuration: map[string]interface{}{"bindname": "admin"}, Enable: true},
	}
	for i, c := range cases {
		t.Run("case", func(t *testing.T) {
			var m DirectoryServicesResourceModel
			r.mapResponseToModel(c, &m)
			if m.Enable.ValueBool() != c.Enable {
				t.Errorf("case %d: Enable mismatch", i)
			}
		})
	}
}

func TestFilesystemACLTemplateResource_MapResponseToModel(t *testing.T) {
	r := &FilesystemACLTemplateResource{}
	cases := []*client.FilesystemACLTemplate{
		{ID: 1, Name: "home", ACLType: "POSIX1E", Comment: "home dir", ACL: json.RawMessage(`[{"tag":"USER","id":0,"perms":"rwx"}]`)},
		{ID: 2, Name: "dataset", ACLType: "NFS4", Builtin: true},
		{ID: 3, Name: "custom", ACLType: "NFS4", Comment: "custom template"},
	}
	for i, tpl := range cases {
		t.Run("case", func(t *testing.T) {
			var m FilesystemACLTemplateResourceModel
			r.mapResponseToModel(tpl, &m)
			if m.Name.ValueString() != tpl.Name {
				t.Errorf("case %d: Name mismatch", i)
			}
			if m.ACLType.ValueString() != tpl.ACLType {
				t.Errorf("case %d: ACLType mismatch", i)
			}
			if m.Builtin.ValueBool() != tpl.Builtin {
				t.Errorf("case %d: Builtin mismatch", i)
			}
		})
	}
}

func TestISCSITargetExtentResource_MapResponseToModel(t *testing.T) {
	r := &ISCSITargetExtentResource{}
	cases := []*client.ISCSITargetExtent{
		{ID: 1, Target: 1, Extent: 1, LunID: 0},
		{ID: 2, Target: 1, Extent: 2, LunID: 1},
		{ID: 3, Target: 2, Extent: 5, LunID: 10},
		{ID: 4, Target: 10, Extent: 100, LunID: 255},
	}
	for i, te := range cases {
		t.Run("case", func(t *testing.T) {
			var m ISCSITargetExtentResourceModel
			r.mapResponseToModel(te, &m)
			if m.Target.ValueInt64() != int64(te.Target) {
				t.Errorf("case %d: Target mismatch", i)
			}
			if m.Extent.ValueInt64() != int64(te.Extent) {
				t.Errorf("case %d: Extent mismatch", i)
			}
			if m.LunID.ValueInt64() != int64(te.LunID) {
				t.Errorf("case %d: LunID mismatch", i)
			}
		})
	}
}

func TestKerberosKeytabResource_MapResponseToModel(t *testing.T) {
	r := &KerberosKeytabResource{}
	cases := []*client.KerberosKeytab{
		{ID: 1, Name: "host", File: "BQIAAA=="},
		{ID: 2, Name: "webdav", File: ""},
		{ID: 3, Name: "service-principal", File: "base64payload"},
	}
	for i, kt := range cases {
		t.Run("case", func(t *testing.T) {
			var m KerberosKeytabResourceModel
			r.mapResponseToModel(kt, &m)
			if m.Name.ValueString() != kt.Name {
				t.Errorf("case %d: Name mismatch", i)
			}
		})
	}
}

func TestKerberosRealmResource_MapResponseToModel(t *testing.T) {
	r := &KerberosRealmResource{}
	ctx := context.Background()
	primaryKDC := "kdc1.corp.example.com"
	cases := []*client.KerberosRealm{
		{ID: 1, Realm: "CORP.EXAMPLE.COM", PrimaryKDC: &primaryKDC, KDC: []string{"kdc1", "kdc2"}, AdminServer: []string{"admin"}},
		{ID: 2, Realm: "OTHER.EXAMPLE.COM", KDC: []string{"kdc1"}},
		{ID: 3, Realm: "TEST.LOCAL"},
	}
	for i, rlm := range cases {
		t.Run("case", func(t *testing.T) {
			var m KerberosRealmResourceModel
			var d diag.Diagnostics
			r.mapResponseToModel(ctx, rlm, &m, &d)
			if m.Realm.ValueString() != rlm.Realm {
				t.Errorf("case %d: Realm mismatch", i)
			}
		})
	}
}

func TestKMIPConfigResource_MapResponseToModel(t *testing.T) {
	r := &KMIPConfigResource{}
	certID := 1
	caID := 2
	server := "kmip.example.com"
	cases := []*client.KMIPConfig{
		{ID: 1, Enabled: true, ManageSEDDisks: true, Certificate: &certID, CertificateAuthority: &caID, Port: 5696, Server: &server, SSLVersion: "PROTOCOL_TLSv1_2"},
		{ID: 1, Enabled: false, Port: 5696, SSLVersion: "PROTOCOL_TLSv1_2"},
		{ID: 1, Enabled: true, ManageZFSKeys: true, Certificate: &certID, Port: 5696, Server: &server, SSLVersion: "PROTOCOL_TLSv1_3"},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m KMIPConfigResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Enabled.ValueBool() != cfg.Enabled {
				t.Errorf("case %d: Enabled mismatch", i)
			}
			if m.Port.ValueInt64() != int64(cfg.Port) {
				t.Errorf("case %d: Port mismatch", i)
			}
		})
	}
}

func TestMailConfigResource_MapResponseToModel(t *testing.T) {
	r := &MailConfigResource{}
	user := "admin"
	cases := []*client.MailConfig{
		{ID: 1, FromEmail: "admin@example.com", FromName: "TrueNAS", OutgoingServer: "smtp.example.com", Port: 587, Security: "TLS", SMTP: true, User: &user, Pass: "p"},
		{ID: 1, FromEmail: "noauth@example.com", OutgoingServer: "smtp.internal", Port: 25, Security: "PLAIN", SMTP: false},
		{ID: 1, FromEmail: "ssl@example.com", OutgoingServer: "smtp.ssl.example.com", Port: 465, Security: "SSL", SMTP: true, User: &user},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m MailConfigResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.FromEmail.ValueString() != cfg.FromEmail {
				t.Errorf("case %d: FromEmail mismatch", i)
			}
			if m.Port.ValueInt64() != int64(cfg.Port) {
				t.Errorf("case %d: Port mismatch", i)
			}
		})
	}
}

func TestNetworkConfigResource_MapResponseToModel(t *testing.T) {
	r := &NetworkConfigResource{}
	ctx := context.Background()
	cases := []*client.FullNetworkConfig{
		{ID: 1, Hostname: "truenas", Domain: "local", IPv4Gateway: "10.0.0.1", Nameserver1: "1.1.1.1"},
		{ID: 1, Hostname: "fs01", Domain: "corp.example.com", IPv4Gateway: "192.168.1.1", IPv6Gateway: "fe80::1", Nameserver1: "8.8.8.8", Nameserver2: "8.8.4.4"},
		{ID: 1, Hostname: "nas", Domain: "", HTTPProxy: "http://proxy.example.com:8080", Hosts: []string{"10.0.0.1 host1", "10.0.0.2 host2"}},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m NetworkConfigResourceModel
			r.mapResponseToModel(ctx, cfg, &m)
			if m.Hostname.ValueString() != cfg.Hostname {
				t.Errorf("case %d: Hostname mismatch", i)
			}
			if m.Domain.ValueString() != cfg.Domain {
				t.Errorf("case %d: Domain mismatch", i)
			}
		})
	}
}

func TestNFSConfigResource_MapResponseToModel(t *testing.T) {
	r := &NFSConfigResource{}
	ctx := context.Background()
	mport := 600
	rport := 601
	lport := 602
	cases := []*client.NFSConfig{
		{ID: 1, Servers: 4, AllowNonroot: false, Protocols: []string{"NFSV3", "NFSV4"}, V4Krb: false},
		{ID: 1, Servers: 8, V4Krb: true, V4Domain: "CORP.EXAMPLE.COM", Protocols: []string{"NFSV4"}},
		{ID: 1, Servers: 2, BindIP: []string{"10.0.0.5"}, MountdPort: &mport, RpcstatdPort: &rport, RpclockdPort: &lport, Protocols: []string{"NFSV3"}},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m NFSConfigResourceModel
			r.mapResponseToModel(ctx, cfg, &m)
			if m.Servers.ValueInt64() != int64(cfg.Servers) {
				t.Errorf("case %d: Servers mismatch", i)
			}
			if m.V4Krb.ValueBool() != cfg.V4Krb {
				t.Errorf("case %d: V4Krb mismatch", i)
			}
		})
	}
}

func TestNVMetHostSubsysResource_MapResponseToModel(t *testing.T) {
	r := &NVMetHostSubsysResource{}
	cases := []*client.NVMetHostSubsys{
		{ID: 1, HostID: 10, SubsysID: 20},
		{ID: 2, Host: &client.NVMetHostSubsysHost{ID: 5}, Subsys: &client.NVMetHostSubsysSubsys{ID: 6}},
		{ID: 3, HostID: 100, SubsysID: 200},
	}
	for i, hs := range cases {
		t.Run("case", func(t *testing.T) {
			var m NVMetHostSubsysResourceModel
			r.mapResponseToModel(hs, &m)
			if m.ID.ValueString() == "" {
				t.Errorf("case %d: ID should be set", i)
			}
		})
	}
}

func TestNVMetNamespaceResource_MapResponseToModel(t *testing.T) {
	r := &NVMetNamespaceResource{}
	nsid1 := 1
	nsid2 := 255
	filesize := int64(1073741824)
	cases := []*client.NVMetNamespace{
		{ID: 1, Nsid: &nsid1, SubsysID: 5, DeviceType: "ZVOL", DevicePath: "zvol/tank/v1", Enabled: true},
		{ID: 2, Nsid: &nsid2, SubsysID: 10, DeviceType: "FILE", DevicePath: "/mnt/tank/file.img", Filesize: &filesize, Enabled: true},
		{ID: 3, SubsysID: 3, DeviceType: "ZVOL", DevicePath: "zvol/p/v", Enabled: false},
	}
	for i, ns := range cases {
		t.Run("case", func(t *testing.T) {
			var m NVMetNamespaceResourceModel
			r.mapResponseToModel(ns, &m)
			if m.DeviceType.ValueString() != ns.DeviceType {
				t.Errorf("case %d: DeviceType mismatch", i)
			}
			if m.Enabled.ValueBool() != ns.Enabled {
				t.Errorf("case %d: Enabled mismatch", i)
			}
		})
	}
}

func TestNVMetPortResource_MapResponseToModel(t *testing.T) {
	r := &NVMetPortResource{}
	inlineSize := 16384
	maxQ := 256
	piEnable := true
	cases := []*client.NVMetPort{
		{ID: 1, Index: 1, AddrTrtype: "TCP", AddrTraddr: "0.0.0.0", AddrTrsvcid: json.RawMessage("4420"), Enabled: true},
		{ID: 2, Index: 2, AddrTrtype: "TCP", AddrTraddr: "10.0.0.5", AddrTrsvcid: json.RawMessage("4420"), InlineDataSize: &inlineSize, MaxQueueSize: &maxQ, Enabled: true},
		{ID: 3, Index: 3, AddrTrtype: "RDMA", AddrTraddr: "10.0.0.6", AddrTrsvcid: json.RawMessage(`"4420"`), PiEnable: &piEnable, Enabled: false},
	}
	for i, p := range cases {
		t.Run("case", func(t *testing.T) {
			var m NVMetPortResourceModel
			r.mapResponseToModel(p, &m)
			if m.AddrTrtype.ValueString() != p.AddrTrtype {
				t.Errorf("case %d: AddrTrtype mismatch", i)
			}
			if m.AddrTraddr.ValueString() != p.AddrTraddr {
				t.Errorf("case %d: AddrTraddr mismatch", i)
			}
		})
	}
}

func TestNVMetPortSubsysResource_MapResponseToModel(t *testing.T) {
	r := &NVMetPortSubsysResource{}
	cases := []*client.NVMetPortSubsys{
		{ID: 1, PortID: 10, SubsysID: 20},
		{ID: 2, Port: &client.NVMetPortSubsysPort{ID: 5}, Subsys: &client.NVMetPortSubsysSubsys{ID: 6}},
		{ID: 3, PortID: 100, SubsysID: 200},
	}
	for i, ps := range cases {
		t.Run("case", func(t *testing.T) {
			var m NVMetPortSubsysResourceModel
			r.mapResponseToModel(ps, &m)
			if m.ID.ValueString() == "" {
				t.Errorf("case %d: ID should be set", i)
			}
		})
	}
}

func TestSNMPConfigResource_MapResponseToModel(t *testing.T) {
	r := &SNMPConfigResource{}
	privProto := "AES"
	privPass := "privpw"
	cases := []*client.SNMPConfig{
		{ID: 1, Community: "public", Contact: "admin@example.com", Location: "dc1", V3: false},
		{ID: 1, Community: "private", V3: true, V3Username: "user", V3AuthType: "SHA", V3Password: "pw", V3PrivProto: &privProto, V3PrivPassphrase: &privPass},
		{ID: 1, Location: "remote", V3: false},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m SNMPConfigResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Community.ValueString() != cfg.Community {
				t.Errorf("case %d: Community mismatch", i)
			}
			if m.V3.ValueBool() != cfg.V3 {
				t.Errorf("case %d: V3 mismatch", i)
			}
		})
	}
}

func TestSystemDatasetResource_MapResponseToModel(t *testing.T) {
	r := &SystemDatasetResource{}
	cases := []*client.SystemDataset{
		{ID: 1, Pool: "tank", PoolSet: true, UUID: "abc-123", Basename: "tank/.system", Path: "/mnt/tank/.system"},
		{ID: 1, Pool: "boot-pool", PoolSet: false, UUID: "def", Basename: "boot-pool/.system", Path: "/boot-pool/.system"},
		{ID: 1, Pool: "", PoolSet: false, UUID: "zero", Basename: "", Path: ""},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m SystemDatasetResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Pool.ValueString() != cfg.Pool {
				t.Errorf("case %d: Pool mismatch", i)
			}
			if m.PoolSet.ValueBool() != cfg.PoolSet {
				t.Errorf("case %d: PoolSet mismatch", i)
			}
			if m.UUID.ValueString() != cfg.UUID {
				t.Errorf("case %d: UUID mismatch", i)
			}
		})
	}
}

func TestUPSConfigResource_MapResponseToModel(t *testing.T) {
	r := &UPSConfigResource{}
	cases := []*client.UPSConfig{
		{ID: 1, Mode: "MASTER", Identifier: "ups", Driver: "usbhid-ups", Port: "auto", Shutdown: "BATT", ShutdownTimer: 30, Description: "primary"},
		{ID: 1, Mode: "SLAVE", RemoteHost: "10.0.0.1", RemotePort: 3493},
		{ID: 1, Mode: "MASTER", Identifier: "backup-ups", Port: "/dev/ttyS0", Shutdown: "LOWBATT"},
	}
	for i, cfg := range cases {
		t.Run("case", func(t *testing.T) {
			var m UPSConfigResourceModel
			r.mapResponseToModel(cfg, &m)
			if m.Mode.ValueString() != cfg.Mode {
				t.Errorf("case %d: Mode mismatch", i)
			}
			if m.RemotePort.ValueInt64() != int64(cfg.RemotePort) {
				t.Errorf("case %d: RemotePort mismatch", i)
			}
		})
	}
}

func TestVMwareResource_MapResponseToModel(t *testing.T) {
	r := &VMwareResource{}
	cases := []*client.VMware{
		{ID: 1, Datastore: "ds1", Filesystem: "tank/vm", Hostname: "vcenter.example.com", Username: "administrator@vsphere.local"},
		{ID: 2, Datastore: "ds2", Filesystem: "tank/nested/vm", Hostname: "esxi.example.com", Username: "root"},
		{ID: 3, Datastore: "backup", Filesystem: "tank/backup", Hostname: "backup-host", Username: "svc-backup"},
	}
	for i, v := range cases {
		t.Run("case", func(t *testing.T) {
			var m VMwareResourceModel
			r.mapResponseToModel(v, &m)
			if m.Datastore.ValueString() != v.Datastore {
				t.Errorf("case %d: Datastore mismatch", i)
			}
			if m.Hostname.ValueString() != v.Hostname {
				t.Errorf("case %d: Hostname mismatch", i)
			}
			if m.Username.ValueString() != v.Username {
				t.Errorf("case %d: Username mismatch", i)
			}
		})
	}
}

// boolTypeHelper forces compile use of types package.
var _ = types.StringValue
