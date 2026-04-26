package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestPoolDataSource_Schema(t *testing.T) {
	ds := NewPoolDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "guid", "path", "status", "healthy", "is_decrypted",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestPoolDataSource_Read_ByID(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/pool/id/3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.Pool{
			ID:          3,
			Name:        "tank",
			GUID:        "1234567890",
			Path:        "/mnt/tank",
			Status:      "ONLINE",
			Healthy:     true,
			IsDecrypted: true,
		})
	}))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(3)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state PoolDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "tank" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.GUID.ValueString() != "1234567890" {
		t.Errorf("GUID: got %q", state.GUID.ValueString())
	}
	if !state.Healthy.ValueBool() {
		t.Errorf("Healthy: got %v", state.Healthy.ValueBool())
	}
}

func TestPoolDataSource_Read_ByName(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Pool{
			{ID: 1, Name: "tank", GUID: "a", Path: "/mnt/tank", Status: "ONLINE", Healthy: true, IsDecrypted: true},
			{ID: 2, Name: "other", GUID: "b"},
		})
	}))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("tank")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state PoolDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueInt64() != 1 {
		t.Errorf("ID: got %d", state.ID.ValueInt64())
	}
	if state.Status.ValueString() != "ONLINE" {
		t.Errorf("Status: got %q", state.Status.ValueString())
	}
}

func TestPoolDataSource_Read_NameNotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Pool{{ID: 1, Name: "other"}})
	}))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestPoolDataSource_Read_IDNotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestPoolDataSource_Read_MissingKey(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Pool{})
	}))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing id/name")
	}
}

func TestPoolDataSource_mapPoolToModel(t *testing.T) {
	ds := &PoolDataSource{}
	p := &client.Pool{ID: 1, Name: "n", GUID: "g", Path: "/p", Status: "ONLINE", Healthy: true, IsDecrypted: false}
	var m PoolDataSourceModel
	ds.mapPoolToModel(p, &m)
	if m.Name.ValueString() != "n" {
		t.Errorf("Name: got %q", m.Name.ValueString())
	}
	if m.IsDecrypted.ValueBool() != false {
		t.Errorf("IsDecrypted: got %v", m.IsDecrypted.ValueBool())
	}
}
