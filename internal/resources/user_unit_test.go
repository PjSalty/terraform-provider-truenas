package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func strPtr(s string) *string { return &s }

func TestUserResource_MapResponseToModel_Cases(t *testing.T) {
	r := &UserResource{}
	ctx := context.Background()

	cases := []struct {
		name         string
		user         *client.User
		wantID       string
		wantUsername string
		wantEmail    string
		wantLocked   bool
		wantGroupID  int64
		wantSSH      string
		wantGroupsN  int
		wantCmdsN    int
	}{
		{
			name: "minimal user with nil email and sshpubkey",
			user: &client.User{
				ID:       1,
				UID:      1000,
				Username: "alice",
				FullName: "Alice",
				Home:     "/home/alice",
				Shell:    "/bin/bash",
				Group:    client.UserGroup{ID: 42},
			},
			wantID:       "1",
			wantUsername: "alice",
			wantEmail:    "",
			wantLocked:   false,
			wantGroupID:  42,
			wantSSH:      "",
			wantGroupsN:  0,
			wantCmdsN:    0,
		},
		{
			name: "locked user with email and groups",
			user: &client.User{
				ID:           7,
				UID:          1007,
				Username:     "bob",
				FullName:     "Bob",
				Email:        strPtr("bob@example.com"),
				Home:         "/home/bob",
				Shell:        "/usr/sbin/nologin",
				Locked:       true,
				SMB:          true,
				Group:        client.UserGroup{ID: 100},
				Groups:       []int{10, 20, 30},
				SudoCommands: []string{"/bin/ls", "/bin/cat"},
			},
			wantID:       "7",
			wantUsername: "bob",
			wantEmail:    "bob@example.com",
			wantLocked:   true,
			wantGroupID:  100,
			wantSSH:      "",
			wantGroupsN:  3,
			wantCmdsN:    2,
		},
		{
			name: "user with ssh pubkey",
			user: &client.User{
				ID:        11,
				UID:       2000,
				Username:  "carol",
				FullName:  "Carol",
				SSHPubKey: strPtr("ssh-rsa AAAA..."),
				Group:     client.UserGroup{ID: 1},
			},
			wantID:       "11",
			wantUsername: "carol",
			wantEmail:    "",
			wantGroupID:  1,
			wantSSH:      "ssh-rsa AAAA...",
			wantGroupsN:  0,
			wantCmdsN:    0,
		},
		{
			name: "zero id user",
			user: &client.User{
				ID:       0,
				UID:      0,
				Username: "root",
				FullName: "root",
				Group:    client.UserGroup{ID: 0},
			},
			wantID:       "0",
			wantUsername: "root",
			wantEmail:    "",
			wantGroupID:  0,
			wantGroupsN:  0,
			wantCmdsN:    0,
		},
		{
			name: "user with many secondary groups",
			user: &client.User{
				ID:       100,
				UID:      3000,
				Username: "poweruser",
				FullName: "Power User",
				Group:    client.UserGroup{ID: 100},
				Groups:   []int{10, 20, 30, 40, 50, 60},
			},
			wantID:       "100",
			wantUsername: "poweruser",
			wantEmail:    "",
			wantGroupID:  100,
			wantGroupsN:  6,
		},
		{
			name: "user with empty email pointer",
			user: &client.User{
				ID:       50,
				UID:      2050,
				Username: "nomail",
				FullName: "No Mail",
				Email:    strPtr(""),
				Group:    client.UserGroup{ID: 100},
			},
			wantID:       "50",
			wantUsername: "nomail",
			wantEmail:    "",
			wantGroupID:  100,
		},
		{
			name: "user with many sudo commands",
			user: &client.User{
				ID:           77,
				UID:          2077,
				Username:     "sudoer",
				FullName:     "Sudoer",
				Group:        client.UserGroup{ID: 0},
				SudoCommands: []string{"/bin/ls", "/bin/cat", "/bin/grep", "/bin/find", "/bin/tail"},
			},
			wantID:       "77",
			wantUsername: "sudoer",
			wantGroupID:  0,
			wantCmdsN:    5,
		},
		{
			name: "SMB user with bash shell",
			user: &client.User{
				ID:       33,
				UID:      2033,
				Username: "smbuser",
				FullName: "SMB User",
				SMB:      true,
				Home:     "/home/smbuser",
				Shell:    "/bin/bash",
				Group:    client.UserGroup{ID: 200},
			},
			wantID:       "33",
			wantUsername: "smbuser",
			wantGroupID:  200,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m UserResourceModel
			r.mapResponseToModel(ctx, tc.user, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Username.ValueString() != tc.wantUsername {
				t.Errorf("Username = %q, want %q", m.Username.ValueString(), tc.wantUsername)
			}
			if m.Email.ValueString() != tc.wantEmail {
				t.Errorf("Email = %q, want %q", m.Email.ValueString(), tc.wantEmail)
			}
			if m.Locked.ValueBool() != tc.wantLocked {
				t.Errorf("Locked = %v, want %v", m.Locked.ValueBool(), tc.wantLocked)
			}
			if m.Group.ValueInt64() != tc.wantGroupID {
				t.Errorf("Group = %d, want %d", m.Group.ValueInt64(), tc.wantGroupID)
			}
			if m.SSHPubKey.ValueString() != tc.wantSSH {
				t.Errorf("SSHPubKey = %q, want %q", m.SSHPubKey.ValueString(), tc.wantSSH)
			}
			if got := len(m.Groups.Elements()); got != tc.wantGroupsN {
				t.Errorf("Groups length = %d, want %d", got, tc.wantGroupsN)
			}
			if got := len(m.SudoCommands.Elements()); got != tc.wantCmdsN {
				t.Errorf("SudoCommands length = %d, want %d", got, tc.wantCmdsN)
			}
		})
	}
}

func TestUserResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewUserResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	required := []string{"id", "uid", "username", "full_name", "email", "password", "group", "groups", "home", "shell", "locked", "smb"}
	for _, k := range required {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if u := attrs["username"]; !u.IsRequired() {
		t.Error("username should be required")
	}
	if id := attrs["id"]; !id.IsComputed() {
		t.Error("id should be computed")
	}
	if pw := attrs["password"]; !pw.IsRequired() {
		t.Error("password should be required")
	}
}
