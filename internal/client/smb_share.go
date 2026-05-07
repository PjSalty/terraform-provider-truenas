package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// SMBShare, SMBShareCreateRequest, SMBShareUpdateRequest moved to
// internal/types/smb_share.go in the v2.0 transport-migration prep.
type (
	SMBShare              = types.SMBShare
	SMBShareCreateRequest = types.SMBShareCreateRequest
	SMBShareUpdateRequest = types.SMBShareUpdateRequest
)

// ListSMBShares retrieves all SMB shares.
func (c *Client) ListSMBShares(ctx context.Context) ([]types.SMBShare, error) {
	tflog.Trace(ctx, "ListSMBShares start")

	resp, err := c.Get(ctx, "/sharing/smb")
	if err != nil {
		return nil, fmt.Errorf("listing SMB shares: %w", err)
	}

	var shares []types.SMBShare
	if err := json.Unmarshal(resp, &shares); err != nil {
		return nil, fmt.Errorf("parsing SMB shares list: %w", err)
	}

	tflog.Trace(ctx, "ListSMBShares success")
	return shares, nil
}

// GetSMBShare retrieves an SMB share by ID.
func (c *Client) GetSMBShare(ctx context.Context, id int) (*types.SMBShare, error) {
	tflog.Trace(ctx, "GetSMBShare start")

	resp, err := c.Get(ctx, fmt.Sprintf("/sharing/smb/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting SMB share %d: %w", id, err)
	}

	var share types.SMBShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing SMB share response: %w", err)
	}

	tflog.Trace(ctx, "GetSMBShare success")
	return &share, nil
}

// CreateSMBShare creates a new SMB share.
func (c *Client) CreateSMBShare(ctx context.Context, req *types.SMBShareCreateRequest) (*types.SMBShare, error) {
	tflog.Trace(ctx, "CreateSMBShare start")

	resp, err := c.Post(ctx, "/sharing/smb", req)
	if err != nil {
		return nil, fmt.Errorf("creating SMB share: %w", err)
	}

	var share types.SMBShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing SMB share create response: %w", err)
	}

	tflog.Trace(ctx, "CreateSMBShare success")
	return &share, nil
}

// UpdateSMBShare updates an existing SMB share.
func (c *Client) UpdateSMBShare(ctx context.Context, id int, req *types.SMBShareUpdateRequest) (*types.SMBShare, error) {
	tflog.Trace(ctx, "UpdateSMBShare start")

	resp, err := c.Put(ctx, fmt.Sprintf("/sharing/smb/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating SMB share %d: %w", id, err)
	}

	var share types.SMBShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing SMB share update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSMBShare success")
	return &share, nil
}

// DeleteSMBShare deletes an SMB share.
func (c *Client) DeleteSMBShare(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteSMBShare start")

	_, err := c.Delete(ctx, fmt.Sprintf("/sharing/smb/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting SMB share %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteSMBShare success")
	return nil
}
