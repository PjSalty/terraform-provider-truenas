package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestCloudBackupResource_MapResponseToModel_Cases(t *testing.T) {
	r := &CloudBackupResource{}
	ctx := context.Background()
	cases := []struct {
		name          string
		cb            *client.CloudBackup
		wantPath      string
		wantCredID    int64
		wantEnabled   bool
		wantKeepLast  int64
		wantSchedMin  string
		wantSchedHour string
	}{
		{
			name: "minimal task",
			cb: &client.CloudBackup{
				ID:          1,
				Description: "nightly",
				Path:        "/mnt/tank/data",
				Credentials: json.RawMessage("3"),
				Schedule:    client.CloudBackupSchedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"},
				Enabled:     true,
				KeepLast:    10,
			},
			wantPath: "/mnt/tank/data", wantCredID: 3, wantEnabled: true, wantKeepLast: 10,
			wantSchedMin: "0", wantSchedHour: "2",
		},
		{
			name: "expanded credentials object",
			cb: &client.CloudBackup{
				ID:          2,
				Path:        "/mnt/tank/backups",
				Credentials: json.RawMessage(`{"id":7,"name":"s3-creds","provider":"S3"}`),
				Schedule:    client.CloudBackupSchedule{Minute: "30", Hour: "3", Dom: "*", Month: "*", Dow: "*"},
				Enabled:     false,
				KeepLast:    5,
				Attributes:  json.RawMessage(`{"bucket":"prod","prefix":"tnas"}`),
			},
			wantPath: "/mnt/tank/backups", wantCredID: 7, wantEnabled: false, wantKeepLast: 5,
			wantSchedMin: "30", wantSchedHour: "3",
		},
		{
			name: "with include/exclude lists",
			cb: &client.CloudBackup{
				ID:          3,
				Path:        "/mnt/tank/home",
				Credentials: json.RawMessage("1"),
				Include:     []string{"/mnt/tank/home/data"},
				Exclude:     []string{"/mnt/tank/home/tmp"},
				Schedule:    client.CloudBackupSchedule{Minute: "0", Hour: "1", Dom: "*", Month: "*", Dow: "*"},
				Enabled:     true,
				KeepLast:    30,
			},
			wantPath: "/mnt/tank/home", wantCredID: 1, wantEnabled: true, wantKeepLast: 30,
			wantSchedMin: "0", wantSchedHour: "1",
		},
		{
			name: "with pre/post scripts and transfer setting",
			cb: &client.CloudBackup{
				ID:              4,
				Path:            "/mnt/tank/db",
				Credentials:     json.RawMessage("2"),
				PreScript:       "/usr/local/bin/pre.sh",
				PostScript:      "/usr/local/bin/post.sh",
				Snapshot:        true,
				Args:            "--fast",
				Schedule:        client.CloudBackupSchedule{Minute: "*/15", Hour: "*", Dom: "*", Month: "*", Dow: "*"},
				Enabled:         true,
				KeepLast:        7,
				TransferSetting: "FAST_STORAGE",
			},
			wantPath: "/mnt/tank/db", wantCredID: 2, wantEnabled: true, wantKeepLast: 7,
			wantSchedMin: "*/15", wantSchedHour: "*",
		},
		{
			name: "zero credentials id",
			cb: &client.CloudBackup{
				ID:          5,
				Path:        "/mnt/x",
				Credentials: json.RawMessage("0"),
				Schedule:    client.CloudBackupSchedule{Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "*"},
				Enabled:     false,
				KeepLast:    0,
			},
			wantPath: "/mnt/x", wantCredID: 0, wantEnabled: false, wantKeepLast: 0,
			wantSchedMin: "0", wantSchedHour: "0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m CloudBackupResourceModel
			// Prime AttributesJSON to empty string so the filter path is
			// exercised consistently across test cases.
			m.AttributesJSON = types.StringValue("")
			r.mapResponseToModel(ctx, tc.cb, &m)
			if m.Path.ValueString() != tc.wantPath {
				t.Errorf("Path = %q, want %q", m.Path.ValueString(), tc.wantPath)
			}
			if m.Credentials.ValueInt64() != tc.wantCredID {
				t.Errorf("Credentials = %d, want %d", m.Credentials.ValueInt64(), tc.wantCredID)
			}
			if m.Enabled.ValueBool() != tc.wantEnabled {
				t.Errorf("Enabled = %v, want %v", m.Enabled.ValueBool(), tc.wantEnabled)
			}
			if m.KeepLast.ValueInt64() != tc.wantKeepLast {
				t.Errorf("KeepLast = %d, want %d", m.KeepLast.ValueInt64(), tc.wantKeepLast)
			}
			if m.ScheduleMinute.ValueString() != tc.wantSchedMin {
				t.Errorf("ScheduleMinute = %q, want %q", m.ScheduleMinute.ValueString(), tc.wantSchedMin)
			}
			if m.ScheduleHour.ValueString() != tc.wantSchedHour {
				t.Errorf("ScheduleHour = %q, want %q", m.ScheduleHour.ValueString(), tc.wantSchedHour)
			}
			// ID must always be populated
			if m.ID.ValueString() == "" {
				t.Error("ID should be set after mapResponseToModel")
			}
		})
	}
}

func TestCloudBackupResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewCloudBackupResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "path", "credentials", "attributes_json", "enabled", "keep_last", "schedule_minute", "schedule_hour"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["path"].IsRequired() {
		t.Error("path should be required")
	}
	if !attrs["credentials"].IsRequired() {
		t.Error("credentials should be required")
	}
}
