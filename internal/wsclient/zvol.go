package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// Zvols are managed via the same pool.dataset namespace with type=VOLUME,
// matching the REST shape. GetZvol/DeleteZvol delegate directly to the
// dataset methods.

// CreateZvol creates a new ZFS volume (zvol).
func (c *Client) CreateZvol(ctx context.Context, req *types.ZvolCreateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "CreateZvol (ws) start")

	req.Type = "VOLUME"
	result, err := c.Call(ctx, "pool.dataset.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating zvol %q: %w", req.Name, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(result, &dataset); err != nil {
		return nil, fmt.Errorf("parsing zvol create response: %w", err)
	}

	tflog.Trace(ctx, "CreateZvol (ws) success")
	return &dataset, nil
}

// GetZvol retrieves a zvol by its full path (pool/name).
func (c *Client) GetZvol(ctx context.Context, id string) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "GetZvol (ws) start")
	tflog.Trace(ctx, "GetZvol (ws) success")
	return c.GetDataset(ctx, id)
}

// UpdateZvol updates an existing zvol.
func (c *Client) UpdateZvol(ctx context.Context, id string, req *types.ZvolUpdateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "UpdateZvol (ws) start")

	result, err := c.Call(ctx, "pool.dataset.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating zvol %q: %w", id, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(result, &dataset); err != nil {
		return nil, fmt.Errorf("parsing zvol update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateZvol (ws) success")
	return &dataset, nil
}

// DeleteZvol deletes a zvol.
func (c *Client) DeleteZvol(ctx context.Context, id string) error {
	tflog.Trace(ctx, "DeleteZvol (ws) start")
	tflog.Trace(ctx, "DeleteZvol (ws) success")
	return c.DeleteDataset(ctx, id)
}
