package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestSMBShareResource_MapResponseToModel_Cases(t *testing.T) {
	r := &SMBShareResource{}
	cases := []struct {
		name  string
		share *client.SMBShare
		want  SMBShareResourceModel
	}{
		{
			name: "minimal share",
			share: &client.SMBShare{
				ID: 1, Path: "/mnt/tank/smb", Name: "smb1",
				Browsable: true, Enabled: true,
			},
		},
		{
			name: "readonly share with comment",
			share: &client.SMBShare{
				ID: 3, Path: "/mnt/tank/ro", Name: "ro", Comment: "read only",
				ReadOnly: true, Browsable: true, ABE: true, Enabled: true,
				Purpose: "NO_PRESET",
			},
		},
		{
			name: "hidden share",
			share: &client.SMBShare{
				ID: 7, Path: "/mnt/tank/hid", Name: "hid$",
				Browsable: false, Enabled: true,
			},
		},
		{
			name:  "disabled share",
			share: &client.SMBShare{ID: 9, Path: "/mnt/x", Name: "x"},
		},
		{
			name: "share with recycle purpose",
			share: &client.SMBShare{
				ID: 11, Path: "/mnt/tank/recycle", Name: "recycle",
				Browsable: true, Enabled: true, Purpose: "NO_PRESET",
				Comment: "recycle-enabled",
			},
		},
		{
			name: "share with access based enumeration",
			share: &client.SMBShare{
				ID: 12, Path: "/mnt/tank/abe", Name: "abeshare",
				Browsable: true, ABE: true, Enabled: true,
			},
		},
		{
			name: "share with multiple booleans",
			share: &client.SMBShare{
				ID: 13, Path: "/mnt/tank/multi", Name: "multi",
				Browsable: true, ReadOnly: false, ABE: true, Enabled: true,
			},
		},
		{
			name: "share with special characters in name",
			share: &client.SMBShare{
				ID: 14, Path: "/mnt/tank/special", Name: "with space",
				Browsable: true, Enabled: true, Comment: "special chars $",
			},
		},
	}
	for _, tc := range cases {
		_ = tc.want
		t.Run(tc.name, func(t *testing.T) {
			var m SMBShareResourceModel
			r.mapResponseToModel(tc.share, &m)
			if m.Path.ValueString() != tc.share.Path {
				t.Errorf("Path = %q, want %q", m.Path.ValueString(), tc.share.Path)
			}
			if m.Name.ValueString() != tc.share.Name {
				t.Errorf("Name = %q, want %q", m.Name.ValueString(), tc.share.Name)
			}
			if m.Comment.ValueString() != tc.share.Comment {
				t.Errorf("Comment = %q, want %q", m.Comment.ValueString(), tc.share.Comment)
			}
			if m.Browsable.ValueBool() != tc.share.Browsable {
				t.Errorf("Browsable mismatch")
			}
			if m.ReadOnly.ValueBool() != tc.share.ReadOnly {
				t.Errorf("ReadOnly mismatch")
			}
			if m.ABE.ValueBool() != tc.share.ABE {
				t.Errorf("ABE mismatch")
			}
			if m.Enabled.ValueBool() != tc.share.Enabled {
				t.Errorf("Enabled mismatch")
			}
			if m.Purpose.ValueString() != tc.share.Purpose {
				t.Errorf("Purpose mismatch")
			}
		})
	}
}

func TestSMBShareResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewSMBShareResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "path", "name", "comment", "browsable", "readonly", "enabled"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["path"].IsRequired() {
		t.Error("path should be required")
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}
