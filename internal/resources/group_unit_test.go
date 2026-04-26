package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestGroupResource_MapResponseToModel_Cases(t *testing.T) {
	r := &GroupResource{}
	cases := []struct {
		name     string
		group    *client.Group
		wantID   string
		wantName string
		wantGID  int64
		wantSMB  bool
		wantCmds int
	}{
		{
			name:     "minimal group",
			group:    &client.Group{ID: 1, GID: 1000, Name: "users"},
			wantID:   "1",
			wantName: "users",
			wantGID:  1000,
		},
		{
			name:     "SMB group with sudo commands",
			group:    &client.Group{ID: 5, GID: 2000, Name: "smbusers", SMB: true, SudoCommands: []string{"/bin/ls", "/usr/bin/git"}},
			wantID:   "5",
			wantName: "smbusers",
			wantGID:  2000,
			wantSMB:  true,
			wantCmds: 2,
		},
		{
			name:     "group with zero GID",
			group:    &client.Group{ID: 0, GID: 0, Name: "wheel"},
			wantID:   "0",
			wantName: "wheel",
			wantGID:  0,
		},
		{
			name:     "builtin group",
			group:    &client.Group{ID: 42, GID: 42, Name: "builtin", Builtin: true},
			wantID:   "42",
			wantName: "builtin",
			wantGID:  42,
		},
		{
			name:     "group with high GID",
			group:    &client.Group{ID: 999, GID: 65533, Name: "nobody"},
			wantID:   "999",
			wantName: "nobody",
			wantGID:  65533,
		},
		{
			name:     "SMB builtin admins",
			group:    &client.Group{ID: 544, GID: 544, Name: "administrators", SMB: true, Builtin: true},
			wantID:   "544",
			wantName: "administrators",
			wantGID:  544,
			wantSMB:  true,
		},
		{
			name: "group with five sudo commands",
			group: &client.Group{
				ID: 300, GID: 3000, Name: "ops",
				SudoCommands: []string{"/bin/ls", "/bin/ps", "/bin/df", "/bin/du", "/bin/free"},
			},
			wantID:   "300",
			wantName: "ops",
			wantGID:  3000,
			wantCmds: 5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m GroupResourceModel
			r.mapResponseToModel(tc.group, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Name.ValueString() != tc.wantName {
				t.Errorf("Name = %q, want %q", m.Name.ValueString(), tc.wantName)
			}
			if m.GID.ValueInt64() != tc.wantGID {
				t.Errorf("GID = %d, want %d", m.GID.ValueInt64(), tc.wantGID)
			}
			if m.SMB.ValueBool() != tc.wantSMB {
				t.Errorf("SMB = %v, want %v", m.SMB.ValueBool(), tc.wantSMB)
			}
			if got := len(m.SudoCommands.Elements()); got != tc.wantCmds {
				t.Errorf("SudoCommands length = %d, want %d", got, tc.wantCmds)
			}
		})
	}
}

func TestGroupResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewGroupResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "gid", "smb", "sudo_commands"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
	if !attrs["id"].IsComputed() {
		t.Error("id should be computed")
	}
}
