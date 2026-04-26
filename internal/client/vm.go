package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- VM API ---

// VM represents a TrueNAS SCALE virtual machine.
type VM struct {
	ID                         int        `json:"id"`
	Name                       string     `json:"name"`
	Description                string     `json:"description"`
	Vcpus                      int        `json:"vcpus"`
	Cores                      int        `json:"cores"`
	Threads                    int        `json:"threads"`
	Memory                     int64      `json:"memory"`
	MinMemory                  *int64     `json:"min_memory"`
	Bootloader                 string     `json:"bootloader"`
	BootloaderOvmf             string     `json:"bootloader_ovmf"`
	Autostart                  bool       `json:"autostart"`
	HideFromMsr                bool       `json:"hide_from_msr"`
	EnsureDisplayDevice        bool       `json:"ensure_display_device"`
	Time                       string     `json:"time"`
	ShutdownTimeout            int        `json:"shutdown_timeout"`
	ArchType                   *string    `json:"arch_type"`
	MachineType                *string    `json:"machine_type"`
	UUID                       *string    `json:"uuid"`
	CommandLineArgs            string     `json:"command_line_args"`
	CPUMode                    string     `json:"cpu_mode"`
	CPUModel                   *string    `json:"cpu_model"`
	Cpuset                     *string    `json:"cpuset"`
	Nodeset                    *string    `json:"nodeset"`
	EnableCPUTopologyExtension bool       `json:"enable_cpu_topology_extension"`
	PinVcpus                   bool       `json:"pin_vcpus"`
	SuspendOnSnapshot          bool       `json:"suspend_on_snapshot"`
	TrustedPlatformModule      bool       `json:"trusted_platform_module"`
	HypervEnlightenments       bool       `json:"hyperv_enlightenments"`
	EnableSecureBoot           bool       `json:"enable_secure_boot"`
	Status                     *VMStatus  `json:"status"`
	Devices                    []VMDevice `json:"devices"`
	DisplayAvailable           bool       `json:"display_available"`
}

// VMStatus represents the runtime status of a VM.
type VMStatus struct {
	State       string `json:"state"`
	PID         *int   `json:"pid"`
	DomainState string `json:"domain_state"`
}

// VMCreateRequest represents the request body for creating a VM.
// Fields are pointers where the TrueNAS API treats absence differently from zero value.
type VMCreateRequest struct {
	Name                       string  `json:"name"`
	Description                *string `json:"description,omitempty"`
	Vcpus                      *int    `json:"vcpus,omitempty"`
	Cores                      *int    `json:"cores,omitempty"`
	Threads                    *int    `json:"threads,omitempty"`
	Memory                     int64   `json:"memory"`
	MinMemory                  *int64  `json:"min_memory,omitempty"`
	Bootloader                 *string `json:"bootloader,omitempty"`
	BootloaderOvmf             *string `json:"bootloader_ovmf,omitempty"`
	Autostart                  *bool   `json:"autostart,omitempty"`
	HideFromMsr                *bool   `json:"hide_from_msr,omitempty"`
	EnsureDisplayDevice        *bool   `json:"ensure_display_device,omitempty"`
	Time                       *string `json:"time,omitempty"`
	ShutdownTimeout            *int    `json:"shutdown_timeout,omitempty"`
	ArchType                   *string `json:"arch_type,omitempty"`
	MachineType                *string `json:"machine_type,omitempty"`
	UUID                       *string `json:"uuid,omitempty"`
	CommandLineArgs            *string `json:"command_line_args,omitempty"`
	CPUMode                    *string `json:"cpu_mode,omitempty"`
	CPUModel                   *string `json:"cpu_model,omitempty"`
	Cpuset                     *string `json:"cpuset,omitempty"`
	Nodeset                    *string `json:"nodeset,omitempty"`
	EnableCPUTopologyExtension *bool   `json:"enable_cpu_topology_extension,omitempty"`
	PinVcpus                   *bool   `json:"pin_vcpus,omitempty"`
	SuspendOnSnapshot          *bool   `json:"suspend_on_snapshot,omitempty"`
	TrustedPlatformModule      *bool   `json:"trusted_platform_module,omitempty"`
	HypervEnlightenments       *bool   `json:"hyperv_enlightenments,omitempty"`
	EnableSecureBoot           *bool   `json:"enable_secure_boot,omitempty"`
}

