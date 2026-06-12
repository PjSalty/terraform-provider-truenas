package datasources

import (
	"context"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestNewAppDataSource(t *testing.T) {
	if NewAppDataSource() == nil {
		t.Fatal("NewAppDataSource returned nil")
	}
}

func TestAppDataSource_Schema(t *testing.T) {
	ds := NewAppDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
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
	c := newWSServer(t.Context(), t, wsReturn(truenas.App{
		ID:               "jellyfin",
		Name:             "jellyfin",
		State:            "RUNNING",
		UpgradeAvailable: true,
		LatestVersion:    "1.2.0",
		HumanVersion:     "1.1.0 (abc)",
		Version:          "1.1.0",
		CustomApp:        false,
		Migrated:         true,
	}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": strVal("jellyfin")})
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
	c := newWSServer(t.Context(), t, wsError(wsclient.CodeMethodCallError, "simulated server error"))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": strVal("any")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestAppDataSource_Read_ListResponse(t *testing.T) {
	// REST-era GetApp had a single-element-list fallback; the WS
	// app.get_instance contract returns a bare object. Keep the test
	// name for history but assert the object path.
	c := newWSServer(t.Context(), t, wsReturn(truenas.App{
		ID: "nextcloud", Name: "nextcloud", State: "STOPPED",
	}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": strVal("nextcloud")})
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
	c := newWSServer(t.Context(), t, wsReturn([]truenas.App{}))

	ds := NewAppDataSource().(*AppDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected 404 error")
	}
}

func TestAppDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
