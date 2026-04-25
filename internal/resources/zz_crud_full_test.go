package resources

// Additional CRUD tests with all optional fields populated. These drive
// the branches in Create/Update that handle optional list/string/bool
// fields, raising coverage on the heavier Create functions.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// strList returns a list[string] tftypes value.
func strList(vals ...string) tftypes.Value {
	items := make([]tftypes.Value, len(vals))
	for i, v := range vals {
		items[i] = tftypes.NewValue(tftypes.String, v)
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, items)
}

// numList returns a list[number] tftypes value.
func numList(vals ...int64) tftypes.Value {
	items := make([]tftypes.Value, len(vals))
	for i, v := range vals {
		items[i] = tftypes.NewValue(tftypes.Number, v)
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, items)
}

// --- NFS share full values ---

func TestNFSShareResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":       1,
		"path":     "/mnt/tank/share",
		"comment":  "data",
		"hosts":    []interface{}{"h1", "h2"},
		"networks": []interface{}{"10.0.0.0/24"},
		"security": []interface{}{"SYS"},
		"readonly": true, "enabled": true,
		"maproot_user": "root", "maproot_group": "wheel",
		"mapall_user": "nobody", "mapall_group": "nogroup",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NFSShareResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"path":          str("/mnt/tank/share"),
		"comment":       str("data"),
		"hosts":         strList("h1", "h2"),
		"networks":      strList("10.0.0.0/24"),
		"security":      strList("SYS"),
		"readonly":      flag(true),
		"enabled":       flag(true),
		"maproot_user":  str("root"),
		"maproot_group": str("wheel"),
		"mapall_user":   str("nobody"),
		"mapall_group":  str("nogroup"),
	})
}

// --- SMB share full values ---

func TestSMBShareResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "path": "/mnt/tank/share", "name": "share", "comment": "data",
		"purpose": "NO_PRESET", "browsable": true, "readonly": true,
		"abe": true, "enabled": true, "hostsallow": []interface{}{"10.0.0.0/24"},
		"hostsdeny": []interface{}{"0.0.0.0/0"},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SMBShareResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"path":       str("/mnt/tank/share"),
		"name":       str("share"),
		"comment":    str("data"),
		"purpose":    str("NO_PRESET"),
		"browsable":  flag(true),
		"readonly":   flag(true),
		"abe":        flag(true),
		"enabled":    flag(true),
		"hostsallow": strList("10.0.0.0/24"),
		"hostsdeny":  strList("0.0.0.0/0"),
	})
}

// --- Dataset full ---

func TestDatasetResource_CRUD_Full(t *testing.T) {
	body := datasetBody("tank/data", "FILESYSTEM")
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &DatasetResource{client: c}
	crudDrive(t, r, c, "tank/data", map[string]tftypes.Value{
		"name":          str("data"),
		"pool":          str("tank"),
		"type":          str("FILESYSTEM"),
		"compression":   str("lz4"),
		"atime":         str("on"),
		"deduplication": str("off"),
		"quota":         num(0),
		"refquota":      num(0),
		"comments":      str(""),
		"sync":          str("standard"),
		"snapdir":       str("hidden"),
		"copies":        num(1),
		"readonly":      str("off"),
		"record_size":   str("128K"),
		"share_type":    str("GENERIC"),
	})
}

// --- Zvol full ---

func TestZvolResource_CRUD_Full(t *testing.T) {
	body := datasetBody("tank/vol1", "VOLUME")
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ZvolResource{client: c}
	crudDrive(t, r, c, "tank/vol1", map[string]tftypes.Value{
		"name":          str("vol1"),
		"pool":          str("tank"),
		"volsize":       num(16777216),
		"volblocksize":  str("16K"),
		"deduplication": str("off"),
		"compression":   str("lz4"),
		"comments":      str(""),
	})
}

// --- User full ---

func TestUserResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "uid": 1000, "username": "alice", "full_name": "Alice",
		"email": "alice@example.com", "home": "/home/alice", "shell": "/bin/bash",
		"locked": false, "smb": true,
		"group":             map[string]interface{}{"id": 100, "bsdgrp_gid": 100},
		"groups":            []interface{}{101},
		"sudo_commands":     []interface{}{"/bin/ls"},
		"sshpubkey":         "ssh-rsa AAA...",
		"password_disabled": false,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &UserResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"uid":               num(1000),
		"username":          str("alice"),
		"full_name":         str("Alice"),
		"email":             str("alice@example.com"),
		"home":              str("/home/alice"),
		"shell":             str("/bin/bash"),
		"group":             num(100),
		"password":          str("hunter2"),
		"locked":            flag(false),
		"smb":               flag(true),
		"groups":            numList(101),
		"sudo_commands":     strList("/bin/ls"),
		"sshpubkey":         str("ssh-rsa AAA..."),
		"password_disabled": flag(false),
	})
}

// --- Group full ---

func TestGroupResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "gid": 1000, "group": "users", "name": "users",
		"builtin": false, "smb": true, "sudo_commands": []interface{}{"/bin/ls"},
		"sudo_commands_nopasswd": []interface{}{"/bin/cat"},
		"users":                  []interface{}{1},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &GroupResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"gid":                    num(1000),
		"name":                   str("users"),
		"smb":                    flag(true),
		"sudo_commands":          strList("/bin/ls"),
		"sudo_commands_nopasswd": strList("/bin/cat"),
		"users":                  numList(1),
	})
}

// --- NVMet host full ---

func TestNVMetHostResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "hostnqn": "nqn.x", "dhchap_key": "key",
		"dhchap_ctrl_key": "ckey", "dhchap_dhgroup": "DH-G2",
		"dhchap_hash": "HMAC_SHA256",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetHostResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"hostnqn":         str("nqn.x"),
		"dhchap_key":      str("key"),
		"dhchap_ctrl_key": str("ckey"),
		"dhchap_dhgroup":  str("DH-G2"),
		"dhchap_hash":     str("HMAC_SHA256"),
	})
}

// --- NVMet subsys full ---

func TestNVMetSubsysResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "name": "tgt", "subnqn": "nqn.2020-01.truenas:tgt",
		"allow_any_host": true, "pi_enable": true, "serial": "SN", "ieee_oui": "00:11:22",
		"qix_max": 128, "ana": false,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetSubsysResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":           str("tgt"),
		"subnqn":         str("nqn.2020-01.truenas:tgt"),
		"allow_any_host": flag(true),
		"pi_enable":      flag(true),
		"serial":         str("SN"),
		"ieee_oui":       str("00:11:22"),
		"qix_max":        num(128),
		"ana":            flag(false),
	})
}

// --- NVMet port full ---

func TestNVMetPortResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "index": 1, "addr_trtype": "TCP", "addr_trsvcid": "4420",
		"addr_traddr": "0.0.0.0", "addr_adrfam": "IPV4", "inline_data_size": 16384,
		"max_queue_size": 128, "pi_enable": false, "enabled": true,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetPortResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"addr_trtype":      str("TCP"),
		"addr_trsvcid":     str("4420"),
		"addr_traddr":      str("0.0.0.0"),
		"addr_adrfam":      str("IPV4"),
		"inline_data_size": num(16384),
		"max_queue_size":   num(128),
		"pi_enable":        flag(false),
		"enabled":          flag(true),
	})
}

// --- NVMet namespace full ---

func TestNVMetNamespaceResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "nsid": 1, "subsys_id": 1, "device_type": "ZVOL",
		"device_path": "zvol/tank/vol1", "filesize": 16777216, "enabled": true,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetNamespaceResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"nsid":        num(1),
		"subsys_id":   num(1),
		"device_type": str("ZVOL"),
		"device_path": str("zvol/tank/vol1"),
		"filesize":    num(16777216),
		"enabled":     flag(true),
	})
}

// --- iSCSI target full ---

func TestISCSITargetResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "name": "tgt1", "alias": "alias1", "mode": "ISCSI",
		"groups": []interface{}{
			map[string]interface{}{"portal": 1, "initiator": 1, "authmethod": "CHAP", "auth": 1},
		},
		"auth_networks": []interface{}{"10.0.0.0/24"},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSITargetResource{client: c}

	// Build the Groups list as a list[object].
	groupType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"portal":      tftypes.Number,
		"initiator":   tftypes.Number,
		"auth_method": tftypes.String,
		"auth":        tftypes.Number,
	}}
	grp := tftypes.NewValue(groupType, map[string]tftypes.Value{
		"portal":      tftypes.NewValue(tftypes.Number, 1),
		"initiator":   tftypes.NewValue(tftypes.Number, 1),
		"auth_method": tftypes.NewValue(tftypes.String, "CHAP"),
		"auth":        tftypes.NewValue(tftypes.Number, 1),
	})
	groupsVal := tftypes.NewValue(
		tftypes.List{ElementType: groupType},
		[]tftypes.Value{grp},
	)
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":   str("tgt1"),
		"alias":  str("alias1"),
		"mode":   str("ISCSI"),
		"groups": groupsVal,
	})
}

// --- iSCSI extent full ---

func TestISCSIExtentResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "name": "e1", "type": "DISK", "path": "zvol/tank/vol1", "disk": "zvol/tank/vol1",
		"blocksize": 4096, "enabled": true, "comment": "data",
		"ro": true, "xen": true, "insecure_tpc": true, "filesize": "0",
		"serial": "abc123", "product_id": "pid", "naa": "0x6589cfc00000",
		"rpm": "SSD", "avail_threshold": 80,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIExtentResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":         str("e1"),
		"type":         str("DISK"),
		"disk":         str("zvol/tank/vol1"),
		"blocksize":    num(4096),
		"enabled":      flag(true),
		"comment":      str("data"),
		"ro":           flag(true),
		"xen":          flag(true),
		"insecure_tpc": flag(true),
	})
}

// --- iSCSI portal full ---

func TestISCSIPortalResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "tag": 1, "comment": "all",
		"listen": []interface{}{
			map[string]interface{}{"ip": "0.0.0.0", "port": 3260},
			map[string]interface{}{"ip": "::", "port": 3260},
		},
		"discovery_authmethod": "CHAP",
		"discovery_authgroup":  1,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIPortalResource{client: c}

	// Build the Listen list as list[object{ip,port}]
	listenType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"ip":   tftypes.String,
		"port": tftypes.Number,
	}}
	l1 := tftypes.NewValue(listenType, map[string]tftypes.Value{
		"ip":   tftypes.NewValue(tftypes.String, "0.0.0.0"),
		"port": tftypes.NewValue(tftypes.Number, 3260),
	})
	l2 := tftypes.NewValue(listenType, map[string]tftypes.Value{
		"ip":   tftypes.NewValue(tftypes.String, "::"),
		"port": tftypes.NewValue(tftypes.Number, 3260),
	})
	listensVal := tftypes.NewValue(
		tftypes.List{ElementType: listenType},
		[]tftypes.Value{l1, l2},
	)
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"comment": str("all"),
		"listen":  listensVal,
	})
}

// --- iSCSI initiator full ---

func TestISCSIInitiatorResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":         1,
		"initiators": []interface{}{"iqn.1994-05.com.redhat"},
		"comment":    "allow redhat",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIInitiatorResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"comment":    str("allow redhat"),
		"initiators": strList("iqn.1994-05.com.redhat"),
	})
}

// --- Certificate with create-imported and CSR variants ---

