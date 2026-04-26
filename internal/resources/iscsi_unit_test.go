package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --- ISCSI Portal ---

func TestISCSIPortalResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ISCSIPortalResource{}
	ctx := context.Background()
	cases := []struct {
		name       string
		portal     *client.ISCSIPortal
		wantID     string
		wantCmt    string
		wantTag    int64
		wantListen int
	}{
		{
			name:   "minimal portal",
			portal: &client.ISCSIPortal{ID: 1, Tag: 1, Comment: "default"},
			wantID: "1", wantCmt: "default", wantTag: 1, wantListen: 0,
		},
		{
			name: "portal with one listen",
			portal: &client.ISCSIPortal{
				ID: 2, Tag: 3, Comment: "one",
				Listen: []client.ISCSIPortalListen{{IP: "0.0.0.0", Port: 3260}},
			},
			wantID: "2", wantCmt: "one", wantTag: 3, wantListen: 1,
		},
		{
			name: "portal with multiple listens",
			portal: &client.ISCSIPortal{
				ID: 5, Tag: 10,
				Listen: []client.ISCSIPortalListen{
					{IP: "10.0.0.1", Port: 3260},
					{IP: "10.0.0.2", Port: 3260},
				},
			},
			wantID: "5", wantTag: 10, wantListen: 2,
		},
		{
			name:   "empty portal",
			portal: &client.ISCSIPortal{ID: 0, Tag: 0},
			wantID: "0",
		},
		{
			name: "portal with ipv6 listen",
			portal: &client.ISCSIPortal{
				ID: 10, Tag: 100, Comment: "v6",
				Listen: []client.ISCSIPortalListen{{IP: "::", Port: 3260}},
			},
			wantID: "10", wantCmt: "v6", wantTag: 100, wantListen: 1,
		},
		{
			name: "portal mixed v4/v6",
			portal: &client.ISCSIPortal{
				ID: 11, Tag: 11,
				Listen: []client.ISCSIPortalListen{
					{IP: "0.0.0.0", Port: 3260},
					{IP: "::", Port: 3260},
				},
			},
			wantID: "11", wantTag: 11, wantListen: 2,
		},
		{
			name: "portal three listens",
			portal: &client.ISCSIPortal{
				ID: 12, Tag: 12,
				Listen: []client.ISCSIPortalListen{
					{IP: "10.0.0.1"}, {IP: "10.0.0.2"}, {IP: "10.0.0.3"},
				},
			},
			wantID: "12", wantTag: 12, wantListen: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ISCSIPortalResourceModel
			r.mapResponseToModel(ctx, tc.portal, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Comment.ValueString() != tc.wantCmt {
				t.Errorf("Comment = %q, want %q", m.Comment.ValueString(), tc.wantCmt)
			}
			if m.Tag.ValueInt64() != tc.wantTag {
				t.Errorf("Tag = %d, want %d", m.Tag.ValueInt64(), tc.wantTag)
			}
			if got := len(m.Listen.Elements()); got != tc.wantListen {
				t.Errorf("Listen len = %d, want %d", got, tc.wantListen)
			}
		})
	}
}

func TestISCSIPortalResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewISCSIPortalResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "comment", "listen", "tag"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
}

// --- ISCSI Target ---

func TestISCSITargetResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ISCSITargetResource{}
	ctx := context.Background()
	cases := []struct {
		name       string
		target     *client.ISCSITarget
		wantID     string
		wantName   string
		wantAlias  string
		wantMode   string
		wantGroups int
	}{
		{
			name:     "minimal target",
			target:   &client.ISCSITarget{ID: 1, Name: "tgt1", Mode: "ISCSI"},
			wantID:   "1",
			wantName: "tgt1",
			wantMode: "ISCSI",
		},
		{
			name: "target with groups",
			target: &client.ISCSITarget{
				ID: 2, Name: "tgt2", Alias: "Alias", Mode: "ISCSI",
				Groups: []client.ISCSITargetGroup{
					{Portal: 1, Initiator: 1, AuthMethod: "NONE", Auth: 0},
					{Portal: 2, Initiator: 2, AuthMethod: "CHAP", Auth: 1},
				},
			},
			wantID: "2", wantName: "tgt2", wantAlias: "Alias", wantMode: "ISCSI", wantGroups: 2,
		},
		{
			name:   "target FC mode",
			target: &client.ISCSITarget{ID: 3, Name: "fctgt", Mode: "FC"},
			wantID: "3", wantName: "fctgt", wantMode: "FC",
		},
		{
			name:   "target with only alias",
			target: &client.ISCSITarget{ID: 4, Name: "tgt4", Alias: "friendly", Mode: "ISCSI"},
			wantID: "4", wantName: "tgt4", wantAlias: "friendly", wantMode: "ISCSI",
		},
		{
			name: "target with 3 groups",
			target: &client.ISCSITarget{
				ID: 5, Name: "tgt5", Mode: "ISCSI",
				Groups: []client.ISCSITargetGroup{
					{Portal: 1, Initiator: 1, AuthMethod: "NONE"},
					{Portal: 2, Initiator: 2, AuthMethod: "CHAP", Auth: 1},
					{Portal: 3, Initiator: 3, AuthMethod: "CHAP_MUTUAL", Auth: 2},
				},
			},
			wantID: "5", wantName: "tgt5", wantMode: "ISCSI", wantGroups: 3,
		},
		{
			name:   "target FC mode with alias",
			target: &client.ISCSITarget{ID: 6, Name: "fc1", Alias: "fiberchannel", Mode: "FC"},
			wantID: "6", wantName: "fc1", wantAlias: "fiberchannel", wantMode: "FC",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ISCSITargetResourceModel
			r.mapResponseToModel(ctx, tc.target, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.Name.ValueString() != tc.wantName {
				t.Errorf("Name = %q", m.Name.ValueString())
			}
			if m.Alias.ValueString() != tc.wantAlias {
				t.Errorf("Alias = %q", m.Alias.ValueString())
			}
			if m.Mode.ValueString() != tc.wantMode {
				t.Errorf("Mode = %q", m.Mode.ValueString())
			}
			if m.Groups.IsNull() {
				if tc.wantGroups != 0 {
					t.Errorf("Groups nil but want %d", tc.wantGroups)
				}
			} else if got := len(m.Groups.Elements()); got != tc.wantGroups {
				t.Errorf("Groups len = %d, want %d", got, tc.wantGroups)
			}
		})
	}
}

func TestISCSITargetResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewISCSITargetResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "alias", "mode", "groups"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- ISCSI Extent ---

func TestISCSIExtentResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ISCSIExtentResource{}
	cases := []struct {
		name     string
		extent   *client.ISCSIExtent
		wantID   string
		wantType string
		wantBS   int64
		wantRO   bool
	}{
		{
			name:     "file extent",
			extent:   &client.ISCSIExtent{ID: 1, Name: "e1", Type: "FILE", Path: "/mnt/tank/e1.img", Blocksize: 512},
			wantID:   "1",
			wantType: "FILE",
			wantBS:   512,
		},
		{
			name:     "disk extent",
			extent:   &client.ISCSIExtent{ID: 2, Name: "e2", Type: "DISK", Blocksize: 4096},
			wantID:   "2",
			wantType: "DISK",
			wantBS:   4096,
		},
		{
			name:     "readonly extent",
			extent:   &client.ISCSIExtent{ID: 3, Name: "e3", Type: "FILE", ReadOnly: true},
			wantID:   "3",
			wantType: "FILE",
			wantRO:   true,
		},
		{
			name:     "extent with comment",
			extent:   &client.ISCSIExtent{ID: 4, Name: "e4", Type: "DISK", Comment: "test", Enabled: true},
			wantID:   "4",
			wantType: "DISK",
		},
		{
			name:     "extent with xen flag",
			extent:   &client.ISCSIExtent{ID: 5, Name: "xen", Type: "DISK", Xen: true, Enabled: true, Blocksize: 512},
			wantID:   "5",
			wantType: "DISK",
			wantBS:   512,
		},
		{
			name:     "extent with insecure_tpc",
			extent:   &client.ISCSIExtent{ID: 6, Name: "tpc", Type: "FILE", InsecureTPC: true, Blocksize: 4096},
			wantID:   "6",
			wantType: "FILE",
			wantBS:   4096,
		},
		{
			name:     "extent with RPM",
			extent:   &client.ISCSIExtent{ID: 7, Name: "rpm", Type: "DISK", RPM: "SSD", Blocksize: 512},
			wantID:   "7",
			wantType: "DISK",
			wantBS:   512,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ISCSIExtentResourceModel
			r.mapResponseToModel(tc.extent, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.Name.ValueString() != tc.extent.Name {
				t.Errorf("Name = %q", m.Name.ValueString())
			}
			if m.Type.ValueString() != tc.wantType {
				t.Errorf("Type = %q", m.Type.ValueString())
			}
			if m.Blocksize.ValueInt64() != tc.wantBS {
				t.Errorf("Blocksize = %d, want %d", m.Blocksize.ValueInt64(), tc.wantBS)
			}
			if m.ReadOnly.ValueBool() != tc.wantRO {
				t.Errorf("ReadOnly = %v", m.ReadOnly.ValueBool())
			}
		})
	}
}

func TestISCSIExtentResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewISCSIExtentResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "type", "blocksize", "enabled"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

// --- ISCSI Auth ---

func TestISCSIAuthResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ISCSIAuthResource{}
	cases := []struct {
		name     string
		auth     *client.ISCSIAuth
		wantID   string
		wantTag  int64
		wantUser string
		wantDisc string
	}{
		{
			name:   "chap auth",
			auth:   &client.ISCSIAuth{ID: 1, Tag: 1, User: "chapuser", Secret: "[REDACTED]"},
			wantID: "1", wantTag: 1, wantUser: "chapuser", wantDisc: "NONE",
		},
		{
			name:   "mutual chap",
			auth:   &client.ISCSIAuth{ID: 2, Tag: 2, User: "u", Peeruser: "p", DiscoveryAuth: "CHAP_MUTUAL"},
			wantID: "2", wantTag: 2, wantUser: "u", wantDisc: "CHAP_MUTUAL",
		},
		{
			name:   "auth empty discovery",
			auth:   &client.ISCSIAuth{ID: 3, Tag: 3, User: "u3"},
			wantID: "3", wantTag: 3, wantUser: "u3", wantDisc: "NONE",
		},
		{
			name:   "auth high tag",
			auth:   &client.ISCSIAuth{ID: 4, Tag: 100, User: "u4", DiscoveryAuth: "CHAP"},
			wantID: "4", wantTag: 100, wantUser: "u4", wantDisc: "CHAP",
		},
		{
			name:   "auth with long username",
			auth:   &client.ISCSIAuth{ID: 5, Tag: 5, User: "verylongchapuser", DiscoveryAuth: "NONE"},
			wantID: "5", wantTag: 5, wantUser: "verylongchapuser", wantDisc: "NONE",
		},
		{
			name:   "auth tag boundary max",
			auth:   &client.ISCSIAuth{ID: 6, Tag: 65535, User: "mx", DiscoveryAuth: "NONE"},
			wantID: "6", wantTag: 65535, wantUser: "mx", wantDisc: "NONE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ISCSIAuthResourceModel
			r.mapResponseToModel(tc.auth, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.Tag.ValueInt64() != tc.wantTag {
				t.Errorf("Tag = %d", m.Tag.ValueInt64())
			}
			if m.User.ValueString() != tc.wantUser {
				t.Errorf("User = %q", m.User.ValueString())
			}
			if m.DiscoveryAuth.ValueString() != tc.wantDisc {
				t.Errorf("DiscoveryAuth = %q, want %q", m.DiscoveryAuth.ValueString(), tc.wantDisc)
			}
		})
	}
}

// --- ISCSI Initiator ---

func TestISCSIInitiatorResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ISCSIInitiatorResource{}
	ctx := context.Background()
	cases := []struct {
		name           string
		init           *client.ISCSIInitiator
		wantID         string
		wantComment    string
		wantInitiators int
	}{
		{
			name:        "empty allow all",
			init:        &client.ISCSIInitiator{ID: 1, Comment: "allow all"},
			wantID:      "1",
			wantComment: "allow all",
		},
		{
			name:           "single initiator",
			init:           &client.ISCSIInitiator{ID: 2, Comment: "one", Initiators: []string{"iqn.1991-05.com.microsoft:host1"}},
			wantID:         "2",
			wantComment:    "one",
			wantInitiators: 1,
		},
		{
			name: "multiple initiators",
			init: &client.ISCSIInitiator{
				ID:      3,
				Comment: "multi",
				Initiators: []string{
					"iqn.1991-05.com.microsoft:host1",
					"iqn.1991-05.com.microsoft:host2",
					"iqn.1991-05.com.microsoft:host3",
				},
			},
			wantID: "3", wantComment: "multi", wantInitiators: 3,
		},
		{
			name:        "zero ID",
			init:        &client.ISCSIInitiator{ID: 0},
			wantID:      "0",
			wantComment: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ISCSIInitiatorResourceModel
			r.mapResponseToModel(ctx, tc.init, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Comment.ValueString() != tc.wantComment {
				t.Errorf("Comment = %q, want %q", m.Comment.ValueString(), tc.wantComment)
			}
			if tc.wantInitiators > 0 {
				if got := len(m.Initiators.Elements()); got != tc.wantInitiators {
					t.Errorf("Initiators len = %d, want %d", got, tc.wantInitiators)
				}
			}
		})
	}
}

func TestISCSIInitiatorResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewISCSIInitiatorResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "initiators", "comment"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
}

func TestISCSIAuthResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewISCSIAuthResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "tag", "user", "secret"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
}
