package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for iSCSI targets: iscsi.target.{...}.

// GetISCSITarget retrieves an iSCSI target by ID.
func (c *Client) GetISCSITarget(ctx context.Context, id int) (*types.ISCSITarget, error) {
	tflog.Trace(ctx, "GetISCSITarget (ws) start")

	result, err := c.Call(ctx, "iscsi.target.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI target %d: %w", id, err)
	}

	var target types.ISCSITarget
	if err := json.Unmarshal(result, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSITarget (ws) success")
	return &target, nil
}

// CreateISCSITarget creates a new iSCSI target.
func (c *Client) CreateISCSITarget(ctx context.Context, req *types.ISCSITargetCreateRequest) (*types.ISCSITarget, error) {
	tflog.Trace(ctx, "CreateISCSITarget (ws) start")

	result, err := c.Call(ctx, "iscsi.target.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI target: %w", err)
	}

	var target types.ISCSITarget
	if err := json.Unmarshal(result, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSITarget (ws) success")
	return &target, nil
}

// UpdateISCSITarget updates an existing iSCSI target.
func (c *Client) UpdateISCSITarget(ctx context.Context, id int, req *types.ISCSITargetUpdateRequest) (*types.ISCSITarget, error) {
	tflog.Trace(ctx, "UpdateISCSITarget (ws) start")

	result, err := c.Call(ctx, "iscsi.target.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI target %d: %w", id, err)
	}

	var target types.ISCSITarget
	if err := json.Unmarshal(result, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSITarget (ws) success")
	return &target, nil
}

// DeleteISCSITarget deletes an iSCSI target.
func (c *Client) DeleteISCSITarget(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSITarget (ws) start")

	if _, err := c.Call(ctx, "iscsi.target.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting iSCSI target %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteISCSITarget (ws) success")
	return nil
}
