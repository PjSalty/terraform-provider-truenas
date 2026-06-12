package datasources

import (
	"context"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestNewSystemDatasetDataSource(t *testing.T) {
	if NewSystemDatasetDataSource() == nil {
		t.Fatal("NewSystemDatasetDataSource returned nil")
	}
}

func TestSystemDatasetDataSource_Schema(t *testing.T) {
	ds := NewSystemDatasetDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "pool", "pool_set", "uuid", "basename", "path"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestSystemDatasetDataSource_Read_Success(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.SystemDataset{
		ID:       1,
		Pool:     "tank",
		PoolSet:  true,
		UUID:     "uuid-1234",
		Basename: "tank/.system",
		Path:     "/mnt/tank/.system",
	}))

	ds := NewSystemDatasetDataSource().(*SystemDatasetDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state SystemDatasetDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Pool.ValueString() != "tank" {
		t.Errorf("Pool: got %q", state.Pool.ValueString())
	}
	if !state.PoolSet.ValueBool() {
		t.Errorf("PoolSet: got %v", state.PoolSet.ValueBool())
	}
	if state.UUID.ValueString() != "uuid-1234" {
		t.Errorf("UUID: got %q", state.UUID.ValueString())
	}
	if state.Path.ValueString() != "/mnt/tank/.system" {
		t.Errorf("Path: got %q", state.Path.ValueString())
	}
	if state.ID.ValueString() != "systemdataset" {
		t.Errorf("ID: got %q, want 'systemdataset'", state.ID.ValueString())
	}
}

func TestSystemDatasetDataSource_Read_ServerError(t *testing.T) {
	c := newWSServer(t.Context(), t, wsError(wsclient.CodeMethodCallError, "simulated server error"))

	ds := NewSystemDatasetDataSource().(*SystemDatasetDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestSystemDatasetDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
