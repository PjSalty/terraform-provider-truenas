package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestServiceDataSource_Schema(t *testing.T) {
	ds := NewServiceDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "service", "enable", "state"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestServiceDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Service{
			{ID: 1, Service: "ssh", Enable: true, State: "RUNNING"},
			{ID: 2, Service: "nfs", Enable: false, State: "STOPPED"},
		})
	}))

	ds := NewServiceDataSource().(*ServiceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"service": strVal("ssh")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state ServiceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueInt64() != 1 {
		t.Errorf("ID: got %d", state.ID.ValueInt64())
	}
	if state.Enable.ValueBool() != true {
		t.Errorf("Enable: got %v", state.Enable.ValueBool())
	}
	if state.State.ValueString() != "RUNNING" {
		t.Errorf("State: got %q", state.State.ValueString())
	}
}

func TestServiceDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Service{{ID: 1, Service: "ssh"}})
	}))

	ds := NewServiceDataSource().(*ServiceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"service": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestServiceDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewServiceDataSource().(*ServiceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"service": strVal("ssh")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestServiceDataSource_Read_StoppedService(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Service{
			{ID: 5, Service: "smbsrv", Enable: false, State: "STOPPED"},
		})
	}))

	ds := NewServiceDataSource().(*ServiceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"service": strVal("smbsrv")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state ServiceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.State.ValueString() != "STOPPED" {
		t.Errorf("State: got %q", state.State.ValueString())
	}
	if state.Enable.ValueBool() != false {
		t.Errorf("Enable: got %v", state.Enable.ValueBool())
	}
}