func TestCertificateResource_CRUD_CSR(t *testing.T) {
	body := map[string]interface{}{
		"id": 2, "name": "csr1", "type": 2,
		"create_type":      "CERTIFICATE_CREATE_CSR",
		"certificate":      "",
		"privatekey":       "",
		"key_type":         "RSA",
		"key_length":       2048,
		"digest_algorithm": "SHA256",
		"lifetime":         365,
		"country":          "US",
		"state":            "CA",
		"city":             "SF",
		"organization":     "org",
		"email":            "a@b.c",
		"common":           "example.com",
		"san":              []interface{}{"example.com", "www.example.com"},
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "core/get_jobs") {
			_ = json.NewEncoder(w).Encode([]interface{}{
				map[string]interface{}{"id": 1, "state": "SUCCESS", "result": body},
			})
			return
		}
		if req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodDelete {
			_, _ = w.Write([]byte("1"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &CertificateResource{client: c}
	crudDrive(t, r, c, "2", map[string]tftypes.Value{
		"name":                str("csr1"),
		"create_type":         str("CERTIFICATE_CREATE_CSR"),
		"key_type":            str("RSA"),
		"key_length":          num(2048),
		"digest_algorithm":    str("SHA256"),
		"lifetime":            num(365),
		"country":             str("US"),
		"state":               str("CA"),
		"city":                str("SF"),
		"organization":        str("org"),
		"organizational_unit": str("unit"),
		"email":               str("a@b.c"),
		"common":              str("example.com"),
		"san":                 strList("example.com", "www.example.com"),
	})
}

// --- Replication full ---

func TestReplicationResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":                         1,
		"name":                       "repl",
		"direction":                  "PUSH",
		"transport":                  "SSH",
		"source_datasets":            []interface{}{"tank/data"},
		"target_dataset":             "backup/data",
		"recursive":                  true,
		"auto":                       true,
		"enabled":                    true,
		"retention_policy":           "CUSTOM",
		"ssh_credentials":            1,
		"naming_schema":              []interface{}{"auto-%Y%m%d"},
		"also_include_naming_schema": []interface{}{"auto-2-%Y%m%d"},
		"lifetime_value":             2,
		"lifetime_unit":              "WEEK",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ReplicationResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":                       str("repl"),
		"direction":                  str("PUSH"),
		"transport":                  str("SSH"),
		"source_datasets":            strList("tank/data"),
		"target_dataset":             str("backup/data"),
		"recursive":                  flag(true),
		"auto":                       flag(true),
		"enabled":                    flag(true),
		"retention_policy":           str("CUSTOM"),
		"ssh_credentials":            num(1),
		"naming_schema":              strList("auto-%Y%m%d"),
		"also_include_naming_schema": strList("auto-2-%Y%m%d"),
		"lifetime_value":             num(2),
		"lifetime_unit":              str("WEEK"),
	})
}

// --- ACME DNS authenticator with attributes populated ---

func TestACMEDNSAuthenticatorResource_CRUD_WithAttrs(t *testing.T) {
	body := map[string]interface{}{
		"id":         8,
		"name":       "cf2",
		"attributes": map[string]interface{}{"authenticator": "cloudflare", "api_token": "t"},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ACMEDNSAuthenticatorResource{client: c}
	// Build a populated attributes map
	attrsVal := tftypes.NewValue(
		tftypes.Map{ElementType: tftypes.String},
		map[string]tftypes.Value{
			"api_token": tftypes.NewValue(tftypes.String, "t"),
		},
	)
	crudDrive(t, r, c, "8", map[string]tftypes.Value{
		"name":          str("cf2"),
		"authenticator": str("cloudflare"),
		"attributes":    attrsVal,
	})
}

// --- CloudSyncCredential with invalid JSON (error branch) ---

func TestCloudSyncCredentialResource_CRUD_InvalidJSON(t *testing.T) {
	body := map[string]interface{}{
		"id":       1,
		"name":     "s3",
		"provider": map[string]interface{}{"type": "S3"},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &CloudSyncCredentialResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":                     str("s3"),
		"provider_type":            str("S3"),
		"provider_attributes_json": str(`{malformed`), // triggers JSON parse error
	})
}

// --- Keychain credential with SSH_CREDENTIALS type (different shape) ---

