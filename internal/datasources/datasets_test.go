package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewDatasetsDataSource(t *testing.T) {
	if NewDatasetsDataSource() == nil {
		t.Fatal("NewDatasetsDataSource returned nil")
	}
}

func TestDatasetsDataSource_Schema(t *testing.T) {
	ds := NewDatasetsDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"pool", "parent_dataset", "datasets"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func datasetsFixture() []client.DatasetResponse {
	return []client.DatasetResponse{
		{ID: "tank", Name: "tank", Pool: "tank", Type: "FILESYSTEM", MountPoint: "/mnt/tank"},
		{ID: "tank/data", Name: "data", Pool: "tank", Type: "FILESYSTEM", MountPoint: "/mnt/tank/data"},
		{ID: "tank/data/sub", Name: "sub", Pool: "tank", Type: "FILESYSTEM", MountPoint: "/mnt/tank/data/sub"},
		{ID: "other", Name: "other", Pool: "other", Type: "FILESYSTEM", MountPoint: "/mnt/other"},
	}
}

func TestDatasetsDataSource_Read_All(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, datasetsFixture())
	}))

	ds := NewDatasetsDataSource().(*DatasetsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state DatasetsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.Datasets.Elements()) != 4 {
		t.Errorf("Datasets: got %d, want 4", len(state.Datasets.Elements()))
	}
}

func TestDatasetsDataSource_Read_PoolFilter(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, datasetsFixture())
	}))

	ds := NewDatasetsDataSource().(*DatasetsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"pool": strVal("tank")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state DatasetsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.Datasets.Elements()) != 3 {
		t.Errorf("Datasets: got %d, want 3 (tank filter)", len(state.Datasets.Elements()))
	}
}

func TestDatasetsDataSource_Read_ParentFilter(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, datasetsFixture())
	}))

	ds := NewDatasetsDataSource().(*DatasetsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"parent_dataset": strVal("tank/data"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state DatasetsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	// parent_dataset includes both tank/data itself and tank/data/sub
	if len(state.Datasets.Elements()) != 2 {
		t.Errorf("Datasets: got %d, want 2 (parent=tank/data)", len(state.Datasets.Elements()))
	}
}

func TestDatasetsDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewDatasetsDataSource().(*DatasetsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestDatasetsDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
