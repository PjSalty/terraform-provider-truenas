package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewShareNFSDataSource(t *testing.T) {
	if NewShareNFSDataSource() == nil {
		t.Fatal("NewShareNFSDataSource returned nil")
	}
}

func TestShareNFSDataSource_Schema(t *testing.T) {
	ds := NewShareNFSDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "path", "aliases", "comment", "hosts", "read_only",
		"maproot_user", "maproot_group", "mapall_user", "mapall_group",
		"security", "enabled", "networks",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestShareNFSDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.NFSShare{
			ID:           42,
			Path:         "/mnt/tank/exports",
			Aliases:      []string{"/exports"},
			Comment:      "shared data",
			Hosts:        []string{"client1", "client2"},
			ReadOnly:     false,
			MaprootUser:  "root",
			MaprootGroup: "wheel",
			Security:     []string{"SYS"},
			Enabled:      true,
			Networks:     []string{"192.168.1.0/24"},
		})
	}))

	ds := NewShareNFSDataSource().(*ShareNFSDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(42)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state ShareNFSDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Path.ValueString() != "/mnt/tank/exports" {
		t.Errorf("Path: got %q", state.Path.ValueString())
	}
	if state.Comment.ValueString() != "shared data" {
		t.Errorf("Comment: got %q", state.Comment.ValueString())
	}
	if len(state.Hosts.Elements()) != 2 {
		t.Errorf("Hosts: got %d, want 2", len(state.Hosts.Elements()))
	}
	if len(state.Networks.Elements()) != 1 {
		t.Errorf("Networks: got %d, want 1", len(state.Networks.Elements()))
	}
	if !state.Enabled.ValueBool() {
		t.Errorf("Enabled: got %v", state.Enabled.ValueBool())
	}
}

func TestShareNFSDataSource_Read_EmptyLists(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.NFSShare{ID: 1, Path: "/mnt/tank/x", Enabled: false})
	}))

	ds := NewShareNFSDataSource().(*ShareNFSDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state ShareNFSDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.Hosts.Elements()) != 0 {
		t.Errorf("Hosts: got %d, want 0", len(state.Hosts.Elements()))
	}
}

func TestShareNFSDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewShareNFSDataSource().(*ShareNFSDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestShareNFSDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewShareNFSDataSource().(*ShareNFSDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestShareNFSDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