func TestKeychainCredentialResource_CRUD_SSHCreds(t *testing.T) {
	body := map[string]interface{}{
		"id":         2,
		"name":       "creds",
		"type":       "SSH_CREDENTIALS",
		"attributes": map[string]interface{}{"host": "backup.example.com", "port": 22, "username": "backup", "private_key": 1, "remote_host_key": "ssh-ed25519 AAA..."},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &KeychainCredentialResource{client: c}
	attrsVal := tftypes.NewValue(
		tftypes.Map{ElementType: tftypes.String},
		map[string]tftypes.Value{
			"host":            tftypes.NewValue(tftypes.String, "backup.example.com"),
			"port":            tftypes.NewValue(tftypes.String, "22"),
			"username":        tftypes.NewValue(tftypes.String, "backup"),
			"private_key":     tftypes.NewValue(tftypes.String, "1"),
			"remote_host_key": tftypes.NewValue(tftypes.String, "ssh-ed25519 AAA..."),
		},
	)
	crudDrive(t, r, c, "2", map[string]tftypes.Value{
		"name":       str("creds"),
		"type":       str("SSH_CREDENTIALS"),
		"attributes": attrsVal,
	})
}

// --- Network interface full ---

func TestNetworkInterfaceResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":                       "bridge0",
		"name":                     "bridge0",
		"type":                     "BRIDGE",
		"description":              "bridge",
		"ipv4_dhcp":                true,
		"ipv6_auto":                true,
		"mtu":                      1500,
		"state":                    map[string]interface{}{"name": "bridge0"},
		"aliases":                  []interface{}{},
		"bridge_members":           []interface{}{"eth0"},
		"lag_protocol":             "LACP",
		"lag_ports":                []interface{}{"eth1"},
		"vlan_parent_interface":    "eth2",
		"vlan_tag":                 10,
		"vlan_pcp":                 0,
		"failover_aliases":         []interface{}{},
		"failover_virtual_aliases": []interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NetworkInterfaceResource{client: c}
	crudDrive(t, r, c, "bridge0", map[string]tftypes.Value{
		"name":                  str("bridge0"),
		"type":                  str("BRIDGE"),
		"description":           str("bridge"),
		"ipv4_dhcp":             flag(true),
		"ipv6_auto":             flag(true),
		"mtu":                   num(1500),
		"bridge_members":        strList("eth0"),
		"lag_protocol":          str("LACP"),
		"lag_ports":             strList("eth1"),
		"vlan_parent_interface": str("eth2"),
		"vlan_tag":              num(10),
		"vlan_pcp":              num(0),
	})
}

// --- Service with start failure ---

func TestServiceResource_CRUD_StartFail(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "service": "ssh", "enable": true, "state": "RUNNING",
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		// Return 400 on /service/start to hit the Start error branch.
		if strings.Contains(req.URL.Path, "service/start") {
			http.Error(w, "no", http.StatusBadRequest)
			return
		}
		if strings.HasSuffix(req.URL.Path, "/service") && req.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]interface{}{body})
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &ServiceResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(true),
	})
}

// --- Service with enable=false (covers the Stop branch) ---

func TestServiceResource_CRUD_Disable(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "service": "ssh", "enable": false, "state": "STOPPED",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ServiceResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(false),
	})
}

// --- CloudBackup full with include/exclude ---

func TestCloudBackupResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":               1,
		"description":      "daily",
		"path":             "/mnt/tank/data",
		"credentials":      3,
		"attributes":       map[string]interface{}{"bucket": "b"},
		"pre_script":       "pre",
		"post_script":      "post",
		"snapshot":         true,
		"include":          []interface{}{"*.dat"},
		"exclude":          []interface{}{"*.tmp"},
		"args":             "--fast",
		"enabled":          true,
		"password":         "p",
		"keep_last":        5,
		"transfer_setting": "FAST",
		"schedule": map[string]interface{}{
			"minute": "0", "hour": "1", "dom": "*", "month": "*", "dow": "*",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &CloudBackupResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"description":      str("daily"),
		"path":             str("/mnt/tank/data"),
		"credentials":      num(3),
		"attributes_json":  str(`{"bucket":"b"}`),
		"pre_script":       str("pre"),
		"post_script":      str("post"),
		"snapshot":         flag(true),
		"include":          strList("*.dat"),
		"exclude":          strList("*.tmp"),
		"args":             str("--fast"),
		"enabled":          flag(true),
		"password":         str("p"),
		"keep_last":        num(5),
		"transfer_setting": str("FAST"),
		"schedule_minute":  str("0"),
		"schedule_hour":    str("1"),
		"schedule_dom":     str("*"),
		"schedule_month":   str("*"),
		"schedule_dow":     str("*"),
	})
}

