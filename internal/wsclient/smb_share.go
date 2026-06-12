package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for SMB shares: sharing.smb.{...}.

// ListSMBShares retrieves all SMB shares.
func (c *Client) ListSMBShares(ctx context.Context) ([]types.SMBShare, error) {
	tflog.Trace(ctx, "ListSMBShares (ws) start")

	result, err := c.Call(ctx, "sharing.smb.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing SMB shares: %w", err)
	}

	var shares []types.SMBShare
	if err := json.Unmarshal(result, &shares); err != nil {
		return nil, fmt.Errorf("parsing SMB shares list: %w", err)
	}

	tflog.Trace(ctx, "ListSMBShares (ws) success")
	return shares, nil
}

// GetSMBShare retrieves an SMB share by ID.
func (c *Client) GetSMBShare(ctx context.Context, id int) (*types.SMBShare, error) {
	tflog.Trace(ctx, "GetSMBShare (ws) start")

	result, err := c.Call(ctx, "sharing.smb.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting SMB share %d: %w", id, err)
	}

	var share types.SMBShare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing SMB share response: %w", err)
	}

	tflog.Trace(ctx, "GetSMBShare (ws) success")
	return &share, nil
}

// CreateSMBShare creates a new SMB share.
func (c *Client) CreateSMBShare(ctx context.Context, req *types.SMBShareCreateRequest) (*types.SMBShare, error) {
	tflog.Trace(ctx, "CreateSMBShare (ws) start")

	result, err := c.Call(ctx, "sharing.smb.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating SMB share: %w", err)
	}

	var share types.SMBShare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing SMB share create response: %w", err)
	}

	tflog.Trace(ctx, "CreateSMBShare (ws) success")
	return &share, nil
}

// UpdateSMBShare updates an existing SMB share.
func (c *Client) UpdateSMBShare(ctx context.Context, id int, req *types.SMBShareUpdateRequest) (*types.SMBShare, error) {
	tflog.Trace(ctx, "UpdateSMBShare (ws) start")

	result, err := c.Call(ctx, "sharing.smb.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating SMB share %d: %w", id, err)
	}

	var share types.SMBShare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing SMB share update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSMBShare (ws) success")
	return &share, nil
}

// DeleteSMBShare deletes an SMB share by ID.
func (c *Client) DeleteSMBShare(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteSMBShare (ws) start")

	if _, err := c.Call(ctx, "sharing.smb.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting SMB share %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteSMBShare (ws) success")
	return nil
}
