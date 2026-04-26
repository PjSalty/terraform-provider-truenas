package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewVMDataSource(t *testing.T) {
	if NewVMDataSource() == nil {
		t.Fatal("NewVMDataSource returned nil")
	}
}

func TestVMDataSource_Schema(t *testing.T) {
	ds := NewVMDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "description", "vcpus", "cores", "threads",
		"memory", "min_memory", "bootloader", "bootloader_ovmf",
		"autostart", "time", "shutdown_timeout", "cpu_mode", "cpu_model",
		"enable_secure_boot", "state", "uuid",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestVMDataSource_Read_Success(t *testing.T) {
	uuid := "abc-123"
	cpuModel := "host"
	minMem := int64(1073741824)
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.VM{
			ID:               12,
			Name:             "vm1",
			Description:      "test vm",
			Vcpus:            2,
			Cores:            4,
			Threads:          2,
			Memory:           4294967296,
			MinMemory:        &minMem,
			Bootloader:       "UEFI",
			BootloaderOvmf:   "OVMF_CODE.fd",
			Autostart:        true,
			Time:             "UTC",
			ShutdownTimeout:  90,
			CPUMode:          "CUSTOM",
			CPUModel:         &cpuModel,
			EnableSecureBoot: true,
			UUID:             &uuid,
			Status:           &client.VMStatus{State: "RUNNING"},
		})
	}))

	ds := NewVMDataSource().(*VMDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(12)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state VMDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "vm1" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.Memory.ValueInt64() != 4294967296 {
		t.Errorf("Memory: got %d", state.Memory.ValueInt64())
	}
	if state.MinMemory.ValueInt64() != 1073741824 {
		t.Errorf("MinMemory: got %d", state.MinMemory.ValueInt64())
	}
	if state.State.ValueString() != "RUNNING" {
		t.Errorf("State: got %q", state.State.ValueString())
	}
	if state.UUID.ValueString() != "abc-123" {
		t.Errorf("UUID: got %q", state.UUID.ValueString())
	}
	if state.CPUModel.ValueString() != "host" {
		t.Errorf("CPUModel: got %q", state.CPUModel.ValueString())
	}
}

func TestVMDataSource_Read_NoStatus(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.VM{ID: 1, Name: "vm", Memory: 1024})
	}))

	ds := NewVMDataSource().(*VMDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state VMDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if !state.State.IsNull() {
		t.Errorf("State: expected null")
	}
	if !state.UUID.IsNull() {
		t.Errorf("UUID: expected null")
	}
	if !state.MinMemory.IsNull() {
		t.Errorf("MinMemory: expected null")
	}
}

func TestVMDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewVMDataSource().(*VMDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestVMDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
