package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewAppsDataSource(t *testing.T) {
	if NewAppsDataSource() == nil {
		t.Fatal("NewAppsDataSource returned nil")
	}
}

func TestAppsDataSource_Schema(t *testing.T) {
	ds := NewAppsDataSource()
	resp := getDataSourceSchema(t, ds)
	if _, ok := resp.Schema.GetAttributes()["apps"]; !ok {
		t.Errorf("missing attribute: apps")
	}
}

func TestAppsDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.App{
			{ID: "a", Name: "a", State: "RUNNING", Version: "1.0", UpgradeAvailable: false, CustomApp: false},
			{ID: "b", Name: "b", State: "STOPPED", Version: "2.0", UpgradeAvailable: true, CustomApp: true},
		})
	}))

	ds := NewAppsDataSource().(*AppsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state AppsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Apps.IsNull() {
		t.Fatal("Apps list is null")
	}
	if len(state.Apps.Elements()) != 2 {
		t.Errorf("Apps: got %d elements, want 2", len(state.Apps.Elements()))
	}
}

func TestAppsDataSource_Read_Empty(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.App{})
	}))

	ds := NewAppsDataSource().(*AppsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state AppsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.Apps.Elements()) != 0 {
		t.Errorf("Apps: got %d elements, want 0", len(state.Apps.Elements()))
	}
}

func TestAppsDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewAppsDataSource().(*AppsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestAppsDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
