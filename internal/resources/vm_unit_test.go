package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func int64Ptr(i int64) *int64 { return &i }
func intPtr(i int) *int       { return &i }

func TestVMResource_MapResponseToModel_Cases(t *testing.T) {
	r := &VMResource{}
	cases := []struct {
		name     string
		vm       *client.VM
		wantID   string
		wantName string
		wantCPU  int64
		wantMem  int64
		wantStat string
	}{
		{
			name:     "minimal vm",
			vm:       &client.VM{ID: 1, Name: "vm1", Vcpus: 2, Cores: 1, Threads: 1, Memory: 2048, Bootloader: "UEFI"},
			wantID:   "1",
			wantName: "vm1",
			wantCPU:  2,
			wantMem:  2048,
		},
		{
			name: "running vm",
			vm: &client.VM{
				ID: 2, Name: "running", Vcpus: 4, Cores: 2, Threads: 2, Memory: 8192,
				Bootloader: "UEFI", Autostart: true,
				Status: &client.VMStatus{State: "RUNNING"},
			},
			wantID: "2", wantName: "running", wantCPU: 4, wantMem: 8192, wantStat: "RUNNING",
		},
		{
			name: "vm with min memory",
			vm: &client.VM{
				ID: 3, Name: "dyn", Vcpus: 2, Cores: 1, Threads: 1, Memory: 4096,
				MinMemory:  int64Ptr(2048),
				Bootloader: "UEFI",
			},
			wantID: "3", wantName: "dyn", wantCPU: 2, wantMem: 4096,
		},
		{
			name:   "vm with nil status",
			vm:     &client.VM{ID: 4, Name: "stopped", Vcpus: 1, Cores: 1, Threads: 1, Memory: 512, Bootloader: "UEFI"},
			wantID: "4", wantName: "stopped", wantCPU: 1, wantMem: 512, wantStat: "",
		},
		{
			name: "vm with high resources",
			vm: &client.VM{
				ID: 5, Name: "big", Vcpus: 16, Cores: 8, Threads: 2, Memory: 65536,
				Bootloader: "UEFI", Autostart: true,
				Status: &client.VMStatus{State: "STOPPED"},
			},
			wantID: "5", wantName: "big", wantCPU: 16, wantMem: 65536, wantStat: "STOPPED",
		},
		{
			name: "vm with BIOS bootloader",
			vm: &client.VM{
				ID: 6, Name: "legacy", Vcpus: 1, Cores: 1, Threads: 1, Memory: 1024,
				Bootloader: "BIOS", Autostart: false,
			},
			wantID: "6", wantName: "legacy", wantCPU: 1, wantMem: 1024,
		},
		{
			name: "vm with description",
			vm: &client.VM{
				ID: 7, Name: "desc", Vcpus: 2, Cores: 1, Threads: 2, Memory: 2048,
				Description: "test vm", Bootloader: "UEFI",
			},
			wantID: "7", wantName: "desc", wantCPU: 2, wantMem: 2048,
		},
		{
			name: "vm suspended",
			vm: &client.VM{
				ID: 8, Name: "paused", Vcpus: 4, Cores: 2, Threads: 2, Memory: 4096,
				Bootloader: "UEFI",
				Status:     &client.VMStatus{State: "SUSPENDED"},
			},
			wantID: "8", wantName: "paused", wantCPU: 4, wantMem: 4096, wantStat: "SUSPENDED",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m VMResourceModel
			r.mapResponseToModel(tc.vm, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.Name.ValueString() != tc.wantName {
				t.Errorf("Name = %q", m.Name.ValueString())
			}
			if m.Vcpus.ValueInt64() != tc.wantCPU {
				t.Errorf("Vcpus = %d", m.Vcpus.ValueInt64())
			}
			if m.Memory.ValueInt64() != tc.wantMem {
				t.Errorf("Memory = %d", m.Memory.ValueInt64())
			}
			if m.Status.ValueString() != tc.wantStat {
				t.Errorf("Status = %q, want %q", m.Status.ValueString(), tc.wantStat)
			}
		})
	}
}

func TestVMResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewVMResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "vcpus", "cores", "threads", "memory", "bootloader", "autostart"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}

func TestVMDeviceResource_MapResponseToModel_Cases(t *testing.T) {
	r := &VMDeviceResource{}
	ctx := context.Background()
	cases := []struct {
		name      string
		dev       *client.VMDevice
		wantID    string
		wantVM    int64
		wantOrder int64
	}{
		{
			name: "disk device",
			dev: &client.VMDevice{
				ID: 1, VM: 5, Order: intPtr(1001),
				Attributes: map[string]interface{}{"dtype": "DISK", "path": "/dev/zvol/tank/v1"},
			},
			wantID: "1", wantVM: 5, wantOrder: 1001,
		},
		{
			name: "nic device no order",
			dev: &client.VMDevice{
				ID: 2, VM: 5,
				Attributes: map[string]interface{}{"dtype": "NIC", "nic_attach": "br0"},
			},
			wantID: "2", wantVM: 5,
		},
		{
			name: "display device",
			dev: &client.VMDevice{
				ID: 3, VM: 7, Order: intPtr(1),
				Attributes: map[string]interface{}{"dtype": "DISPLAY"},
			},
			wantID: "3", wantVM: 7, wantOrder: 1,
		},
		{
			name: "cdrom device",
			dev: &client.VMDevice{
				ID: 4, VM: 1, Order: intPtr(999),
				Attributes: map[string]interface{}{"dtype": "CDROM"},
			},
			wantID: "4", wantVM: 1, wantOrder: 999,
		},
		{
			name: "raw device",
			dev: &client.VMDevice{
				ID: 5, VM: 10, Order: intPtr(100),
				Attributes: map[string]interface{}{
					"dtype":      "RAW",
					"path":       "/dev/disk/by-id/abc",
					"type":       "AHCI",
					"sectorsize": float64(512),
				},
			},
			wantID: "5", wantVM: 10, wantOrder: 100,
		},
		{
			name: "pci passthrough",
			dev: &client.VMDevice{
				ID: 6, VM: 20, Order: intPtr(500),
				Attributes: map[string]interface{}{
					"dtype":  "PCI",
					"pptdev": "pci_0000_01_00_0",
				},
			},
			wantID: "6", wantVM: 20, wantOrder: 500,
		},
		{
			name: "usb device",
			dev: &client.VMDevice{
				ID: 7, VM: 30,
				Attributes: map[string]interface{}{"dtype": "USB"},
			},
			wantID: "7", wantVM: 30,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m VMDeviceResourceModel
			r.mapResponseToModel(ctx, tc.dev, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.VM.ValueInt64() != tc.wantVM {
				t.Errorf("VM = %d", m.VM.ValueInt64())
			}
			if tc.dev.Order != nil && m.Order.ValueInt64() != tc.wantOrder {
				t.Errorf("Order = %d, want %d", m.Order.ValueInt64(), tc.wantOrder)
			}
			if tc.dev.Order == nil && !m.Order.IsNull() {
				t.Errorf("Order should be null when dev.Order is nil")
			}
		})
	}
}

func TestVMDeviceResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewVMDeviceResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "vm", "dtype", "attributes"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["vm"].IsRequired() {
		t.Error("vm should be required")
	}
	if !attrs["dtype"].IsRequired() {
		t.Error("dtype should be required")
	}
}
