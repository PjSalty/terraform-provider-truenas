package resources

// Batch CRUD coverage tests — batch 2. See zz_crud_batch_test.go for the
// approach. This file covers the cloud/task/replication families.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// --- CloudBackup ---

func TestCloudBackupResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":               1,
		"description":      "daily",
		"path":             "/mnt/tank/data",
		"credentials":      3,
		"attributes":       map[string]interface{}{"bucket": "b"},
		"pre_script":       "",
		"post_script":      "",
		"snapshot":         true,
		"include":          []interface{}{},
		"exclude":          []interface{}{},
		"args":             "",
		"enabled":          true,
		"password":         "",
		"keep_last":        5,
		"transfer_setting": "DEFAULT",
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

// --- CloudSync ---

func TestCloudSyncResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":            1,
		"description":   "sync",
		"path":          "/mnt/tank/data",
		"credentials":   3,
		"direction":     "PUSH",
		"transfer_mode": "COPY",
		"enabled":       true,
		"attributes":    map[string]interface{}{"bucket": "b"},
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

// --- CloudSyncCredential ---

func TestCloudSyncCredentialResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":       1,
		"name":     "s3",
		"provider": map[string]interface{}{"type": "S3", "access_key_id": "AK", "secret_access_key": "SK"},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &CloudSyncCredentialResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":                     str("s3"),
		"provider_type":            str("S3"),
		"provider_attributes_json": str(`{"access_key_id":"AK","secret_access_key":"SK"}`),
	})
}

// --- CronJob ---

func TestCronJobResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":          1,
		"user":        "root",
		"command":     "true",
		"description": "",
		"enabled":     true,
		"stdout":      true,
		"stderr":      true,
		"schedule": map[string]interface{}{
			"minute": "0", "hour": "*", "dom": "*", "month": "*", "dow": "*",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &CronJobResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"user":            str("root"),
		"command":         str("true"),
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

// --- RsyncTask ---

func TestRsyncTaskResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":              1,
		"path":            "/mnt/tank/src",
		"user":            "root",
		"remotehost":      "rsync.example.com",
		"remoteport":      22,
		"mode":            "SSH",
		"remotemodule":    "",
		"remotepath":      "/backup",
		"direction":       "PUSH",
		"desc":            "",
		"enabled":         true,
		"recursive":       true,
		"times":           true,
		"compress":        true,
		"archive":         false,
		"delete":          false,
		"quiet":           false,
		"preserveperm":    false,
		"preserveattr":    false,
		"delayupdates":    false,
		"extra":           []interface{}{},
		"ssh_credentials": nil,
		"schedule": map[string]interface{}{
			"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &RsyncTaskResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"path":            str("/mnt/tank/src"),
		"user":            str("root"),
		"remotehost":      str("rsync.example.com"),
		"mode":            str("SSH"),
		"direction":       str("PUSH"),
		"schedule_minute": str("0"),
		"schedule_hour":   str("2"),
		"schedule_dom":    str("*"),
		"schedule_month":  str("*"),
		"schedule_dow":    str("*"),
	})
}

// --- ScrubTask ---

func TestScrubTaskResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":          1,
		"pool":        1,
		"threshold":   35,
		"description": "",
		"enabled":     true,
		"schedule": map[string]interface{}{
			"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "7",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ScrubTaskResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
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

// --- SnapshotTask ---

func TestSnapshotTaskResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":             1,
		"dataset":        "tank/data",
		"recursive":      true,
		"lifetime_value": 2,
		"lifetime_unit":  "WEEK",
		"naming_schema":  "auto-%Y%m%d.%H%M",
		"enabled":        true,
		"allow_empty":    true,
		"exclude":        []interface{}{},
		"schedule": map[string]interface{}{
			"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*",
			"begin": "09:00", "end": "18:00",
		},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &SnapshotTaskResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
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

// --- Replication ---

func TestReplicationResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":                1,
		"name":              "repl",
		"direction":         "PUSH",
		"transport":         "SSH",
		"source_datasets":   []interface{}{"tank/data"},
		"target_dataset":    "backup/data",
		"recursive":         true,
		"auto":              true,
		"enabled":           true,
		"retention_policy":  "SOURCE",
		"ssh_credentials":   nil,
		"exclude":           []interface{}{},
		"naming_schema":     []interface{}{},
		"schedule":          nil,
		"restrict_schedule": nil,
		"life_time_value":   0,
		"life_time_unit":    "",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ReplicationResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
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

// --- ReportingExporter ---

func TestReportingExporterResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":         1,
		"name":       "graphite",
		"type":       "GRAPHITE",
		"enabled":    true,
		"attributes": map[string]interface{}{"host": "localhost", "port": 2003},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &ReportingExporterResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":            str("graphite"),
		"type":            str("GRAPHITE"),
		"enabled":         flag(true),
		"attributes_json": str(`{"host":"localhost","port":2003}`),
	})
}

// --- Privilege ---

func TestPrivilegeResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":           1,
		"name":         "admins",
		"builtin_name": nil,
		"web_shell":    false,
		"allowlist":    []interface{}{},
		"roles":        []interface{}{"FULL_ADMIN"},
		"ds_groups":    []interface{}{},
		"local_groups": []interface{}{map[string]interface{}{"id": 1000, "gid": 1000, "name": "admins"}},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &PrivilegeResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"name":         str("admins"),
		"web_shell":    flag(false),
		"local_groups": numList(1000),
		"roles":        strList("FULL_ADMIN"),
	})
}

// --- StaticRoute ---

func TestStaticRouteResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":          1,
		"destination": "10.0.0.0/24",
		"gateway":     "192.168.1.1",
		"description": "",
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &StaticRouteResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"destination": str("10.0.0.0/24"),
		"gateway":     str("192.168.1.1"),
	})
}

// --- Tunable ---

func TestTunableResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":      1,
		"var":     "net.core.somaxconn",
		"value":   "1024",
		"type":    "SYSCTL",
		"comment": "test",
		"enabled": true,
	}
	// Tunable has both /tunable (list) and /tunable/id/1 (single) GET endpoints.
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			_, _ = w.Write([]byte("true"))
			return
		}
		if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/tunable") {
			_ = json.NewEncoder(w).Encode([]interface{}{body})
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &TunableResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"var":     str("net.core.somaxconn"),
		"value":   str("1024"),
		"type":    str("SYSCTL"),
		"comment": str("test"),
		"enabled": flag(true),
	})
}

// --- VMware ---

func TestVMwareResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"id":          1,
		"hostname":    "vcenter.example.com",
		"username":    "admin",
		"password":    "secret",
		"filesystems": []interface{}{"tank/data"},
	}
	c, srv := newTestServerClient(t, jsonHandler(body))
	defer srv.Close()
	r := &VMwareResource{client: c}
	crudDrive(t, r, c, "1", map[string]tftypes.Value{
		"hostname": str("vcenter.example.com"),
		"username": str("admin"),
		"password": str("secret"),
	})
}