// --- CloudSync full ---

func TestCloudSyncResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":            1,
		"description":   "sync",
		"path":          "/mnt/tank/data",
		"credentials":   3,
		"direction":     "PULL",
		"transfer_mode": "MOVE",
		"enabled":       true,
		"attributes":    map[string]interface{}{"bucket": "b", "prefix": "pfx"},
		"pre_script":    "pre",
		"post_script":   "post",
		"snapshot":      true,
		"include":       []interface{}{"*.dat"},
		"exclude":       []interface{}{"*.tmp"},
		"args":          "--fast",
		"schedule": map[string]interface{}{
			"minute": "0", "hour": "1", "dom": "*", "month": "*", "dow": "*",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &CloudSyncResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"description":     str("sync"),
		"path":            str("/mnt/tank/data"),
		"credentials":     num(3),
		"direction":       str("PULL"),
		"transfer_mode":   str("MOVE"),
		"enabled":         flag(true),
		"attributes_json": str(`{"bucket":"b","prefix":"pfx"}`),
		"pre_script":      str("pre"),
		"post_script":     str("post"),
		"snapshot":        flag(true),
		"include":         strList("*.dat"),
		"exclude":         strList("*.tmp"),
		"args":            str("--fast"),
		"schedule_minute": str("0"),
		"schedule_hour":   str("1"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	})
}

// --- Network config full ---

func TestNetworkConfigResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":                   1,
		"hostname":             "truenas",
		"hostname_b":           "",
		"hostname_virtual":     "",
		"domain":               "local",
		"domains":              []interface{}{"local"},
		"nameserver1":          "1.1.1.1",
		"nameserver2":          "1.0.0.1",
		"nameserver3":          "",
		"ipv4gateway":          "192.168.1.1",
		"ipv6gateway":          "fe80::1",
		"httpproxy":            "http://proxy:8080",
		"netwait_enabled":      true,
		"netwait_ip":           []interface{}{"1.1.1.1"},
		"hosts":                []interface{}{"192.168.1.100 truenas"},
		"service_announcement": map[string]interface{}{},
		"activity":             map[string]interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NetworkConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"hostname":        str("truenas"),
		"domain":          str("local"),
		"nameserver1":     str("1.1.1.1"),
		"nameserver2":     str("1.0.0.1"),
		"ipv4gateway":     str("192.168.1.1"),
		"ipv6gateway":     str("fe80::1"),
		"httpproxy":       str("http://proxy:8080"),
		"netwait_enabled": flag(true),
		"netwait_ip":      strList("1.1.1.1"),
		"hosts":           strList("192.168.1.100 truenas"),
	})
}

// --- Privilege full ---

func TestPrivilegeResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":           1,
		"name":         "admins",
		"builtin_name": nil,
		"web_shell":    true,
		"allowlist":    []interface{}{},
		"roles":        []interface{}{"FULL_ADMIN", "READONLY_ADMIN"},
		"ds_groups":    []interface{}{},
		"local_groups": []interface{}{
			map[string]interface{}{"id": 1000, "gid": 1000, "name": "admins"},
			map[string]interface{}{"id": 1001, "gid": 1001, "name": "ops"},
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &PrivilegeResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":         str("admins"),
		"web_shell":    flag(true),
		"local_groups": numList(1000, 1001),
		"roles":        strList("FULL_ADMIN", "READONLY_ADMIN"),
	})
}

// --- Certificate with CSR create + imported-signed ---

