package resources

// Batch CRUD coverage tests — batch 4. Covers the storage (dataset/zvol/pool),
// VM, certificate, and app resources. These have more complex wire formats;
// the tests rely on the handlers accumulating partial coverage via multiple
// endpoint patterns.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// datasetBody returns a realistic DatasetResponse map matching the client
// DatasetResponse struct (with PropertyValue wrappers).
func datasetBody(id, datasetType string) map[string]interface{} {
	pv := func(s string) map[string]interface{} {
		return map[string]interface{}{"value": s, "rawvalue": s, "source": "LOCAL"}
	}
	body := map[string]interface{}{
		"id":              id,
		"name":            id,
		"pool":            strings.SplitN(id, "/", 2)[0],
		"type":            datasetType,
		"mountpoint":      "/mnt/" + id,
		"comments":        pv(""),
		"user_properties": map[string]interface{}{},
		"quota":           map[string]interface{}{"parsed": "0", "rawvalue": "0", "source": "DEFAULT"},
		"refquota":        map[string]interface{}{"parsed": "0", "rawvalue": "0", "source": "DEFAULT"},
		"compression":     pv("lz4"),
		"atime":           pv("on"),
		"deduplication":   pv("off"),
		"sync":            pv("standard"),
		"snapdir":         pv("hidden"),
		"copies":          pv("1"),
		"readonly":        pv("off"),
		"recordsize":      pv("128K"),
		"share_type":      pv("GENERIC"),
	}
	if datasetType == "VOLUME" {
		body["volsize"] = map[string]interface{}{"parsed": "16777216", "rawvalue": "16777216", "source": "LOCAL"}
		body["volblocksize"] = pv("16K")
	}
	return body
}

// --- Dataset ---

func TestDatasetResource_CRUD(t *testing.T) {
	body := datasetBody("tank/data", "FILESYSTEM")
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &DatasetResource{client: c}
	crudDrive(t, r, c, "tank/data", map[string]tftypes.Value{
		"name": str("data"),
		"pool": str("tank"),
		"type": str("FILESYSTEM"),
	})
}

// --- Zvol ---

func TestZvolResource_CRUD(t *testing.T) {
	body := datasetBody("tank/vol1", "VOLUME")
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ZvolResource{client: c}
	crudDrive(t, r, c, "tank/vol1", map[string]tftypes.Value{
		"name":    str("vol1"),
		"pool":    str("tank"),
		"volsize": num(16777216),
	})
}

// --- Certificate ---

func TestCertificateResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":                  1,
		"name":                "c1",
		"type":                1,
		"create_type":         "CERTIFICATE_CREATE_IMPORTED",
		"certificate":         "-----BEGIN CERTIFICATE-----\nMII...\n-----END CERTIFICATE-----\n",
		"privatekey":          "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
		"key_type":            "RSA",
		"key_length":          2048,
		"digest_algorithm":    "SHA256",
		"lifetime":            365,
		"country":             "US",
		"state":               "CA",
		"city":                "SF",
		"organization":        "org",
		"organizational_unit": "unit",
		"email":               "admin@example.com",
		"common":              "example.com",
		"san":                 []interface{}{"example.com"},
		"DN":                  "CN=example.com",
		"from":                "2025-01-01",
		"until":               "2026-01-01",
		"expired":             false,
	}
	// Certificate Create/Update/Delete are job-based. Wire a handler that
	// returns a job id on POST/PUT/DELETE, responds to /core/get_jobs, and
	// returns the body on GET.
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
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":             str("c1"),
		"create_type":      str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate":      str("-----BEGIN CERTIFICATE-----\nMII...\n-----END CERTIFICATE-----\n"),
		"privatekey":       str("-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"),
		"key_type":         str("RSA"),
		"key_length":       num(2048),
		"digest_algorithm": str("SHA256"),
	})
}

// --- VM ---

func vmBody() map[string]interface{} {
	return map[string]interface{}{
		"id":                      1,
		"name":                    "vm1",
		"description":             "",
		"vcpus":                   2,
		"cores":                   1,
		"threads":                 1,
		"memory":                  2048,
		"min_memory":              0,
		"bootloader":              "UEFI",
		"bootloader_ovmf":         "OVMF_CODE.fd",
		"autostart":               false,
		"hide_from_msr":           false,
		"ensure_display_device":   true,
		"time":                    "LOCAL",
		"shutdown_timeout":        90,
		"arch_type":               "",
		"machine_type":            "",
		"uuid":                    "00000000-0000-0000-0000-000000000001",
		"command_line_args":       "",
		"cpu_mode":                "CUSTOM",
		"cpu_model":               "",
		"cpuset":                  "",
		"nodeset":                 "",
		"pin_vcpus":               false,
		"suspend_on_snapshot":     false,
		"trusted_platform_module": false,
		"hyperv_enlightenments":   false,
		"enable_secure_boot":      false,
		"status":                  map[string]interface{}{"state": "STOPPED"},
	}
}

