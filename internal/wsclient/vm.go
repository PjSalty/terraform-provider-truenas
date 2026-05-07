package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespaces:
//   vm.{query,get_instance,create,update,delete,start,stop}
//   vm.device.{get_instance,create,update,delete}

// ListVMs retrieves all VMs.
func (c *Client) ListVMs(ctx context.Context) ([]types.VM, error) {
	tflog.Trace(ctx, "ListVMs (ws) start")

	result, err := c.Call(ctx, "vm.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing VMs: %w", err)
	}

	var vms []types.VM
	if err := json.Unmarshal(result, &vms); err != nil {
		return nil, fmt.Errorf("parsing VM list response: %w", err)
	}
	tflog.Trace(ctx, "ListVMs (ws) success")
	return vms, nil
}

// GetVM retrieves a VM by its numeric ID.
func (c *Client) GetVM(ctx context.Context, id int) (*types.VM, error) {
	tflog.Trace(ctx, "GetVM (ws) start")

	result, err := c.Call(ctx, "vm.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting VM %d: %w", id, err)
	}

	var vm types.VM
	if err := json.Unmarshal(result, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM response: %w", err)
	}
	tflog.Trace(ctx, "GetVM (ws) success")
	return &vm, nil
}

// CreateVM creates a new VM.
func (c *Client) CreateVM(ctx context.Context, req *types.VMCreateRequest) (*types.VM, error) {
	tflog.Trace(ctx, "CreateVM (ws) start")

	result, err := c.Call(ctx, "vm.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating VM %q: %w", req.Name, err)
	}

	var vm types.VM
	if err := json.Unmarshal(result, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM create response: %w", err)
	}
	tflog.Trace(ctx, "CreateVM (ws) success")
	return &vm, nil
}

// UpdateVM updates an existing VM.
func (c *Client) UpdateVM(ctx context.Context, id int, req *types.VMUpdateRequest) (*types.VM, error) {
	tflog.Trace(ctx, "UpdateVM (ws) start")

	result, err := c.Call(ctx, "vm.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating VM %d: %w", id, err)
	}

	var vm types.VM
	if err := json.Unmarshal(result, &vm); err != nil {
		return nil, fmt.Errorf("parsing VM update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateVM (ws) success")
	return &vm, nil
}

// DeleteVM deletes a VM. Pass options to control whether associated zvols
// are removed and whether a running VM is forcibly stopped.
func (c *Client) DeleteVM(ctx context.Context, id int, opts *types.VMDeleteOptions) error {
	tflog.Trace(ctx, "DeleteVM (ws) start")

	if opts == nil {
		opts = &types.VMDeleteOptions{Force: true, Zvols: false}
	}
	if _, err := c.Call(ctx, "vm.delete",
		[]interface{}{id, opts}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting VM %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteVM (ws) success")
	return nil
}

// StartVM powers on a VM.
func (c *Client) StartVM(ctx context.Context, id int) error {
	tflog.Trace(ctx, "StartVM (ws) start")

	if _, err := c.Call(ctx, "vm.start",
		[]interface{}{id, map[string]interface{}{}}, CallOptions{}); err != nil {
		return fmt.Errorf("starting VM %d: %w", id, err)
	}
	tflog.Trace(ctx, "StartVM (ws) success")
	return nil
}

// StopVM powers off a VM.
func (c *Client) StopVM(ctx context.Context, id int, force bool) error {
	tflog.Trace(ctx, "StopVM (ws) start")

	body := map[string]interface{}{
		"force":               force,
		"force_after_timeout": force,
	}
	if _, err := c.Call(ctx, "vm.stop",
		[]interface{}{id, body}, CallOptions{}); err != nil {
		return fmt.Errorf("stopping VM %d: %w", id, err)
	}
	tflog.Trace(ctx, "StopVM (ws) success")
	return nil
}

// --- VM Device API ---

// GetVMDevice retrieves a VM device by its numeric ID.
func (c *Client) GetVMDevice(ctx context.Context, id int) (*types.VMDevice, error) {
	tflog.Trace(ctx, "GetVMDevice (ws) start")

	result, err := c.Call(ctx, "vm.device.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting VM device %d: %w", id, err)
	}

	var dev types.VMDevice
	if err := json.Unmarshal(result, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device response: %w", err)
	}
	tflog.Trace(ctx, "GetVMDevice (ws) success")
	return &dev, nil
}

// CreateVMDevice creates a new VM device.
func (c *Client) CreateVMDevice(ctx context.Context, req *types.VMDeviceCreateRequest) (*types.VMDevice, error) {
	tflog.Trace(ctx, "CreateVMDevice (ws) start")

	result, err := c.Call(ctx, "vm.device.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating VM device on VM %d: %w", req.VM, err)
	}

	var dev types.VMDevice
	if err := json.Unmarshal(result, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device create response: %w", err)
	}
	tflog.Trace(ctx, "CreateVMDevice (ws) success")
	return &dev, nil
}

// UpdateVMDevice updates an existing VM device.
func (c *Client) UpdateVMDevice(ctx context.Context, id int, req *types.VMDeviceUpdateRequest) (*types.VMDevice, error) {
	tflog.Trace(ctx, "UpdateVMDevice (ws) start")

	result, err := c.Call(ctx, "vm.device.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating VM device %d: %w", id, err)
	}

	var dev types.VMDevice
	if err := json.Unmarshal(result, &dev); err != nil {
		return nil, fmt.Errorf("parsing VM device update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateVMDevice (ws) success")
	return &dev, nil
}

// DeleteVMDevice deletes a VM device.
func (c *Client) DeleteVMDevice(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteVMDevice (ws) start")

	if _, err := c.Call(ctx, "vm.device.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting VM device %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteVMDevice (ws) success")
	return nil
}
