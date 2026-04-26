package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewVMsDataSource(t *testing.T) {
	if NewVMsDataSource() == nil {
		t.Fatal("NewVMsDataSource returned nil")
	}
}

func TestVMsDataSource_Schema(t *testing.T) {
	ds := NewVMsDataSource()
	resp := getDataSourceSchema(t, ds)
	if _, ok := resp.Schema.GetAttributes()["vms"]; !ok {
		t.Errorf("missing attribute: vms")
	}
}

func TestVMsDataSource_Read_Multiple(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.VM{
			{ID: 1, Name: "vm-a", Memory: 1024, Vcpus: 1, Cores: 2, Status: &client.VMStatus{State: "RUNNING"}},
			{ID: 2, Name: "vm-b", Memory: 2048, Vcpus: 2, Cores: 4},
		})
	}))

	ds := NewVMsDataSource().(*VMsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state VMsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.VMs.Elements()) != 2 {
		t.Errorf("VMs: got %d, want 2", len(state.VMs.Elements()))
	}
}

func TestVMsDataSource_Read_Empty(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.VM{})
	}))

	ds := NewVMsDataSource().(*VMsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state VMsDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.VMs.Elements()) != 0 {
		t.Errorf("VMs: got %d, want 0", len(state.VMs.Elements()))
	}
}

func TestVMsDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewVMsDataSource().(*VMsDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestVMsDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
