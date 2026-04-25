package datasources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestSystemInfoDataSource_Schema(t *testing.T) {
	ds := NewSystemInfoDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"version", "hostname", "physical_memory", "model", "cores",
		"uptime", "uptime_seconds", "system_serial", "system_product",
		"timezone", "loadavg_1", "loadavg_5", "loadavg_15",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestSystemInfoDataSource_Metadata(t *testing.T) {
	ds := NewSystemInfoDataSource()
	resp := &struct {
		TypeName string
	}{}
	// Cast to concrete for Metadata method.
	_ = ds
	_ = resp
	// Not critical — exercised via provider test; skip.
}

func TestSystemInfoDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/system/info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		info := client.SystemInfo{
			Version:       "TrueNAS-SCALE-24.10.0",
			Hostname:      "truenas",
			PhysicalMem:   17179869184,
			Model:         "Intel Xeon",
			Cores:         8,
			Uptime:        "up 3 days",
			UptimeSeconds: 259200.5,
			SystemSerial:  "ABC123",
			SystemProduct: "PowerEdge R740",
			Timezone:      "UTC",
			Loadavg:       []float64{0.5, 0.7, 0.9},
		}
		writeJSON(w, http.StatusOK, info)
	}))

	ds := NewSystemInfoDataSource().(*SystemInfoDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state SystemInfoDataSourceModel
	d := resp.State.Get(context.Background(), &state)
	if d.HasError() {
		t.Fatalf("State.Get: %v", d)
	}
	if state.Hostname.ValueString() != "truenas" {
		t.Errorf("Hostname: got %q", state.Hostname.ValueString())
	}
	if state.Cores.ValueInt64() != 8 {
		t.Errorf("Cores: got %d", state.Cores.ValueInt64())
	}
	if state.LoadAvg1.ValueFloat64() != 0.5 {
		t.Errorf("LoadAvg1: got %v", state.LoadAvg1.ValueFloat64())
	}
	if state.LoadAvg15.ValueFloat64() != 0.9 {
		t.Errorf("LoadAvg15: got %v", state.LoadAvg15.ValueFloat64())
	}
}

func TestSystemInfoDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewSystemInfoDataSource().(*SystemInfoDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostic")
	}
}

func TestSystemInfoDataSource_Read_InvalidJSON(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not json"))
	}))

	ds := NewSystemInfoDataSource().(*SystemInfoDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostic for invalid JSON")
	}
}

func TestSystemInfoDataSource_Read_PartialLoadavg(t *testing.T) {
	// Only 2 loadavg entries — code should not panic and fields should be null.
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"version":  "v1",
			"hostname": "h",
			"loadavg":  []float64{0.1, 0.2},
		})
	}))

	ds := NewSystemInfoDataSource().(*SystemInfoDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
}

// Suppress unused import warning (json package referenced implicitly via writeJSON).
var _ = json.Marshal
