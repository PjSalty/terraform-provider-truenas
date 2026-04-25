package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- NFS Share API ---

// NFSShare represents an NFS share in TrueNAS.
type NFSShare struct {
	ID           int      `json:"id"`
	Path         string   `json:"path"`
	Aliases      []string `json:"aliases,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	ReadOnly     bool     `json:"ro"`
	MaprootUser  string   `json:"maproot_user,omitempty"`
	MaprootGroup string   `json:"maproot_group,omitempty"`
	MapallUser   string   `json:"mapall_user,omitempty"`
	MapallGroup  string   `json:"mapall_group,omitempty"`
	Security     []string `json:"security,omitempty"`
	Enabled      bool     `json:"enabled"`
	Networks     []string `json:"networks,omitempty"`
}

// NFSShareCreateRequest represents the request to create an NFS share.
type NFSShareCreateRequest struct {
	Path         string   `json:"path"`
	Aliases      []string `json:"aliases,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	ReadOnly     bool     `json:"ro"`
	MaprootUser  string   `json:"maproot_user,omitempty"`
	MaprootGroup string   `json:"maproot_group,omitempty"`
	MapallUser   string   `json:"mapall_user,omitempty"`
	MapallGroup  string   `json:"mapall_group,omitempty"`
	Security     []string `json:"security,omitempty"`
	Enabled      bool     `json:"enabled"`
	Networks     []string `json:"networks,omitempty"`
}

// NFSShareUpdateRequest represents the request to update an NFS share.
type NFSShareUpdateRequest struct {
	Path         string   `json:"path,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	ReadOnly     *bool    `json:"ro,omitempty"`
	MaprootUser  string   `json:"maproot_user,omitempty"`
	MaprootGroup string   `json:"maproot_group,omitempty"`
	MapallUser   string   `json:"mapall_user,omitempty"`
	MapallGroup  string   `json:"mapall_group,omitempty"`
	Security     []string `json:"security,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
	Networks     []string `json:"networks,omitempty"`
}

// GetNFSShare retrieves an NFS share by ID.
func (c *Client) GetNFSShare(ctx context.Context, id int) (*NFSShare, error) {
	tflog.Trace(ctx, "GetNFSShare start")

	resp, err := c.Get(ctx, fmt.Sprintf("/sharing/nfs/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting NFS share %d: %w", id, err)
	}

	var share NFSShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share response: %w", err)
	}

	tflog.Trace(ctx, "GetNFSShare success")
	return &share, nil
}

// ListNFSShares retrieves all NFS shares.
func (c *Client) ListNFSShares(ctx context.Context) ([]NFSShare, error) {
	tflog.Trace(ctx, "ListNFSShares start")

	resp, err := c.Get(ctx, "/sharing/nfs")
	if err != nil {
		return nil, fmt.Errorf("listing NFS shares: %w", err)
	}

	var shares []NFSShare
	if err := json.Unmarshal(resp, &shares); err != nil {
		return nil, fmt.Errorf("parsing NFS shares list: %w", err)
	}

	tflog.Trace(ctx, "ListNFSShares success")
	return shares, nil
}

// CreateNFSShare creates a new NFS share.
func (c *Client) CreateNFSShare(ctx context.Context, req *NFSShareCreateRequest) (*NFSShare, error) {
	tflog.Trace(ctx, "CreateNFSShare start")

	resp, err := c.Post(ctx, "/sharing/nfs", req)
	if err != nil {
		return nil, fmt.Errorf("creating NFS share: %w", err)
	}

	var share NFSShare
	if err := json.Unmarshal(resp, &share); err != nil {
		return nil, fmt.Errorf("parsing NFS share create response: %w", err)
	}

	tflog.Trace(ctx, "CreateNFSShare success")
	return &share, nil
}

// UpdateNFSShare updates an existing NFS share.
func (c *Client) UpdateNFSShare(ctx context.Context, id int, req *NFSShareUpdateRequest) (*NFSShare, error) {
	tflog.Trace(ctx, "UpdateNFSShare start")

	resp, err := c.Put(ctx, fmt.Sprintf("/sharing/nfs/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating NFS share %d: %w", id, err)
	}

	var share NFSShare
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
