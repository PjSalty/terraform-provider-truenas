package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ZvolCreateRequest, ZvolUpdateRequest moved to internal/types/zvol.go in
// the v2.0 transport-migration prep.
type (
	ZvolCreateRequest = types.ZvolCreateRequest
	ZvolUpdateRequest = types.ZvolUpdateRequest
)

// CreateZvol creates a new ZFS volume (zvol).
func (c *Client) CreateZvol(ctx context.Context, req *types.ZvolCreateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "CreateZvol start")

	req.Type = "VOLUME"
	resp, err := c.Post(ctx, "/pool/dataset", req)
	if err != nil {
		return nil, fmt.Errorf("creating zvol %q: %w", req.Name, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing zvol create response: %w", err)
	}

	tflog.Trace(ctx, "CreateZvol success")
	return &dataset, nil
}

// GetZvol retrieves a zvol by its full path (pool/name).
func (c *Client) GetZvol(ctx context.Context, id string) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "GetZvol start")

	tflog.Trace(ctx, "GetZvol success")
	return c.GetDataset(ctx, id)
}

// UpdateZvol updates an existing zvol.
func (c *Client) UpdateZvol(ctx context.Context, id string, req *types.ZvolUpdateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "UpdateZvol start")

	encodedID := url.PathEscape(id)
	resp, err := c.Put(ctx, "/pool/dataset/id/"+encodedID, req)
	if err != nil {
		return nil, fmt.Errorf("updating zvol %q: %w", id, err)
	}

	var dataset types.DatasetResponse
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
