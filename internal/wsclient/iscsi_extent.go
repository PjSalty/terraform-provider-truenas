package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for iSCSI extents: iscsi.extent.{...}.

// GetISCSIExtent retrieves an iSCSI extent by ID.
func (c *Client) GetISCSIExtent(ctx context.Context, id int) (*types.ISCSIExtent, error) {
	tflog.Trace(ctx, "GetISCSIExtent (ws) start")

	result, err := c.Call(ctx, "iscsi.extent.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI extent %d: %w", id, err)
	}

	var extent types.ISCSIExtent
	if err := json.Unmarshal(result, &extent); err != nil {
		return nil, fmt.Errorf("parsing iSCSI extent response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIExtent (ws) success")
	return &extent, nil
}

// CreateISCSIExtent creates a new iSCSI extent.
func (c *Client) CreateISCSIExtent(ctx context.Context, req *types.ISCSIExtentCreateRequest) (*types.ISCSIExtent, error) {
	tflog.Trace(ctx, "CreateISCSIExtent (ws) start")

	result, err := c.Call(ctx, "iscsi.extent.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI extent: %w", err)
	}

	var extent types.ISCSIExtent
	if err := json.Unmarshal(result, &extent); err != nil {
		return nil, fmt.Errorf("parsing iSCSI extent create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIExtent (ws) success")
	return &extent, nil
}

// UpdateISCSIExtent updates an existing iSCSI extent.
func (c *Client) UpdateISCSIExtent(ctx context.Context, id int, req *types.ISCSIExtentUpdateRequest) (*types.ISCSIExtent, error) {
	tflog.Trace(ctx, "UpdateISCSIExtent (ws) start")

	result, err := c.Call(ctx, "iscsi.extent.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI extent %d: %w", id, err)
	}

	var extent types.ISCSIExtent
	if err := json.Unmarshal(result, &extent); err != nil {
		return nil, fmt.Errorf("parsing iSCSI extent update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIExtent (ws) success")
	return &extent, nil
}

// DeleteISCSIExtent deletes an iSCSI extent.
func (c *Client) DeleteISCSIExtent(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIExtent (ws) start")

	if _, err := c.Call(ctx, "iscsi.extent.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting iSCSI extent %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteISCSIExtent (ws) success")
	return nil
}
