package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for NFS shares: sharing.nfs.{...}.

// GetNFSShare retrieves an NFS share by ID.
func (c *Client) GetNFSShare(ctx context.Context, id int) (*types.NFSShare, error) {
	tflog.Trace(ctx, "GetNFSShare (ws) start")

	result, err := c.Call(ctx, "sharing.nfs.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting NFS share %d: %w", id, err)
	}

	var share types.NFSShare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share response: %w", err)
	}

	tflog.Trace(ctx, "GetNFSShare (ws) success")
	return &share, nil
}

// ListNFSShares retrieves all NFS shares.
func (c *Client) ListNFSShares(ctx context.Context) ([]types.NFSShare, error) {
	tflog.Trace(ctx, "ListNFSShares (ws) start")

	result, err := c.Call(ctx, "sharing.nfs.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing NFS shares: %w", err)
	}

	var shares []types.NFSShare
	if err := json.Unmarshal(result, &shares); err != nil {
		return nil, fmt.Errorf("parsing NFS shares list: %w", err)
	}

	tflog.Trace(ctx, "ListNFSShares (ws) success")
	return shares, nil
}

// CreateNFSShare creates a new NFS share.
func (c *Client) CreateNFSShare(ctx context.Context, req *types.NFSShareCreateRequest) (*types.NFSShare, error) {
	tflog.Trace(ctx, "CreateNFSShare (ws) start")

	result, err := c.Call(ctx, "sharing.nfs.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating NFS share: %w", err)
	}

	var share types.NFSShare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share create response: %w", err)
	}

	tflog.Trace(ctx, "CreateNFSShare (ws) success")
	return &share, nil
}

// UpdateNFSShare updates an existing NFS share.
func (c *Client) UpdateNFSShare(ctx context.Context, id int, req *types.NFSShareUpdateRequest) (*types.NFSShare, error) {
	tflog.Trace(ctx, "UpdateNFSShare (ws) start")

	result, err := c.Call(ctx, "sharing.nfs.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating NFS share %d: %w", id, err)
	}

	var share types.NFSShare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNFSShare (ws) success")
	return &share, nil
}

// DeleteNFSShare deletes an NFS share by ID.
func (c *Client) DeleteNFSShare(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNFSShare (ws) start")

	if _, err := c.Call(ctx, "sharing.nfs.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting NFS share %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteNFSShare (ws) success")
	return nil
}
