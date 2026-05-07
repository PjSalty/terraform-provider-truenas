package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// NFSShare, NFSShareCreateRequest, NFSShareUpdateRequest moved to
// internal/types/nfs_share.go in the v2.0 transport-migration prep.
type (
	NFSShare              = types.NFSShare
	NFSShareCreateRequest = types.NFSShareCreateRequest
	NFSShareUpdateRequest = types.NFSShareUpdateRequest
)

// GetNFSShare retrieves an NFS share by ID.
func (c *Client) GetNFSShare(ctx context.Context, id int) (*types.NFSShare, error) {
	tflog.Trace(ctx, "GetNFSShare start")

	resp, err := c.Get(ctx, fmt.Sprintf("/sharing/nfs/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting NFS share %d: %w", id, err)
	}

	var share types.NFSShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share response: %w", err)
	}

	tflog.Trace(ctx, "GetNFSShare success")
	return &share, nil
}

// ListNFSShares retrieves all NFS shares.
func (c *Client) ListNFSShares(ctx context.Context) ([]types.NFSShare, error) {
	tflog.Trace(ctx, "ListNFSShares start")

	resp, err := c.Get(ctx, "/sharing/nfs")
	if err != nil {
		return nil, fmt.Errorf("listing NFS shares: %w", err)
	}

	var shares []types.NFSShare
	if err := json.Unmarshal(resp, &shares); err != nil {
		return nil, fmt.Errorf("parsing NFS shares list: %w", err)
	}

	tflog.Trace(ctx, "ListNFSShares success")
	return shares, nil
}

// CreateNFSShare creates a new NFS share.
func (c *Client) CreateNFSShare(ctx context.Context, req *types.NFSShareCreateRequest) (*types.NFSShare, error) {
	tflog.Trace(ctx, "CreateNFSShare start")

	resp, err := c.Post(ctx, "/sharing/nfs", req)
	if err != nil {
		return nil, fmt.Errorf("creating NFS share: %w", err)
	}

	var share types.NFSShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share create response: %w", err)
	}

	tflog.Trace(ctx, "CreateNFSShare success")
	return &share, nil
}

// UpdateNFSShare updates an existing NFS share.
func (c *Client) UpdateNFSShare(ctx context.Context, id int, req *types.NFSShareUpdateRequest) (*types.NFSShare, error) {
	tflog.Trace(ctx, "UpdateNFSShare start")

	resp, err := c.Put(ctx, fmt.Sprintf("/sharing/nfs/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating NFS share %d: %w", id, err)
	}

	var share types.NFSShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNFSShare success")
	return &share, nil
}

// DeleteNFSShare deletes an NFS share.
func (c *Client) DeleteNFSShare(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNFSShare start")

	_, err := c.Delete(ctx, fmt.Sprintf("/sharing/nfs/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting NFS share %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNFSShare success")
	return nil
}
