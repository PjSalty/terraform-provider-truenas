package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Zvol API (uses dataset endpoint with type=VOLUME) ---

// ZvolCreateRequest represents the request to create a zvol.
type ZvolCreateRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Volsize       int64  `json:"volsize"`
	Volblocksize  string `json:"volblocksize,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Compression   string `json:"compression,omitempty"`
	Comments      string `json:"comments,omitempty"`
}

// ZvolUpdateRequest represents the request to update a zvol.
type ZvolUpdateRequest struct {
	Volsize       int64  `json:"volsize,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Compression   string `json:"compression,omitempty"`
	Comments      string `json:"comments,omitempty"`
}

// CreateZvol creates a new ZFS volume (zvol).
func (c *Client) CreateZvol(ctx context.Context, req *ZvolCreateRequest) (*DatasetResponse, error) {
	tflog.Trace(ctx, "CreateZvol start")

	req.Type = "VOLUME"
	resp, err := c.Post(ctx, "/pool/dataset", req)
	if err != nil {
		return nil, fmt.Errorf("creating zvol %q: %w", req.Name, err)
	}

	var dataset DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing zvol create response: %w", err)
	}

	tflog.Trace(ctx, "CreateZvol success")
	return &dataset, nil
}

// GetZvol retrieves a zvol by its full path (pool/name).
func (c *Client) GetZvol(ctx context.Context, id string) (*DatasetResponse, error) {
	tflog.Trace(ctx, "GetZvol start")

	tflog.Trace(ctx, "GetZvol success")
	return c.GetDataset(ctx, id)
}

// UpdateZvol updates an existing zvol.
func (c *Client) UpdateZvol(ctx context.Context, id string, req *ZvolUpdateRequest) (*DatasetResponse, error) {
	tflog.Trace(ctx, "UpdateZvol start")

	encodedID := url.PathEscape(id)
	resp, err := c.Put(ctx, "/pool/dataset/id/"+encodedID, req)
	if err != nil {
		return nil, fmt.Errorf("updating zvol %q: %w", id, err)
	}

	var dataset DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing zvol update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateZvol success")
	return &dataset, nil
}

// DeleteZvol deletes a zvol.
func (c *Client) DeleteZvol(ctx context.Context, id string) error {
	tflog.Trace(ctx, "DeleteZvol start")

	tflog.Trace(ctx, "DeleteZvol success")
	return c.DeleteDataset(ctx, id)
}