func TestVMResource_CRUD(t *testing.T) {
	body := vmBody()
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &VMResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":    str("vm1"),
		"vcpus":   num(2),
		"cores":   num(1),
		"threads": num(1),
		"memory":  num(2048),
	})
}

// --- VMDevice ---

func TestVMDeviceResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":    1,
		"vm":    1,
		"dtype": "DISK",
		"order": 1001,
		"attributes": map[string]interface{}{
			"path": "/dev/zvol/tank/vol1",
			"type": "AHCI",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &VMDeviceResource{client: c}
	attrsVal := tftypes.NewValue(
		tftypes.Map{ElementType: tftypes.String},
		map[string]tftypes.Value{
			"path": tftypes.NewValue(tftypes.String, "/dev/zvol/tank/vol1"),
			"type": tftypes.NewValue(tftypes.String, "AHCI"),
		},
	)
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"vm":         num(1),
		"dtype":      str("DISK"),
		"order":      num(1001),
		"attributes": attrsVal,
	})
}

// --- Pool ---
//
// Pool Create is async (job-based) and returns a job ID. We use a custom
// handler that honors both the pool-create job and the get_jobs path.

func TestPoolResource_CRUD(t *testing.T) {
	poolBody := map[string]interface{}{
		"id":       1,
		"name":     "tank",
		"path":     "/mnt/tank",
		"status":   "ONLINE",
		"healthy":  true,
		"topology": map[string]interface{}{"data": []interface{}{}},
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "core/get_jobs") {
			_ = json.NewEncoder(w).Encode([]interface{}{
				map[string]interface{}{"id": 1, "state": "SUCCESS", "result": poolBody},
			})
			return
		}
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/pool") {
			_, _ = w.Write([]byte("1"))
			return
		}
		if req.Method == http.MethodDelete {
			_, _ = w.Write([]byte("true"))
			return
		}
		_ = json.NewEncoder(w).Encode(poolBody)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &PoolResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":                    str("tank"),
		"topology_json":           str(`{"data":[{"type":"STRIPE","disks":["sda"]}]}`),
		"encryption":              flag(false),
		"encryption_options_json": str(`{}`),
		"deduplication":           str("OFF"),
		"checksum":                str("SHA256"),
		"allow_duplicate_serials": flag(false),
	})
}

// --- App (async, string id) ---

func TestAppResource_CRUD(t *testing.T) {
	appBody := map[string]interface{}{
		"id":                "minio",
		"name":              "minio",
		"catalog_app":       "minio",
		"train":             "stable",
		"version":           "1.0.0",
		"state":             "RUNNING",
		"upgrade_available": false,
		"human_version":     "1.0.0",
		"config":            map[string]interface{}{},
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "core/get_jobs") {
			_ = json.NewEncoder(w).Encode([]interface{}{
				map[string]interface{}{"id": 1, "state": "SUCCESS", "result": appBody},
			})
			return
		}
		// Any POST/PUT/DELETE that returns a job id.
		if req.Method == http.MethodPost || req.Method == http.MethodDelete {
			if req.Method == http.MethodDelete {
				_, _ = w.Write([]byte("1"))
				return
			}
			_, _ = w.Write([]byte("1"))
			return
		}
		if req.Method == http.MethodPut {
			_, _ = w.Write([]byte("1"))
			return
		}
		_ = json.NewEncoder(w).Encode(appBody)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &AppResource{client: c}
	crudDrive(t, r, c, "minio", map[string]tftypes.Value{
		"app_name":    str("minio"),
		"catalog_app": str("minio"),
		"train":       str("stable"),
		"version":     str("1.0.0"),
		"values":      str("{}"),
	})
}

// --- NetworkConfig (singleton) ---

func TestNetworkConfigResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":                   1,
		"hostname":             "truenas",
		"hostname_b":           "",
		"hostname_virtual":     "",
		"domain":               "local",
		"domains":              []interface{}{},
		"nameserver1":          "1.1.1.1",
		"nameserver2":          "",
		"nameserver3":          "",
		"ipv4gateway":          "192.168.1.1",
		"ipv6gateway":          "",
		"httpproxy":            "",
		"netwait_enabled":      false,
		"netwait_ip":           []interface{}{},
		"hosts":                []interface{}{},
		"service_announcement": map[string]interface{}{},
		"activity":             map[string]interface{}{},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &NetworkConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"hostname":    str("truenas"),
		"domain":      str("local"),
		"ipv4gateway": str("192.168.1.1"),
	})
}

