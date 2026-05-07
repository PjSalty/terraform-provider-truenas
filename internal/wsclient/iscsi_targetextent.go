package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for iSCSI target-extent associations:
// iscsi.targetextent.{...}.

// GetISCSITargetExtent retrieves an iSCSI target-extent association by ID.
func (c *Client) GetISCSITargetExtent(ctx context.Context, id int) (*types.ISCSITargetExtent, error) {
	tflog.Trace(ctx, "GetISCSITargetExtent (ws) start")

	result, err := c.Call(ctx, "iscsi.targetextent.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI target-extent %d: %w", id, err)
	}

	var te types.ISCSITargetExtent
	if err := json.Unmarshal(result, &te); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target-extent response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSITargetExtent (ws) success")
	return &te, nil
}

// CreateISCSITargetExtent creates a new iSCSI target-extent association.
func (c *Client) CreateISCSITargetExtent(ctx context.Context, req *types.ISCSITargetExtentCreateRequest) (*types.ISCSITargetExtent, error) {
	tflog.Trace(ctx, "CreateISCSITargetExtent (ws) start")

	result, err := c.Call(ctx, "iscsi.targetextent.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI target-extent: %w", err)
	}

	var te types.ISCSITargetExtent
	if err := json.Unmarshal(result, &te); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target-extent create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSITargetExtent (ws) success")
	return &te, nil
}

// UpdateISCSITargetExtent updates an existing iSCSI target-extent association.
func (c *Client) UpdateISCSITargetExtent(ctx context.Context, id int, req *types.ISCSITargetExtentUpdateRequest) (*types.ISCSITargetExtent, error) {
	tflog.Trace(ctx, "UpdateISCSITargetExtent (ws) start")

	result, err := c.Call(ctx, "iscsi.targetextent.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI target-extent %d: %w", id, err)
	}

	var te types.ISCSITargetExtent
	if err := json.Unmarshal(result, &te); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target-extent update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSITargetExtent (ws) success")
	return &te, nil
}

// DeleteISCSITargetExtent deletes an iSCSI target-extent association.
func (c *Client) DeleteISCSITargetExtent(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSITargetExtent (ws) start")

	if _, err := c.Call(ctx, "iscsi.targetextent.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting iSCSI target-extent %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteISCSITargetExtent (ws) success")
	return nil
}
