package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestNewAlertServiceDataSource(t *testing.T) {
	if NewAlertServiceDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestAlertServiceDataSource_Schema(t *testing.T) {
	ds := NewAlertServiceDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	for _, want := range []string{"id", "name", "type", "enabled", "level", "settings_json"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestAlertServiceDataSource_Read_Success(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.AlertService{
		ID:      3,
		Name:    "slack",
		Enabled: true,
		Level:   "WARNING",
		Settings: map[string]interface{}{
			"type":    "Slack",
			"cluster": "prod",
		},
	}))

	ds := NewAlertServiceDataSource().(*AlertServiceDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(3)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state AlertServiceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "slack" {
		t.Errorf("Name: %q", state.Name.ValueString())
	}
	if state.Type.ValueString() != "Slack" {
		t.Errorf("Type: %q", state.Type.ValueString())
	}
	if state.Level.ValueString() != "WARNING" {
		t.Errorf("Level: %q", state.Level.ValueString())
	}
	if !strings.Contains(state.SettingsJSON.ValueString(), "prod") {
		t.Errorf("SettingsJSON: %q", state.SettingsJSON.ValueString())
	}
}

func TestAlertServiceDataSource_Read_NilSettings(t *testing.T) {
	skipWSCutover(t)
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, truenas.AlertService{ID: 1, Name: "x", Level: "INFO"})
	}))
	ds := NewAlertServiceDataSource().(*AlertServiceDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state AlertServiceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.SettingsJSON.ValueString() != "{}" {
		t.Errorf("expected empty, got %q", state.SettingsJSON.ValueString())
	}
}

func TestAlertServiceDataSource_Read_NotFound(t *testing.T) {
	skipWSCutover(t)
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	ds := NewAlertServiceDataSource().(*AlertServiceDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestAlertServiceDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
