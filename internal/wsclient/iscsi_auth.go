package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for iSCSI CHAP auth: iscsi.auth.{...}.

// GetISCSIAuth retrieves an iSCSI auth entry by ID.
func (c *Client) GetISCSIAuth(ctx context.Context, id int) (*types.ISCSIAuth, error) {
	tflog.Trace(ctx, "GetISCSIAuth (ws) start")

	result, err := c.Call(ctx, "iscsi.auth.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI auth %d: %w", id, err)
	}

	var a types.ISCSIAuth
	if err := json.Unmarshal(result, &a); err != nil {
		return nil, fmt.Errorf("parsing iSCSI auth response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIAuth (ws) success")
	return &a, nil
}

// CreateISCSIAuth creates an iSCSI auth entry.
func (c *Client) CreateISCSIAuth(ctx context.Context, req *types.ISCSIAuthCreateRequest) (*types.ISCSIAuth, error) {
	tflog.Trace(ctx, "CreateISCSIAuth (ws) start")

	result, err := c.Call(ctx, "iscsi.auth.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI auth: %w", err)
	}

	var a types.ISCSIAuth
	if err := json.Unmarshal(result, &a); err != nil {
		return nil, fmt.Errorf("parsing iSCSI auth create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIAuth (ws) success")
	return &a, nil
}

// UpdateISCSIAuth updates an iSCSI auth entry by ID.
func (c *Client) UpdateISCSIAuth(ctx context.Context, id int, req *types.ISCSIAuthUpdateRequest) (*types.ISCSIAuth, error) {
	tflog.Trace(ctx, "UpdateISCSIAuth (ws) start")

	result, err := c.Call(ctx, "iscsi.auth.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI auth %d: %w", id, err)
	}

	var a types.ISCSIAuth
	if err := json.Unmarshal(result, &a); err != nil {
		return nil, fmt.Errorf("parsing iSCSI auth update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIAuth (ws) success")
	return &a, nil
}

// DeleteISCSIAuth deletes an iSCSI auth entry.
func (c *Client) DeleteISCSIAuth(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIAuth (ws) start")

	if _, err := c.Call(ctx, "iscsi.auth.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting iSCSI auth %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteISCSIAuth (ws) success")
	return nil
}
