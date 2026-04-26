package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewSnapshotTaskDataSource(t *testing.T) {
	if NewSnapshotTaskDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestSnapshotTaskDataSource_Schema(t *testing.T) {
	ds := NewSnapshotTaskDataSource()
	resp := getDataSourceSchema(t, ds)
	for _, want := range []string{
		"id", "dataset", "recursive", "lifetime_value", "lifetime_unit",
		"naming_schema", "enabled", "allow_empty", "exclude",
		"schedule_minute", "schedule_hour", "schedule_dom", "schedule_month", "schedule_dow",
	} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestSnapshotTaskDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/pool/snapshottask/id/11" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.SnapshotTask{
			ID:           11,
			Dataset:      "tank/data",
			Recursive:    true,
			Lifetime:     30,
			LifetimeUnit: "DAY",
			NamingSchema: "auto-%Y-%m-%d_%H-%M",
			Enabled:      true,
			AllowEmpty:   false,
			Exclude:      []string{"tank/data/scratch"},
			Schedule: client.Schedule{
				Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*",
			},
		})
	}))

	ds := NewSnapshotTaskDataSource().(*SnapshotTaskDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(11)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state SnapshotTaskDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Dataset.ValueString() != "tank/data" {
		t.Errorf("Dataset: %q", state.Dataset.ValueString())
	}
	if !state.Recursive.ValueBool() {
		t.Errorf("Recursive false")
	}
	if state.LifetimeUnit.ValueString() != "DAY" {
		t.Errorf("LifetimeUnit: %q", state.LifetimeUnit.ValueString())
	}
	if state.Exclude.IsNull() {
		t.Fatal("Exclude null")
	}
}

func TestSnapshotTaskDataSource_Read_Empty(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, client.SnapshotTask{ID: 1, Schedule: client.Schedule{Minute: "*/15"}})
	}))
	ds := NewSnapshotTaskDataSource().(*SnapshotTaskDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
}

func TestSnapshotTaskDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	ds := NewSnapshotTaskDataSource().(*SnapshotTaskDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestSnapshotTaskDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
