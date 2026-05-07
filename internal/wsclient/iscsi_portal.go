package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for iSCSI portals: iscsi.portal.{...}.

// GetISCSIPortal retrieves an iSCSI portal by ID.
func (c *Client) GetISCSIPortal(ctx context.Context, id int) (*types.ISCSIPortal, error) {
	tflog.Trace(ctx, "GetISCSIPortal (ws) start")

	result, err := c.Call(ctx, "iscsi.portal.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI portal %d: %w", id, err)
	}

	var portal types.ISCSIPortal
	if err := json.Unmarshal(result, &portal); err != nil {
		return nil, fmt.Errorf("parsing iSCSI portal response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIPortal (ws) success")
	return &portal, nil
}

// CreateISCSIPortal creates a new iSCSI portal.
func (c *Client) CreateISCSIPortal(ctx context.Context, req *types.ISCSIPortalCreateRequest) (*types.ISCSIPortal, error) {
	tflog.Trace(ctx, "CreateISCSIPortal (ws) start")

	result, err := c.Call(ctx, "iscsi.portal.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI portal: %w", err)
	}

	var portal types.ISCSIPortal
	if err := json.Unmarshal(result, &portal); err != nil {
		return nil, fmt.Errorf("parsing iSCSI portal create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIPortal (ws) success")
	return &portal, nil
}

// UpdateISCSIPortal updates an existing iSCSI portal.
func (c *Client) UpdateISCSIPortal(ctx context.Context, id int, req *types.ISCSIPortalUpdateRequest) (*types.ISCSIPortal, error) {
	tflog.Trace(ctx, "UpdateISCSIPortal (ws) start")

	result, err := c.Call(ctx, "iscsi.portal.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI portal %d: %w", id, err)
	}

	var portal types.ISCSIPortal
	if err := json.Unmarshal(result, &portal); err != nil {
		return nil, fmt.Errorf("parsing iSCSI portal update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIPortal (ws) success")
	return &portal, nil
}

// DeleteISCSIPortal deletes an iSCSI portal.
func (c *Client) DeleteISCSIPortal(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIPortal (ws) start")

	if _, err := c.Call(ctx, "iscsi.portal.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting iSCSI portal %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteISCSIPortal (ws) success")
	return nil
}
