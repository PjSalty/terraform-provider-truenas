package resources

// Batch CRUD coverage tests — batch 3. Covers the iscsi/nvmet/share families
// and remaining singletons.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// --- FTPConfig (singleton) — Create just echoes through Update; no separate fn ---

func TestFTPConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "port": 21, "clients": 32, "ipconnections": 8, "loginattempt": 3,
		"timeout": 120, "onlyanonymous": false, "onlylocal": false, "banner": "",
		"filemask": "077", "dirmask": "077", "fxp": false, "resume": false,
		"defaultroot": true, "tls": false,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &FTPConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"port":          num(21),
		"clients":       num(32),
		"ipconnections": num(8),
		"loginattempt":  num(3),
		"timeout":       num(120),
	})
}

func TestMailConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "fromemail": "admin@example.com", "fromname": "TrueNAS",
		"outgoingserver": "smtp.example.com", "port": 587, "security": "TLS",
		"smtp": true, "user": "admin", "pass": "",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &MailConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"fromemail":       str("admin@example.com"),
		"fromname":        str("TrueNAS"),
		"outgoing_server": str("smtp.example.com"),
		"port":            num(587),
		"security":        str("TLS"),
		"smtp":            flag(true),
		"user":            str("admin"),
	})
}

func TestNFSConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "servers": 4, "allow_nonroot": false,
		"protocols": []interface{}{"NFSV3"}, "v4_krb": false, "v4_domain": "",
		"bindip": []interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NFSConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"servers":       num(4),
		"allow_nonroot": flag(false),
	})
}

func TestSMBConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "netbiosname": "NAS", "workgroup": "WG",
		"description": "", "enable_smb1": false, "unixcharset": "UTF-8",
		"aapl_extensions": false, "guest": "nobody", "filemask": "0775", "dirmask": "0775",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SMBConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"netbiosname": str("NAS"),
		"workgroup":   str("WG"),
	})
}

func TestSNMPConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "community": "public", "contact": "admin", "location": "dc1",
		"v3": false, "v3_username": "", "v3_authtype": "", "v3_password": "",
		"v3_privproto": nil, "v3_privpassphrase": nil,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SNMPConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"community": str("public"),
		"contact":   str("admin"),
		"location":  str("dc1"),
	})
}

func TestSSHConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "tcpport": 22, "passwordauth": true, "kerberosauth": false,
		"tcpfwd": true, "compression": false, "sftp_log_level": "",
		"sftp_log_facility": "", "weak_ciphers": []interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SSHConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"tcpport":      num(22),
		"passwordauth": flag(true),
	})
}

func TestUPSConfigResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "mode": "MASTER", "identifier": "ups", "driver": "usbhid-ups",
		"port": "auto", "remotehost": "", "remoteport": 3493,
		"shutdown": "BATT", "shutdowntimer": 30, "description": "",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &UPSConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"mode":       str("MASTER"),
		"identifier": str("ups"),
		"driver":     str("usbhid-ups"),
	})
}

// --- iSCSI family: auth/extent/initiator/portal/target/targetextent ---

func TestISCSIAuthResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "tag": 1, "user": "chap", "secret": "abcdefghi1234567",
		"peeruser": "", "peersecret": "", "discovery_auth": "NONE",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIAuthResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"tag":    num(1),
		"user":   str("chap"),
		"secret": str("abcdefghi1234567"),
	})
}

func TestISCSIExtentResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "name": "e1", "type": "FILE", "path": "/mnt/tank/e1",
		"blocksize": 512, "enabled": true, "comment": "",
		"ro": false, "xen": false, "insecure_tpc": false, "filesize": "0",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIExtentResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":      str("e1"),
		"type":      str("FILE"),
		"path":      str("/mnt/tank/e1"),
		"blocksize": num(512),
		"enabled":   flag(true),
	})
}

func TestISCSIInitiatorResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "initiators": []interface{}{}, "comment": "all",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIInitiatorResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"comment": str("all"),
	})
}

func TestISCSIPortalResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "tag": 1, "comment": "",
		"listen": []interface{}{map[string]interface{}{"ip": "0.0.0.0", "port": 3260}},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSIPortalResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"comment": str(""),
	})
}

func TestISCSITargetResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "name": "tgt1", "alias": "", "mode": "ISCSI",
		"groups":        []interface{}{},
		"auth_networks": []interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSITargetResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name": str("tgt1"),
		"mode": str("ISCSI"),
	})
}

func TestISCSITargetExtentResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "target": 1, "extent": 1, "lunid": 0,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ISCSITargetExtentResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"target": num(1),
		"extent": num(1),
	})
}

// --- NVMet family ---

func TestNVMetGlobalResource_CreateOnly(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "basenqn": "nqn.2020-01.truenas", "kernel": true,
		"ana": false, "rdma": false, "xport_referral": true,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetGlobalResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"basenqn": str("nqn.2020-01.truenas"),
		"kernel":  flag(true),
	})
}

func TestNVMetHostResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "hostnqn": "nqn.x",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetHostResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"hostnqn": str("nqn.x"),
	})
}

func TestNVMetHostSubsysResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "host_id": 1, "subsys_id": 1,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetHostSubsysResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"host_id":   num(1),
		"subsys_id": num(1),
	})
}

func TestNVMetNamespaceResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "nsid": 1, "subsys_id": 1, "device_type": "ZVOL",
		"device_path": "zvol/tank/vol1", "filesize": 0, "enabled": true,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetNamespaceResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"subsys_id":   num(1),
		"device_type": str("ZVOL"),
		"device_path": str("zvol/tank/vol1"),
	})
}

func TestNVMetPortResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "index": 1, "addr_trtype": "TCP", "addr_trsvcid": "4420",
		"addr_traddr": "0.0.0.0", "addr_adrfam": "IPV4", "inline_data_size": 16384, "enabled": true,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetPortResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"addr_trtype":  str("TCP"),
		"addr_trsvcid": str("4420"),
		"addr_traddr":  str("0.0.0.0"),
	})
}

func TestNVMetPortSubsysResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "port_id": 1, "subsys_id": 1,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetPortSubsysResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"port_id":   num(1),
		"subsys_id": num(1),
	})
}

func TestNVMetSubsysResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "name": "tgt", "subnqn": "nqn.2020-01.truenas:tgt",
		"allow_any_host": false, "serial": "SN", "ieee_oui": "", "pi_enable": false,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NVMetSubsysResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":           str("tgt"),
		"allow_any_host": flag(false),
	})
}

// --- Shares ---

func TestNFSShareResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "path": "/mnt/tank/share", "comment": "",
		"hosts": []interface{}{}, "networks": []interface{}{}, "security": []interface{}{},
		"readonly": false, "enabled": true,
		"maproot_user": "", "maproot_group": "", "mapall_user": "", "mapall_group": "",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NFSShareResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"path":     str("/mnt/tank/share"),
		"readonly": flag(false),
		"enabled":  flag(true),
	})
}

func TestSMBShareResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "path": "/mnt/tank/share", "name": "share", "comment": "",
		"purpose": "NO_PRESET", "browsable": true, "readonly": false,
		"abe": false, "enabled": true, "hostsallow": []interface{}{}, "hostsdeny": []interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SMBShareResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"path":      str("/mnt/tank/share"),
		"name":      str("share"),
		"purpose":   str("NO_PRESET"),
		"browsable": flag(true),
		"readonly":  flag(false),
		"enabled":   flag(true),
	})
}

// --- User/Group ---

func TestUserResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "uid": 1000, "username": "alice", "full_name": "Alice",
		"email": nil, "home": "/home/alice", "shell": "/bin/bash",
		"locked": false, "smb": true,
		"group":             map[string]interface{}{"id": 100, "bsdgrp_gid": 100},
		"groups":            []interface{}{},
		"sudo_commands":     []interface{}{},
		"sshpubkey":         nil,
		"password_disabled": false,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &UserResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"uid":       num(1000),
		"username":  str("alice"),
		"full_name": str("Alice"),
		"home":      str("/home/alice"),
		"shell":     str("/bin/bash"),
		"group":     num(100),
		"password":  str("hunter2"),
	})
}

func TestGroupResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id": 1, "gid": 1000, "group": "users", "name": "users",
		"builtin": false, "smb": false, "sudo_commands": []interface{}{},
		"sudo_commands_nopasswd": []interface{}{},
		"users":                  []interface{}{},
	}
	// CreateGroup POST returns a bare int ID; GetGroup returns the full object.
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/group") {
			_, _ = w.Write([]byte("1"))
			return
		}
		if req.Method == http.MethodDelete {
			_, _ = w.Write([]byte("true"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &GroupResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"gid":  num(1000),
		"name": str("users"),
	})
}
