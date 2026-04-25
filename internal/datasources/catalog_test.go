package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewCatalogDataSource(t *testing.T) {
	if NewCatalogDataSource() == nil {
		t.Fatal("NewCatalogDataSource returned nil")
	}
}

func TestCatalogDataSource_Schema(t *testing.T) {
	ds := NewCatalogDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "label", "preferred_trains", "location"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestCatalogDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.Catalog{
			ID:              "TRUENAS",
			Label:           "TRUENAS",
			PreferredTrains: []string{"stable", "community"},
			Location:        "/mnt/.ix-apps/catalog",
		})
	}))

	ds := NewCatalogDataSource().(*CatalogDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state CatalogDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Label.ValueString() != "TRUENAS" {
		t.Errorf("Label: got %q", state.Label.ValueString())
	}
	if state.Location.ValueString() != "/mnt/.ix-apps/catalog" {
		t.Errorf("Location: got %q", state.Location.ValueString())
	}
	if len(state.PreferredTrains.Elements()) != 2 {
		t.Errorf("PreferredTrains: got %d elements", len(state.PreferredTrains.Elements()))
	}
}

func TestCatalogDataSource_Read_EmptyID(t *testing.T) {
	// Should default ID to "catalog" when blank.
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.Catalog{
			ID:              "",
			Label:           "TRUENAS",
			PreferredTrains: []string{},
		})
	}))

	ds := NewCatalogDataSource().(*CatalogDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state CatalogDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "catalog" {
		t.Errorf("ID: got %q, want 'catalog'", state.ID.ValueString())
	}
}

func TestCatalogDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewCatalogDataSource().(*CatalogDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestCatalogDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