func TestCertificateResource_CRUD_Signed(t *testing.T) {
	body := map[string]interface{}{
		"id": 3, "name": "signed", "type": 1,
		"create_type":         "CERTIFICATE_CREATE_IMPORTED",
		"certificate":         "-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n",
		"privatekey":          "-----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----\n",
		"key_type":            "EC",
		"key_length":          256,
		"digest_algorithm":    "SHA384",
		"lifetime":            90,
		"country":             "US",
		"state":               "CA",
		"city":                "LA",
		"organization":        "org",
		"organizational_unit": "unit",
		"email":               "a@b.c",
		"common":              "example.com",
		"san":                 []interface{}{"example.com"},
		"DN":                  "CN=example.com",
		"from":                "2025-01-01",
		"until":               "2026-01-01",
		"expired":             false,
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "core/get_jobs") {
			_ = json.NewEncoder(w).Encode([]interface{}{
				map[string]interface{}{"id": 1, "state": "SUCCESS", "result": body},
			})
			return
		}
		if req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodDelete {
			_, _ = w.Write([]byte("1"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &CertificateResource{client: c}
	crudDrive(t, r, c, "3", map[string]tftypes.Value{
		"name":             str("signed"),
		"create_type":      str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate":      str("-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n"),
		"privatekey":       str("-----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----\n"),
		"key_type":         str("EC"),
		"key_length":       num(256),
		"digest_algorithm": str("SHA384"),
	})
}

// --- Reporting exporter invalid JSON error ---

func TestReportingExporterResource_CRUD_InvalidJSON(t *testing.T) {
	body := map[string]interface{}{
		"id":         1,
		"name":       "broken",
		"type":       "GRAPHITE",
		"enabled":    true,
		"attributes": map[string]interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ReportingExporterResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":            str("broken"),
		"type":            str("GRAPHITE"),
		"enabled":         flag(true),
		"attributes_json": str(`{malformed`),
	})
}

// --- Reporting exporter full ---

func TestReportingExporterResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":         1,
		"name":       "graphite",
		"type":       "GRAPHITE",
		"enabled":    true,
		"attributes": map[string]interface{}{"host": "localhost", "port": 2003, "prefix": "truenas", "interval": 60},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ReportingExporterResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":            str("graphite"),
		"type":            str("GRAPHITE"),
		"enabled":         flag(true),
		"attributes_json": str(`{"host":"localhost","port":2003,"prefix":"truenas","interval":60}`),
	})
}

// --- Catalog full ---

func TestCatalogResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"id":               "TRUENAS",
		"label":            "TRUENAS",
		"preferred_trains": []interface{}{"stable", "enterprise"},
		"location":         "/mnt/tank/catalog",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &CatalogResource{client: c}
	crudDrive(t, r, c, "TRUENAS", map[string]tftypes.Value{
		"label":            str("TRUENAS"),
		"preferred_trains": strList("stable", "enterprise"),
		"location":         str("/mnt/tank/catalog"),
		"sync_on_create":   flag(true),
	})
}

// --- Filesystem ACL full ---

func TestFilesystemACLResource_CRUD_Full(t *testing.T) {
	body := map[string]interface{}{
		"path":        "/mnt/tank/share",
		"uid":         1000,
		"gid":         1000,
		"acltype":     "NFS4",
		"acl":         []interface{}{},
		"nfs41_flags": map[string]interface{}{"autoinherit": true, "protected": true},
		"flags":       map[string]interface{}{"setuid": true, "setgid": true, "sticky": true},
		"trivial":     false,
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "core/get_jobs") {
			_ = json.NewEncoder(w).Encode([]interface{}{
				map[string]interface{}{"id": 1, "state": "SUCCESS", "result": body},
			})
			return
		}
		if req.Method == http.MethodPost {
			_, _ = w.Write([]byte("1"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &FilesystemACLResource{client: c}
	crudDrive(t, r, c, "/mnt/tank/share", map[string]tftypes.Value{
		"path":     str("/mnt/tank/share"),
		"acltype":  str("NFS4"),
		"uid":      num(1000),
		"gid":      num(1000),
		"acl_json": str(`[]`),
	})
}
