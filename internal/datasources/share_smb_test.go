package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewShareSMBDataSource(t *testing.T) {
	if NewShareSMBDataSource() == nil {
		t.Fatal("NewShareSMBDataSource returned nil")
	}
}

func TestShareSMBDataSource_Schema(t *testing.T) {
	ds := NewShareSMBDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "path", "name", "comment", "browsable", "read_only",
		"access_based_share_enumeration", "enabled", "purpose",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestShareSMBDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.SMBShare{
			ID:        7,
			Path:      "/mnt/tank/smb",
			Name:      "public",
			Comment:   "public share",
			Browsable: true,
			ReadOnly:  false,
			ABE:       true,
			Enabled:   true,
			Purpose:   "NO_PRESET",
		})
	}))

	ds := NewShareSMBDataSource().(*ShareSMBDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(7)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state ShareSMBDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "public" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.Path.ValueString() != "/mnt/tank/smb" {
		t.Errorf("Path: got %q", state.Path.ValueString())
	}
	if !state.Browsable.ValueBool() {
		t.Errorf("Browsable: got %v", state.Browsable.ValueBool())
	}
	if !state.ABE.ValueBool() {
		t.Errorf("ABE: got %v", state.ABE.ValueBool())
	}
	if state.Purpose.ValueString() != "NO_PRESET" {
		t.Errorf("Purpose: got %q", state.Purpose.ValueString())
	}
}

func TestShareSMBDataSource_Read_ReadOnly(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.SMBShare{ID: 8, Name: "ro", ReadOnly: true, Enabled: false})
	}))

	ds := NewShareSMBDataSource().(*ShareSMBDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(8)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state ShareSMBDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if !state.ReadOnly.ValueBool() {
		t.Errorf("ReadOnly: got %v", state.ReadOnly.ValueBool())
	}
}

func TestShareSMBDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewShareSMBDataSource().(*ShareSMBDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestShareSMBDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
