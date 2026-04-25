package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --- NVMetHost ---

func TestNVMetHostResource_MapResponseToModel_Cases(t *testing.T) {
	r := &NVMetHostResource{}
	cases := []struct {
		name      string
		host      *client.NVMetHost
		wantID    string
		wantNQN   string
		wantDhKey bool
		wantHash  string
	}{
		{
			name:    "minimal host",
			host:    &client.NVMetHost{ID: 1, Hostnqn: "nqn.2014-08.org.nvmexpress:uuid:abc"},
			wantID:  "1",
			wantNQN: "nqn.2014-08.org.nvmexpress:uuid:abc",
		},
		{
			name: "host with dhchap key",
			host: &client.NVMetHost{
				ID:        2,
				Hostnqn:   "nqn.2014-08.org.nvmexpress:uuid:def",
				DhchapKey: strPtr("DHHC-1:00:secret-value-here"),
			},
			wantID: "2", wantNQN: "nqn.2014-08.org.nvmexpress:uuid:def", wantDhKey: true,
		},
		{
			name: "host with full dhchap config",
			host: &client.NVMetHost{
				ID:            3,
				Hostnqn:       "nqn.2014-08.org.nvmexpress:uuid:xyz",
				DhchapKey:     strPtr("k"),
				DhchapCtrlKey: strPtr("c"),
				DhchapDhgroup: strPtr("ffdhe2048"),
				DhchapHash:    strPtr("sha256"),
			},
			wantID: "3", wantNQN: "nqn.2014-08.org.nvmexpress:uuid:xyz", wantDhKey: true, wantHash: "sha256",
		},
		{
			name:    "host nil pointers stay null",
			host:    &client.NVMetHost{ID: 4, Hostnqn: "nqn.test:null"},
			wantID:  "4",
			wantNQN: "nqn.test:null",
		},
		{
			name: "host zero id",
			host: &client.NVMetHost{
				ID:      0,
				Hostnqn: "nqn.zero",
			},
			wantID:  "0",
			wantNQN: "nqn.zero",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m NVMetHostResourceModel
			r.mapResponseToModel(tc.host, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Hostnqn.ValueString() != tc.wantNQN {
				t.Errorf("Hostnqn = %q, want %q", m.Hostnqn.ValueString(), tc.wantNQN)
			}
			if tc.wantDhKey && m.DhchapKey.IsNull() {
				t.Error("DhchapKey should not be null")
			}
			if !tc.wantDhKey && !m.DhchapKey.IsNull() && m.DhchapKey.ValueString() != "" {
				t.Errorf("DhchapKey = %q, want null", m.DhchapKey.ValueString())
			}
			if tc.wantHash != "" && m.DhchapHash.ValueString() != tc.wantHash {
				t.Errorf("DhchapHash = %q, want %q", m.DhchapHash.ValueString(), tc.wantHash)
			}
		})
	}
}

func TestNVMetHostResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewNVMetHostResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "hostnqn", "dhchap_key", "dhchap_ctrl_key", "dhchap_dhgroup", "dhchap_hash"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["hostnqn"].IsRequired() {
		t.Error("hostnqn should be required")
	}
}

// --- NVMetSubsys ---

func TestNVMetSubsysResource_MapResponseToModel_Cases(t *testing.T) {
	r := &NVMetSubsysResource{}
	boolT := true
	boolF := false
	qid := 128
	oui := "00A098"
	cases := []struct {
		name      string
		subsys    *client.NVMetSubsys
		wantID    string
		wantName  string
		wantAAH   bool
		wantQid   int64
		wantOui   string
		wantAnaOK bool
	}{
		{
			name:     "minimal subsys",
			subsys:   &client.NVMetSubsys{ID: 1, Name: "target1", AllowAnyHost: true, Serial: "SN-1"},
			wantID:   "1",
			wantName: "target1",
			wantAAH:  true,
		},
		{
			name: "subsys with subnqn",
			subsys: &client.NVMetSubsys{
				ID:           2,
				Name:         "tgt2",
				Subnqn:       strPtr("nqn.2020-01.example:tgt2"),
				AllowAnyHost: false,
				Serial:       "SN-2",
			},
			wantID: "2", wantName: "tgt2", wantAAH: false,
		},
		{
			name: "subsys with all optional fields",
			subsys: &client.NVMetSubsys{
				ID:           3,
				Name:         "tgt3",
				Subnqn:       strPtr("nqn.tgt3"),
				AllowAnyHost: true,
				Serial:       "SN-3",
				PiEnable:     &boolT,
				QidMax:       &qid,
				IeeeOui:      &oui,
				Ana:          &boolF,
			},
			wantID: "3", wantName: "tgt3", wantAAH: true, wantQid: 128, wantOui: "00A098", wantAnaOK: true,
		},
		{
			name:     "subsys nil pointers",
			subsys:   &client.NVMetSubsys{ID: 4, Name: "tgt4", Serial: "SN-4"},
			wantID:   "4",
			wantName: "tgt4",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m NVMetSubsysResourceModel
			r.mapResponseToModel(tc.subsys, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q, want %q", m.ID.ValueString(), tc.wantID)
			}
			if m.Name.ValueString() != tc.wantName {
				t.Errorf("Name = %q, want %q", m.Name.ValueString(), tc.wantName)
			}
			if m.AllowAnyHost.ValueBool() != tc.wantAAH {
				t.Errorf("AllowAnyHost = %v, want %v", m.AllowAnyHost.ValueBool(), tc.wantAAH)
			}
			if tc.wantQid != 0 && m.QidMax.ValueInt64() != tc.wantQid {
				t.Errorf("QidMax = %d, want %d", m.QidMax.ValueInt64(), tc.wantQid)
			}
			if tc.wantOui != "" && m.IeeeOui.ValueString() != tc.wantOui {
				t.Errorf("IeeeOui = %q, want %q", m.IeeeOui.ValueString(), tc.wantOui)
			}
			if tc.wantAnaOK && m.Ana.IsNull() {
				t.Error("Ana should not be null")
			}
		})
	}
}

func TestNVMetSubsysResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewNVMetSubsysResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "subnqn", "serial", "allow_any_host", "pi_enable", "qid_max", "ieee_oui", "ana"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}
