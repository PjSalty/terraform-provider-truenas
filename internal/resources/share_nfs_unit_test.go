package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNFSShareResource_MapResponseToModel_Cases(t *testing.T) {
	r := &NFSShareResource{}
	ctx := context.Background()
	cases := []struct {
		name        string
		share       *client.NFSShare
		wantID      string
		wantPath    string
		wantComment string
		wantRO      bool
		wantEnabled bool
		wantHostsN  int
		wantNetsN   int
		wantSecN    int
	}{
		{
			name:        "minimal share",
			share:       &client.NFSShare{ID: 1, Path: "/mnt/tank/nfs", Enabled: true},
			wantID:      "1",
			wantPath:    "/mnt/tank/nfs",
			wantEnabled: true,
		},
		{
			name: "share with hosts and networks",
			share: &client.NFSShare{
				ID:       5,
				Path:     "/mnt/tank/share",
				Comment:  "team share",
				Hosts:    []string{"host1", "host2"},
				Networks: []string{"10.0.0.0/24"},
				Security: []string{"SYS"},
				Enabled:  true,
			},
			wantID:      "5",
			wantPath:    "/mnt/tank/share",
			wantComment: "team share",
			wantEnabled: true,
			wantHostsN:  2,
			wantNetsN:   1,
			wantSecN:    1,
		},
		{
			name: "readonly share with mapping",
			share: &client.NFSShare{
				ID:          7,
				Path:        "/mnt/tank/ro",
				ReadOnly:    true,
				MaprootUser: "root",
				MapallUser:  "nobody",
			},
			wantID:   "7",
			wantPath: "/mnt/tank/ro",
			wantRO:   true,
		},
		{
			name:   "disabled share",
			share:  &client.NFSShare{ID: 9, Path: "/mnt/x"},
			wantID: "9", wantPath: "/mnt/x",
		},
		{
			name: "share with all security mechanisms",
			share: &client.NFSShare{
				ID:       10,
				Path:     "/mnt/tank/secure",
				Security: []string{"SYS", "KRB5", "KRB5I", "KRB5P"},
				Enabled:  true,
			},
			wantID: "10", wantPath: "/mnt/tank/secure", wantEnabled: true, wantSecN: 4,
		},
		{
			name: "share with CIDR networks",
			share: &client.NFSShare{
				ID:       11,
				Path:     "/mnt/tank/cidr",
				Networks: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
				Enabled:  true,
			},
			wantID: "11", wantPath: "/mnt/tank/cidr", wantEnabled: true, wantNetsN: 3,
		},
		{
			name: "share with mapall group",
			share: &client.NFSShare{
				ID:           12,
				Path:         "/mnt/tank/shared",
				MapallUser:   "nobody",
				MapallGroup:  "nogroup",
				MaprootUser:  "root",
				MaprootGroup: "wheel",
			},
			wantID: "12", wantPath: "/mnt/tank/shared",
		},
		{
			name: "share with many hosts",
			share: &client.NFSShare{
				ID:      13,
				Path:    "/mnt/tank/clients",
				Hosts:   []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8"},
				Enabled: true,
			},
			wantID: "13", wantPath: "/mnt/tank/clients", wantEnabled: true, wantHostsN: 8,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m NFSShareResourceModel
			r.mapResponseToModel(ctx, tc.share, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Path.ValueString() != tc.wantPath {
				t.Errorf("Path = %q, want %q", m.Path.ValueString(), tc.wantPath)
			}
			if m.Comment.ValueString() != tc.wantComment {
				t.Errorf("Comment = %q, want %q", m.Comment.ValueString(), tc.wantComment)
			}
			if m.ReadOnly.ValueBool() != tc.wantRO {
				t.Errorf("ReadOnly = %v, want %v", m.ReadOnly.ValueBool(), tc.wantRO)
			}
			if m.Enabled.ValueBool() != tc.wantEnabled {
				t.Errorf("Enabled = %v, want %v", m.Enabled.ValueBool(), tc.wantEnabled)
			}
			if got := len(m.Hosts.Elements()); got != tc.wantHostsN {
				t.Errorf("Hosts len = %d, want %d", got, tc.wantHostsN)
			}
			if got := len(m.Networks.Elements()); got != tc.wantNetsN {
				t.Errorf("Networks len = %d, want %d", got, tc.wantNetsN)
			}
			if got := len(m.Security.Elements()); got != tc.wantSecN {
				t.Errorf("Security len = %d, want %d", got, tc.wantSecN)
			}
		})
	}
}

func TestNFSShareResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewNFSShareResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "path", "comment", "hosts", "networks", "readonly", "enabled"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["path"].IsRequired() {
		t.Error("path should be required")
	}
	if !attrs["id"].IsComputed() {
		t.Error("id should be computed")
	}
}
