package datasources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewISCSIExtentDataSource(t *testing.T) {
	if NewISCSIExtentDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestISCSIExtentDataSource_Schema(t *testing.T) {
	ds := NewISCSIExtentDataSource()
	resp := getDataSourceSchema(t, ds)
	for _, want := range []string{"id", "name", "type", "disk", "path", "filesize", "blocksize", "rpm", "enabled", "comment", "readonly", "xen", "insecure_tpc"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestISCSIExtentDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/iscsi/extent/id/4" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.ISCSIExtent{
			ID:        4,
			Name:      "ext1",
			Type:      "DISK",
			Disk:      json.RawMessage(`"zvol/tank/vol1"`),
			Filesize:  json.RawMessage(`0`),
			Blocksize: 512,
			RPM:       "SSD",
			Enabled:   true,
			Comment:   "prod",
			ReadOnly:  false,
		})
	}))

	ds := NewISCSIExtentDataSource().(*ISCSIExtentDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(4)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state ISCSIExtentDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "ext1" {
		t.Errorf("Name: %q", state.Name.ValueString())
	}
	if state.Type.ValueString() != "DISK" {
		t.Errorf("Type: %q", state.Type.ValueString())
	}
	if state.Disk.ValueString() != "zvol/tank/vol1" {
		t.Errorf("Disk: %q", state.Disk.ValueString())
	}
	if state.Blocksize.ValueInt64() != 512 {
		t.Errorf("Blocksize: %d", state.Blocksize.ValueInt64())
	}
}

func TestISCSIExtentDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	ds := NewISCSIExtentDataSource().(*ISCSIExtentDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestISCSIExtentDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