// VMUpdateRequest is identical in shape to VMCreateRequest but memory is optional.
type VMUpdateRequest struct {
	Name                       *string `json:"name,omitempty"`
	Description                *string `json:"description,omitempty"`
	Vcpus                      *int    `json:"vcpus,omitempty"`
	Cores                      *int    `json:"cores,omitempty"`
	Threads                    *int    `json:"threads,omitempty"`
	Memory                     *int64  `json:"memory,omitempty"`
	MinMemory                  *int64  `json:"min_memory,omitempty"`
	Bootloader                 *string `json:"bootloader,omitempty"`
	BootloaderOvmf             *string `json:"bootloader_ovmf,omitempty"`
	Autostart                  *bool   `json:"autostart,omitempty"`
	HideFromMsr                *bool   `json:"hide_from_msr,omitempty"`
	EnsureDisplayDevice        *bool   `json:"ensure_display_device,omitempty"`
	Time                       *string `json:"time,omitempty"`
	ShutdownTimeout            *int    `json:"shutdown_timeout,omitempty"`
	ArchType                   *string `json:"arch_type,omitempty"`
	MachineType                *string `json:"machine_type,omitempty"`
	CommandLineArgs            *string `json:"command_line_args,omitempty"`
	CPUMode                    *string `json:"cpu_mode,omitempty"`
	CPUModel                   *string `json:"cpu_model,omitempty"`
	Cpuset                     *string `json:"cpuset,omitempty"`
	Nodeset                    *string `json:"nodeset,omitempty"`
	EnableCPUTopologyExtension *bool   `json:"enable_cpu_topology_extension,omitempty"`
	PinVcpus                   *bool   `json:"pin_vcpus,omitempty"`
	SuspendOnSnapshot          *bool   `json:"suspend_on_snapshot,omitempty"`
	TrustedPlatformModule      *bool   `json:"trusted_platform_module,omitempty"`
	HypervEnlightenments       *bool   `json:"hyperv_enlightenments,omitempty"`
	EnableSecureBoot           *bool   `json:"enable_secure_boot,omitempty"`
}

// VMDeleteOptions represents the options accepted by the VM delete endpoint.
type VMDeleteOptions struct {
	Zvols bool `json:"zvols"`
	Force bool `json:"force"`
}

// ListVMs retrieves all VMs.
func (c *Client) ListVMs(ctx context.Context) ([]VM, error) {
	tflog.Trace(ctx, "ListVMs start")

	resp, err := c.Get(ctx, "/vm")
	if err != nil {
		return nil, fmt.Errorf("listing VMs: %w", err)
	}

	var vms []VM
	if err := json.Unmarshal(resp, &vms); err != nil {
		return nil, fmt.Errorf("parsing VM list response: %w", err)
	}
	tflog.Trace(ctx, "ListVMs success")
	return vms, nil
}

// GetVM retrieves a VM by its numeric ID.
func (c *Client) GetVM(ctx context.Context, id int) (*VM, error) {
	tflog.Trace(ctx, "GetVM start")

	resp, err := c.Get(ctx, fmt.Sprintf("/vm/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting VM %d: %w", id, err)
	}

	var vm VM
	if err := json.Unmarshal(resp, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM response: %w", err)
	}
	tflog.Trace(ctx, "GetVM success")
	return &vm, nil
}

// CreateVM creates a new VM.
func (c *Client) CreateVM(ctx context.Context, req *VMCreateRequest) (*VM, error) {
	tflog.Trace(ctx, "CreateVM start")

	resp, err := c.Post(ctx, "/vm", req)
	if err != nil {
		return nil, fmt.Errorf("creating VM %q: %w", req.Name, err)
	}

	var vm VM
	if err := json.Unmarshal(resp, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM create response: %w", err)
	}
	tflog.Trace(ctx, "CreateVM success")
	return &vm, nil
}

