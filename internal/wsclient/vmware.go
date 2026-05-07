package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for vmware: vmware.{...}.

// GetVMware retrieves a VMware integration by ID.
func (c *Client) GetVMware(ctx context.Context, id int) (*types.VMware, error) {
	tflog.Trace(ctx, "GetVMware (ws) start")

	result, err := c.Call(ctx, "vmware.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting VMware %d: %w", id, err)
	}

	var v types.VMware
	if err := json.Unmarshal(result, &v); err != nil {
		return nil, fmt.Errorf("parsing VMware response: %w", err)
	}

	tflog.Trace(ctx, "GetVMware (ws) success")
	return &v, nil
}

// CreateVMware creates a new VMware integration.
func (c *Client) CreateVMware(ctx context.Context, req *types.VMwareCreateRequest) (*types.VMware, error) {
	tflog.Trace(ctx, "CreateVMware (ws) start")

	result, err := c.Call(ctx, "vmware.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating VMware: %w", err)
	}

	var v types.VMware
	if err := json.Unmarshal(result, &v); err != nil {
		return nil, fmt.Errorf("parsing VMware create response: %w", err)
	}

	tflog.Trace(ctx, "CreateVMware (ws) success")
	return &v, nil
}

// UpdateVMware updates an existing VMware integration.
func (c *Client) UpdateVMware(ctx context.Context, id int, req *types.VMwareUpdateRequest) (*types.VMware, error) {
	tflog.Trace(ctx, "UpdateVMware (ws) start")

	result, err := c.Call(ctx, "vmware.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating VMware %d: %w", id, err)
	}

	var v types.VMware
	if err := json.Unmarshal(result, &v); err != nil {
		return nil, fmt.Errorf("parsing VMware update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateVMware (ws) success")
	return &v, nil
}

// DeleteVMware deletes a VMware integration.
func (c *Client) DeleteVMware(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteVMware (ws) start")

	if _, err := c.Call(ctx, "vmware.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting VMware %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteVMware (ws) success")
	return nil
}
