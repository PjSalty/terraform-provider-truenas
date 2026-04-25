package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewAppDataSource(t *testing.T) {
	if NewAppDataSource() == nil {
		t.Fatal("NewAppDataSource returned nil")
	}
}

func TestAppDataSource_Schema(t *testing.T) {
	ds := NewAppDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "state", "upgrade_available", "latest_version",
		"human_version", "version", "custom_app", "migrated",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestAppDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.App{
			ID:               "jellyfin",
			Name:             "jellyfin",
			State:            "RUNNING",
			UpgradeAvailable: true,
			LatestVersion:    "1.2.0",
			HumanVersion:     "1.1.0 (abc)",
			Version:          "1.1.0",
			CustomApp:        false,
			Migrated:         true,
		})
	}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("jellyfin")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state AppDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "jellyfin" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if !state.UpgradeAvailable.ValueBool() {
		t.Errorf("UpgradeAvailable: got %v", state.UpgradeAvailable.ValueBool())
	}
	if state.LatestVersion.ValueString() != "1.2.0" {
		t.Errorf("LatestVersion: got %q", state.LatestVersion.ValueString())
	}
	if !state.Migrated.ValueBool() {
		t.Errorf("Migrated: got %v", state.Migrated.ValueBool())
	}
}

func TestAppDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("any")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestAppDataSource_Read_ListResponse(t *testing.T) {
	// GetApp falls back to parsing a single-element list.
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.App{
			{ID: "nextcloud", Name: "nextcloud", State: "STOPPED"},
		})
	}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("nextcloud")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state AppDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.State.ValueString() != "STOPPED" {
		t.Errorf("State: got %q", state.State.ValueString())
	}
}

func TestAppDataSource_Read_EmptyList(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.App{})
	}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected 404 error")
	}
}

func TestAppDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