// --- NetworkInterface ---

func TestNetworkInterfaceResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":                       "eth0",
		"name":                     "eth0",
		"type":                     "PHYSICAL",
		"description":              "",
		"ipv4_dhcp":                false,
		"ipv6_auto":                false,
		"mtu":                      1500,
		"state":                    map[string]interface{}{"name": "eth0"},
		"aliases":                  []interface{}{},
		"failover_aliases":         []interface{}{},
		"failover_virtual_aliases": []interface{}{},
		"bridge_members":           []interface{}{},
		"lag_protocol":             "",
		"lag_ports":                []interface{}{},
		"vlan_parent_interface":    "",
		"vlan_tag":                 nil,
		"vlan_pcp":                 nil,
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "commit") || strings.Contains(req.URL.Path, "checkin") {
			_, _ = w.Write([]byte("null"))
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
	r := &NetworkInterfaceResource{client: c}
	crudDrive(t, r, c, "eth0", map[string]tftypes.Value{
		"name": str("eth0"),
		"type": str("PHYSICAL"),
	})
}

// --- KMIPConfig (singleton) ---

func TestKMIPConfigResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":                    1,
		"server":                "kmip.example.com",
		"port":                  5696,
		"certificate":           nil,
		"certificate_authority": nil,
		"manage_sed_disks":      false,
		"manage_zfs_keys":       false,
		"change_server":         false,
		"validate":              true,
		"enabled":               false,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &KMIPConfigResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"server": str("kmip.example.com"),
		"port":   num(5696),
	})
}

// --- SystemDataset (singleton) ---

func TestSystemDatasetResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":       1,
		"pool":     "tank",
		"uuid":     "abc123",
		"uuid_b":   "",
		"basename": ".system",
		"path":     "/mnt/tank/.system",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SystemDatasetResource{client: c}
	crudDrive(t, r, c, "systemdataset", map[string]tftypes.Value{
		"pool": str("tank"),
	})
}

// --- DirectoryServices (singleton) ---

func TestDirectoryServicesResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":                   1,
		"service_type":         "",
		"enable":               false,
		"enable_account_cache": true,
		"enable_dns_updates":   true,
		"timeout":              60,
		"kerberos_realm":       nil,
		"credential":           nil,
		"configuration":        nil,
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &DirectoryServicesResource{client: c}
	crudDrive(t, r, c, "directoryservices", map[string]tftypes.Value{
		"enable":       flag(false),
		"service_type": str(""),
	})
}

// --- Service (by name, not numeric id) ---

func TestServiceResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":      1,
		"service": "ssh",
		"enable":  true,
		"state":   "RUNNING",
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "start") || strings.Contains(req.URL.Path, "stop") || strings.Contains(req.URL.Path, "reload") {
			_, _ = w.Write([]byte("true"))
			return
		}
		if req.Method == http.MethodGet && (strings.HasSuffix(req.URL.Path, "/service") || strings.Contains(req.URL.Path, "service?")) {
			_ = json.NewEncoder(w).Encode([]interface{}{body})
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &ServiceResource{client: c}
	crudDrive(t, r, c, "ssh", map[string]tftypes.Value{
		"service": str("ssh"),
		"enable":  flag(true),
		"state":   str("RUNNING"),
	})
}

// --- FilesystemACL ---

func TestFilesystemACLResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"path":    "/mnt/tank/share",
		"uid":     1000,
		"gid":     1000,
		"acltype": "POSIX1E",
		"acl": []interface{}{
			map[string]interface{}{
				"tag":     "USER_OBJ",
				"id":      -1,
				"perms":   map[string]interface{}{"READ": true, "WRITE": true, "EXECUTE": true},
				"default": false,
			},
		},
		"trivial": true,
	}
	// GetFilesystemACL POSTs to /filesystem/getacl and expects a single object.
	// SetFilesystemACL POSTs to /filesystem/setacl and ignores response.
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost && strings.Contains(req.URL.Path, "setacl") {
			_, _ = w.Write([]byte("null"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &FilesystemACLResource{client: c}

	// Build a dacl entry
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

	crudDrive(t, r, c, "/mnt/tank/share", map[string]tftypes.Value{
		"path":    str("/mnt/tank/share"),
		"acltype": str("POSIX1E"),
		"uid":     num(1000),
		"gid":     num(1000),
		"dacl":    daclVal,
	})
}