// UpdateVM updates an existing VM.
func (c *Client) UpdateVM(ctx context.Context, id int, req *VMUpdateRequest) (*VM, error) {
	tflog.Trace(ctx, "UpdateVM start")

	resp, err := c.Put(ctx, fmt.Sprintf("/vm/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating VM %d: %w", id, err)
	}

	var vm VM
	if err := json.Unmarshal(resp, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateVM success")
	return &vm, nil
}

// DeleteVM deletes a VM. The body controls whether associated zvols are also removed
// and whether a running VM is forcibly stopped.
func (c *Client) DeleteVM(ctx context.Context, id int, opts *VMDeleteOptions) error {
	tflog.Trace(ctx, "DeleteVM start")

	if opts == nil {
		opts = &VMDeleteOptions{Force: true, Zvols: false}
	}
	_, err := c.DeleteWithBody(ctx, fmt.Sprintf("/vm/id/%d", id), opts)
	if err != nil {
		return fmt.Errorf("deleting VM %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteVM success")
	return nil
}

// StartVM powers on a VM.
func (c *Client) StartVM(ctx context.Context, id int) error {
	tflog.Trace(ctx, "StartVM start")

	_, err := c.Post(ctx, fmt.Sprintf("/vm/id/%d/start", id), map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("starting VM %d: %w", id, err)
	}
	tflog.Trace(ctx, "StartVM success")
	return nil
}

// StopVM powers off a VM.
func (c *Client) StopVM(ctx context.Context, id int, force bool) error {
	tflog.Trace(ctx, "StopVM start")

	body := map[string]interface{}{
		"force":               force,
		"force_after_timeout": force,
	}
	_, err := c.Post(ctx, fmt.Sprintf("/vm/id/%d/stop", id), body)
	if err != nil {
		return fmt.Errorf("stopping VM %d: %w", id, err)
	}
	tflog.Trace(ctx, "StopVM success")
	return nil
}

// --- VM Device API ---

// VMDevice represents a device attached to a VM (DISK, NIC, CDROM, DISPLAY, RAW, PCI, USB).
type VMDevice struct {
	ID         int                    `json:"id"`
	VM         int                    `json:"vm"`
	Order      *int                   `json:"order"`
	Attributes map[string]interface{} `json:"attributes"`
}

// VMDeviceCreateRequest represents the request body for creating a VM device.
type VMDeviceCreateRequest struct {
	VM         int                    `json:"vm"`
	Order      *int                   `json:"order,omitempty"`
	Attributes map[string]interface{} `json:"attributes"`
}

// VMDeviceUpdateRequest represents the request body for updating a VM device.
type VMDeviceUpdateRequest struct {
	VM         *int                   `json:"vm,omitempty"`
	Order      *int                   `json:"order,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// GetVMDevice retrieves a VM device by its numeric ID.
func (c *Client) GetVMDevice(ctx context.Context, id int) (*VMDevice, error) {
	tflog.Trace(ctx, "GetVMDevice start")

	resp, err := c.Get(ctx, fmt.Sprintf("/vm/device/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting VM device %d: %w", id, err)
	}

	var dev VMDevice
	if err := json.Unmarshal(resp, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device response: %w", err)
	}
	tflog.Trace(ctx, "GetVMDevice success")
	return &dev, nil
}

// CreateVMDevice creates a new VM device.
func (c *Client) CreateVMDevice(ctx context.Context, req *VMDeviceCreateRequest) (*VMDevice, error) {
	tflog.Trace(ctx, "CreateVMDevice start")

	resp, err := c.Post(ctx, "/vm/device", req)
	if err != nil {
		return nil, fmt.Errorf("creating VM device on VM %d: %w", req.VM, err)
	}

	var dev VMDevice
	if err := json.Unmarshal(resp, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device create response: %w", err)
	}
	tflog.Trace(ctx, "CreateVMDevice success")
	return &dev, nil
}

// UpdateVMDevice updates an existing VM device.
func (c *Client) UpdateVMDevice(ctx context.Context, id int, req *VMDeviceUpdateRequest) (*VMDevice, error) {
	tflog.Trace(ctx, "UpdateVMDevice start")

	resp, err := c.Put(ctx, fmt.Sprintf("/vm/device/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating VM device %d: %w", id, err)
	}

	var dev VMDevice
	if err := json.Unmarshal(resp, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateVMDevice success")
	return &dev, nil
}

// DeleteVMDevice deletes a VM device.
func (c *Client) DeleteVMDevice(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteVMDevice start")

	_, err := c.Delete(ctx, fmt.Sprintf("/vm/device/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting VM device %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteVMDevice success")
	return nil
}
