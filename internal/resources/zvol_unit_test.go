package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestZvolResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ZvolResource{}
	cases := []struct {
		name        string
		ds          *client.DatasetResponse
		wantID      string
		wantPool    string
		wantDSName  string
		wantComp    string
		wantComment string
	}{
		{
			name: "basic zvol no properties",
			ds: &client.DatasetResponse{
				ID:   "tank/myvol",
				Type: "VOLUME",
			},
			wantID:     "tank/myvol",
			wantPool:   "tank",
			wantDSName: "myvol",
		},
		{
			name: "zvol with compression and comment",
			ds: &client.DatasetResponse{
				ID:          "tank/vol1",
				Type:        "VOLUME",
				Compression: pv("ZSTD"),
				Comments:    pv("db vol"),
			},
			wantID:      "tank/vol1",
			wantPool:    "tank",
			wantDSName:  "vol1",
			wantComp:    "ZSTD",
			wantComment: "db vol",
		},
		{
			name: "zvol with nested path",
			ds: &client.DatasetResponse{
				ID:   "pool2/sub/vol",
				Type: "VOLUME",
			},
			wantID:     "pool2/sub/vol",
			wantPool:   "pool2",
			wantDSName: "sub/vol",
		},
		{
			name: "zvol with nil comments defaults to empty",
			ds: &client.DatasetResponse{
				ID:   "p/v",
				Type: "VOLUME",
			},
			wantID:      "p/v",
			wantPool:    "p",
			wantDSName:  "v",
			wantComment: "",
		},
		{
			name: "zvol with volsize and volblocksize",
			ds: &client.DatasetResponse{
				ID:           "tank/bigvol",
				Type:         "VOLUME",
				Volsize:      &client.PropertyRawVal{Rawvalue: "10737418240"},
				Volblocksize: &client.PropertyValue{Value: "16K"},
				Compression:  pv("LZ4"),
			},
			wantID:     "tank/bigvol",
			wantPool:   "tank",
			wantDSName: "bigvol",
			wantComp:   "LZ4",
		},
		{
			name: "zvol small block",
			ds: &client.DatasetResponse{
				ID:           "fast/v1",
				Type:         "VOLUME",
				Volsize:      &client.PropertyRawVal{Rawvalue: "1048576"},
				Volblocksize: &client.PropertyValue{Value: "4K"},
			},
			wantID:     "fast/v1",
			wantPool:   "fast",
			wantDSName: "v1",
		},
		{
			name: "zvol with full property set",
			ds: &client.DatasetResponse{
				ID:           "tank/prop",
				Type:         "VOLUME",
				Volsize:      &client.PropertyRawVal{Rawvalue: "134217728"},
				Volblocksize: &client.PropertyValue{Value: "128K"},
				Compression:  pv("ZSTD"),
				Sync:         pv("DISABLED"),
				Comments:     pv("test zvol"),
			},
			wantID:      "tank/prop",
			wantPool:    "tank",
			wantDSName:  "prop",
			wantComp:    "ZSTD",
			wantComment: "test zvol",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ZvolResourceModel
			r.mapResponseToModel(tc.ds, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Pool.ValueString() != tc.wantPool {
				t.Errorf("Pool = %q, want %q", m.Pool.ValueString(), tc.wantPool)
			}
			if m.Name.ValueString() != tc.wantDSName {
				t.Errorf("Name = %q, want %q", m.Name.ValueString(), tc.wantDSName)
			}
			if tc.wantComp != "" && m.Compression.ValueString() != tc.wantComp {
				t.Errorf("Compression = %q, want %q", m.Compression.ValueString(), tc.wantComp)
			}
			if m.Comments.ValueString() != tc.wantComment {
				t.Errorf("Comments = %q, want %q", m.Comments.ValueString(), tc.wantComment)
			}
		})
	}
}

func TestZvolResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewZvolResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "pool", "volsize", "volblocksize", "compression", "comments"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["pool"].IsRequired() {
		t.Error("pool should be required")
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
	if !attrs["id"].IsComputed() {
		t.Error("id should be computed")
	}
}
