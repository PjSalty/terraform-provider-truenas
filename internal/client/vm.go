package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// VM, VMStatus, VMCreateRequest, VMUpdateRequest, VMDeleteOptions, VMDevice,
// VMDeviceCreateRequest, VMDeviceUpdateRequest moved to internal/types/vm.go
// in the v2.0 transport-migration prep.
type (
	VM                    = types.VM
	VMStatus              = types.VMStatus
	VMCreateRequest       = types.VMCreateRequest
	VMUpdateRequest       = types.VMUpdateRequest
	VMDeleteOptions       = types.VMDeleteOptions
	VMDevice              = types.VMDevice
	VMDeviceCreateRequest = types.VMDeviceCreateRequest
	VMDeviceUpdateRequest = types.VMDeviceUpdateRequest
)

// ListVMs retrieves all VMs.
func (c *Client) ListVMs(ctx context.Context) ([]types.VM, error) {
	tflog.Trace(ctx, "ListVMs start")

	resp, err := c.Get(ctx, "/vm")
	if err != nil {
		return nil, fmt.Errorf("listing VMs: %w", err)
	}

	var vms []types.VM
	if err := json.Unmarshal(resp, &vms); err != nil {
		return nil, fmt.Errorf("parsing VM list response: %w", err)
	}
	tflog.Trace(ctx, "ListVMs success")
	return vms, nil
}

// GetVM retrieves a VM by its numeric ID.
func (c *Client) GetVM(ctx context.Context, id int) (*types.VM, error) {
	tflog.Trace(ctx, "GetVM start")

	resp, err := c.Get(ctx, fmt.Sprintf("/vm/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting VM %d: %w", id, err)
	}

	var vm types.VM
	if err := json.Unmarshal(resp, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM response: %w", err)
	}
	tflog.Trace(ctx, "GetVM success")
	return &vm, nil
}

// CreateVM creates a new VM.
func (c *Client) CreateVM(ctx context.Context, req *types.VMCreateRequest) (*types.VM, error) {
	tflog.Trace(ctx, "CreateVM start")

	resp, err := c.Post(ctx, "/vm", req)
	if err != nil {
		return nil, fmt.Errorf("creating VM %q: %w", req.Name, err)
	}

	var vm types.VM
	if err := json.Unmarshal(resp, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM create response: %w", err)
	}
	tflog.Trace(ctx, "CreateVM success")
	return &vm, nil
}

// UpdateVM updates an existing VM.
func (c *Client) UpdateVM(ctx context.Context, id int, req *types.VMUpdateRequest) (*types.VM, error) {
	tflog.Trace(ctx, "UpdateVM start")

	resp, err := c.Put(ctx, fmt.Sprintf("/vm/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating VM %d: %w", id, err)
	}

	var vm types.VM
	if err := json.Unmarshal(resp, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateVM success")
	return &vm, nil
}

// DeleteVM deletes a VM. The body controls whether associated zvols are also removed
// and whether a running VM is forcibly stopped.
func (c *Client) DeleteVM(ctx context.Context, id int, opts *types.VMDeleteOptions) error {
	tflog.Trace(ctx, "DeleteVM start")

	if opts == nil {
		opts = &types.VMDeleteOptions{Force: true, Zvols: false}
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

// GetVMDevice retrieves a VM device by its numeric ID.
func (c *Client) GetVMDevice(ctx context.Context, id int) (*types.VMDevice, error) {
	tflog.Trace(ctx, "GetVMDevice start")

	resp, err := c.Get(ctx, fmt.Sprintf("/vm/device/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting VM device %d: %w", id, err)
	}

	var dev types.VMDevice
	if err := json.Unmarshal(resp, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device response: %w", err)
	}
	tflog.Trace(ctx, "GetVMDevice success")
	return &dev, nil
}

// CreateVMDevice creates a new VM device.
func (c *Client) CreateVMDevice(ctx context.Context, req *types.VMDeviceCreateRequest) (*types.VMDevice, error) {
	tflog.Trace(ctx, "CreateVMDevice start")

	resp, err := c.Post(ctx, "/vm/device", req)
	if err != nil {
		return nil, fmt.Errorf("creating VM device on VM %d: %w", req.VM, err)
	}

	var dev types.VMDevice
	if err := json.Unmarshal(resp, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device create response: %w", err)
	}
	tflog.Trace(ctx, "CreateVMDevice success")
	return &dev, nil
}

// UpdateVMDevice updates an existing VM device.
func (c *Client) UpdateVMDevice(ctx context.Context, id int, req *types.VMDeviceUpdateRequest) (*types.VMDevice, error) {
	tflog.Trace(ctx, "UpdateVMDevice start")

	resp, err := c.Put(ctx, fmt.Sprintf("/vm/device/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating VM device %d: %w", id, err)
	}

	var dev types.VMDevice
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
