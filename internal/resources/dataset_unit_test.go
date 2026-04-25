package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func pv(v string) *client.PropertyValue { return &client.PropertyValue{Value: v} }
func prv(raw string) *client.PropertyRawVal {
	return &client.PropertyRawVal{Rawvalue: raw, Value: raw}
}

func TestDatasetResource_MapResponseToModel_Cases(t *testing.T) {
	r := &DatasetResource{}
	cases := []struct {
		name        string
		ds          *client.DatasetResponse
		wantID      string
		wantPool    string
		wantName    string
		wantParent  string
		wantType    string
		wantComp    string
		wantQuota   int64
		wantComment string
	}{
		{
			name: "root dataset no compression",
			ds: &client.DatasetResponse{
				ID:         "tank/data",
				MountPoint: "/mnt/tank/data",
				Type:       "FILESYSTEM",
			},
			wantID:   "tank/data",
			wantPool: "tank",
			wantName: "data",
			wantType: "FILESYSTEM",
		},
		{
			name: "nested dataset with compression and quota",
			ds: &client.DatasetResponse{
				ID:          "tank/apps/postgres",
				MountPoint:  "/mnt/tank/apps/postgres",
				Type:        "FILESYSTEM",
				Compression: pv("LZ4"),
				Quota:       prv("1073741824"),
				Comments:    pv("postgres data"),
			},
			wantID:      "tank/apps/postgres",
			wantPool:    "tank",
			wantName:    "postgres",
			wantParent:  "apps",
			wantType:    "FILESYSTEM",
			wantComp:    "LZ4",
			wantQuota:   1073741824,
			wantComment: "postgres data",
		},
		{
			name: "volume type dataset",
			ds: &client.DatasetResponse{
				ID:         "pool1/vol1",
				MountPoint: "",
				Type:       "VOLUME",
			},
			wantID:   "pool1/vol1",
			wantPool: "pool1",
			wantName: "vol1",
			wantType: "VOLUME",
		},
		{
			name: "dataset with nil comments",
			ds: &client.DatasetResponse{
				ID:         "tank/foo",
				MountPoint: "/mnt/tank/foo",
				Type:       "FILESYSTEM",
			},
			wantID:   "tank/foo",
			wantPool: "tank",
			wantName: "foo",
			wantType: "FILESYSTEM",
		},
		{
			name: "dataset with zero quota raw",
			ds: &client.DatasetResponse{
				ID:       "tank/nolimit",
				Type:     "FILESYSTEM",
				Quota:    prv("0"),
				Refquota: prv("0"),
			},
			wantID:   "tank/nolimit",
			wantPool: "tank",
			wantName: "nolimit",
			wantType: "FILESYSTEM",
		},
		{
			name: "dataset with atime and dedup",
			ds: &client.DatasetResponse{
				ID:            "tank/sec",
				Type:          "FILESYSTEM",
				Atime:         pv("OFF"),
				Deduplication: pv("ON"),
			},
			wantID:   "tank/sec",
			wantPool: "tank",
			wantName: "sec",
			wantType: "FILESYSTEM",
		},
		{
			name: "dataset deep nested",
			ds: &client.DatasetResponse{
				ID:         "big/a/b/c/d/e/final",
				Type:       "FILESYSTEM",
				MountPoint: "/mnt/big/a/b/c/d/e/final",
			},
			wantID:     "big/a/b/c/d/e/final",
			wantPool:   "big",
			wantName:   "final",
			wantParent: "a/b/c/d/e",
			wantType:   "FILESYSTEM",
		},
		{
			name: "dataset with sync and readonly",
			ds: &client.DatasetResponse{
				ID:       "tank/sync",
				Type:     "FILESYSTEM",
				Sync:     pv("ALWAYS"),
				Readonly: pv("ON"),
			},
			wantID: "tank/sync", wantPool: "tank", wantName: "sync", wantType: "FILESYSTEM",
		},
		{
			name: "dataset with recordsize",
			ds: &client.DatasetResponse{
				ID:         "tank/rs",
				Type:       "FILESYSTEM",
				RecordSize: pv("1M"),
			},
			wantID: "tank/rs", wantPool: "tank", wantName: "rs", wantType: "FILESYSTEM",
		},
		{
			name: "dataset with copies",
			ds: &client.DatasetResponse{
				ID:     "tank/cp",
				Type:   "FILESYSTEM",
				Copies: pv("3"),
			},
			wantID: "tank/cp", wantPool: "tank", wantName: "cp", wantType: "FILESYSTEM",
		},
		{
			name: "dataset with snapdir",
			ds: &client.DatasetResponse{
				ID:      "tank/snap",
				Type:    "FILESYSTEM",
				Snapdir: pv("VISIBLE"),
			},
			wantID: "tank/snap", wantPool: "tank", wantName: "snap", wantType: "FILESYSTEM",
		},
		{
			name: "dataset with user_properties comment (25.10 shape)",
			ds: &client.DatasetResponse{
				ID:   "tank/np",
				Type: "FILESYSTEM",
				UserProperties: map[string]*client.PropertyValue{
					"comments": {Value: "new-shape"},
				},
			},
			wantID: "tank/np", wantPool: "tank", wantName: "np", wantType: "FILESYSTEM",
		},
		{
			name: "dataset with ShareType from API",
			ds: &client.DatasetResponse{
				ID:        "tank/smb",
				Type:      "FILESYSTEM",
				ShareType: pv("SMB"),
			},
			wantID: "tank/smb", wantPool: "tank", wantName: "smb", wantType: "FILESYSTEM",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m DatasetResourceModel
			r.mapResponseToModel(tc.ds, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Pool.ValueString() != tc.wantPool {
				t.Errorf("Pool = %q, want %q", m.Pool.ValueString(), tc.wantPool)
			}
			if m.Name.ValueString() != tc.wantName {
				t.Errorf("Name = %q, want %q", m.Name.ValueString(), tc.wantName)
			}
			if tc.wantParent != "" && m.ParentDataset.ValueString() != tc.wantParent {
				t.Errorf("Parent = %q, want %q", m.ParentDataset.ValueString(), tc.wantParent)
			}
			if m.Type.ValueString() != tc.wantType {
				t.Errorf("Type = %q, want %q", m.Type.ValueString(), tc.wantType)
			}
			if tc.wantComp != "" && m.Compression.ValueString() != tc.wantComp {
				t.Errorf("Compression = %q, want %q", m.Compression.ValueString(), tc.wantComp)
			}
			if tc.wantQuota != 0 && m.Quota.ValueInt64() != tc.wantQuota {
				t.Errorf("Quota = %d, want %d", m.Quota.ValueInt64(), tc.wantQuota)
			}
			if tc.wantComment != "" && m.Comments.ValueString() != tc.wantComment {
				t.Errorf("Comments = %q, want %q", m.Comments.ValueString(), tc.wantComment)
			}
		})
	}
}

func TestDatasetResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewDatasetResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "pool", "type", "compression", "atime", "deduplication", "quota", "mount_point"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
	if !attrs["pool"].IsRequired() {
		t.Error("pool should be required")
	}
	if !attrs["id"].IsComputed() {
		t.Error("id should be computed")
	}
}
