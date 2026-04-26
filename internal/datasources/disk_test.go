package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestDiskDataSource_Schema(t *testing.T) {
	ds := NewDiskDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"name", "serial", "size", "type", "pool", "model",
		"description", "bus", "identifier", "devname",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestDiskDataSource_Read_Success(t *testing.T) {
	pool := "tank"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Disk{
			{
				Name:        "sda",
				Serial:      "S1",
				Size:        1099511627776,
				Type:        "HDD",
				Model:       "WD",
				Description: "data disk",
				Bus:         "SATA",
				Identifier:  "{serial}S1",
				Devname:     "sda",
				Pool:        &pool,
			},
			{Name: "sdb", Serial: "S2", Size: 500000000000, Type: "SSD"},
		})
	}))

	ds := NewDiskDataSource().(*DiskDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("sda")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state DiskDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Serial.ValueString() != "S1" {
		t.Errorf("Serial: got %q", state.Serial.ValueString())
	}
	if state.Size.ValueInt64() != 1099511627776 {
		t.Errorf("Size: got %d", state.Size.ValueInt64())
	}
	if state.Type.ValueString() != "HDD" {
		t.Errorf("Type: got %q", state.Type.ValueString())
	}
	if state.Pool.ValueString() != "tank" {
		t.Errorf("Pool: got %q", state.Pool.ValueString())
	}
	if state.Bus.ValueString() != "SATA" {
		t.Errorf("Bus: got %q", state.Bus.ValueString())
	}
}

func TestDiskDataSource_Read_NilPool(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Disk{
			{Name: "sdc", Serial: "S3", Size: 100, Type: "SSD", Pool: nil},
		})
	}))

	ds := NewDiskDataSource().(*DiskDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("sdc")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state DiskDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Pool.ValueString() != "" {
		t.Errorf("Pool: got %q, want empty", state.Pool.ValueString())
	}
}

func TestDiskDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Disk{{Name: "sda"}})
	}))

	ds := NewDiskDataSource().(*DiskDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("sdz")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing disk")
	}
}

func TestDiskDataSource_Read_ListError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewDiskDataSource().(*DiskDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("sda")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestDiskDataSource_mapDiskToModel(t *testing.T) {
	ds := &DiskDataSource{}
	disk := &client.Disk{
		Name:        "sdd",
		Serial:      "X",
		Size:        42,
		Type:        "SSD",
		Model:       "Samsung",
		Description: "d",
		Bus:         "NVMe",
		Identifier:  "id",
		Devname:     "sdd",
	}
	var m DiskDataSourceModel
	ds.mapDiskToModel(disk, &m)
	if m.Name.ValueString() != "sdd" {
		t.Errorf("Name: got %q", m.Name.ValueString())
	}
	if m.Size.ValueInt64() != 42 {
		t.Errorf("Size: got %d", m.Size.ValueInt64())
	}
	if m.Pool.ValueString() != "" {
		t.Errorf("Pool: got %q, want empty", m.Pool.ValueString())
	}
}
