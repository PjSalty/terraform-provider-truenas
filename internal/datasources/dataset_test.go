package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestDatasetDataSource_Schema(t *testing.T) {
	ds := NewDatasetDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "pool", "type", "mount_point", "compression",
		"atime", "deduplication", "quota", "refquota", "comments",
		"sync", "readonly", "record_size",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestDatasetDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.DatasetResponse{
			ID:         "tank/data",
			Name:       "data",
			Pool:       "tank",
			Type:       "FILESYSTEM",
			MountPoint: "/mnt/tank/data",
			Compression: &client.PropertyValue{
				Value: "lz4",
			},
			Atime: &client.PropertyValue{Value: "on"},
			Quota: &client.PropertyRawVal{
				Value:    "10G",
				Rawvalue: "10737418240",
			},
			Refquota: &client.PropertyRawVal{
				Value:    "5G",
				Rawvalue: "5368709120",
			},
			Sync:          &client.PropertyValue{Value: "standard"},
			Readonly:      &client.PropertyValue{Value: "off"},
			RecordSize:    &client.PropertyValue{Value: "128K"},
			Deduplication: &client.PropertyValue{Value: "off"},
			Comments:      &client.PropertyValue{Value: "hello"},
		})
	}))

	ds := NewDatasetDataSource().(*DatasetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"id": strVal("tank/data"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state DatasetDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "data" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.Pool.ValueString() != "tank" {
		t.Errorf("Pool: got %q", state.Pool.ValueString())
	}
	if state.MountPoint.ValueString() != "/mnt/tank/data" {
		t.Errorf("MountPoint: got %q", state.MountPoint.ValueString())
	}
	if state.Compression.ValueString() != "lz4" {
		t.Errorf("Compression: got %q", state.Compression.ValueString())
	}
	if state.Quota.ValueInt64() != 10737418240 {
		t.Errorf("Quota: got %d", state.Quota.ValueInt64())
	}
	if state.Refquota.ValueInt64() != 5368709120 {
		t.Errorf("Refquota: got %d", state.Refquota.ValueInt64())
	}
}

func TestDatasetDataSource_Read_MinimalFields(t *testing.T) {
	// Only required fields populated — none of the property pointers set.
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.DatasetResponse{
			ID:   "tank/minimal",
			Name: "minimal",
			Pool: "tank",
			Type: "FILESYSTEM",
		})
	}))

	ds := NewDatasetDataSource().(*DatasetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"id": strVal("tank/minimal"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
}

func TestDatasetDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
	}))

	ds := NewDatasetDataSource().(*DatasetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"id": strVal("tank/missing"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestDatasetDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewDatasetDataSource().(*DatasetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"id": strVal("tank/x"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}
