package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewPoolsDataSource(t *testing.T) {
	if NewPoolsDataSource() == nil {
		t.Fatal("NewPoolsDataSource returned nil")
	}
}

func TestPoolsDataSource_Schema(t *testing.T) {
	ds := NewPoolsDataSource()
	resp := getDataSourceSchema(t, ds)
	if _, ok := resp.Schema.GetAttributes()["pools"]; !ok {
		t.Errorf("missing attribute: pools")
	}
}

func TestPoolsDataSource_Read_Multiple(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Pool{
			{ID: 1, Name: "tank", GUID: "a", Path: "/mnt/tank", Status: "ONLINE", Healthy: true, IsDecrypted: true},
			{ID: 2, Name: "fast", GUID: "b", Path: "/mnt/fast", Status: "DEGRADED", Healthy: false, IsDecrypted: true},
		})
	}))

	ds := NewPoolsDataSource().(*PoolsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state PoolsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.Pools.Elements()) != 2 {
		t.Errorf("Pools: got %d, want 2", len(state.Pools.Elements()))
	}
}

func TestPoolsDataSource_Read_Empty(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Pool{})
	}))

	ds := NewPoolsDataSource().(*PoolsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state PoolsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.Pools.Elements()) != 0 {
		t.Errorf("Pools: got %d, want 0", len(state.Pools.Elements()))
	}
}

func TestPoolsDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewPoolsDataSource().(*PoolsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestPoolsDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
