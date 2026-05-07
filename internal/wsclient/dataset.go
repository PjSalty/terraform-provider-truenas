package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for datasets: pool.dataset.{...}.

// GetDataset retrieves a dataset by its ID (full path like "tank/dataset").
// Implemented via pool.dataset.query with the id filter so we get the
// expanded property objects identical to REST GET /pool/dataset/id/X.
func (c *Client) GetDataset(ctx context.Context, id string) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "GetDataset (ws) start")

	result, err := c.Call(ctx, "pool.dataset.query",
		[]interface{}{
			[]interface{}{[]interface{}{"id", "=", id}},
			map[string]interface{}{"get": true},
		},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting dataset %q: %w", id, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(result, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset response: %w", err)
	}

	tflog.Trace(ctx, "GetDataset (ws) success")
	return &dataset, nil
}

// ListDatasets retrieves all datasets.
func (c *Client) ListDatasets(ctx context.Context) ([]types.DatasetResponse, error) {
	tflog.Trace(ctx, "ListDatasets (ws) start")

	result, err := c.Call(ctx, "pool.dataset.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing datasets: %w", err)
	}

	var datasets []types.DatasetResponse
	if err := json.Unmarshal(result, &datasets); err != nil {
		return nil, fmt.Errorf("parsing datasets list response: %w", err)
	}

	tflog.Trace(ctx, "ListDatasets (ws) success")
	return datasets, nil
}

// CreateDataset creates a new ZFS dataset.
func (c *Client) CreateDataset(ctx context.Context, req *types.DatasetCreateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "CreateDataset (ws) start")

	result, err := c.Call(ctx, "pool.dataset.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating dataset %q: %w", req.Name, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(result, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset create response: %w", err)
	}

	tflog.Trace(ctx, "CreateDataset (ws) success")
	return &dataset, nil
}

// UpdateDataset updates an existing ZFS dataset.
func (c *Client) UpdateDataset(ctx context.Context, id string, req *types.DatasetUpdateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "UpdateDataset (ws) start")

	result, err := c.Call(ctx, "pool.dataset.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating dataset %q: %w", id, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(result, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateDataset (ws) success")
	return &dataset, nil
}

// DeleteDataset deletes a ZFS dataset.
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	tflog.Trace(ctx, "DeleteDataset (ws) start")

	if _, err := c.Call(ctx, "pool.dataset.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting dataset %q: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteDataset (ws) success")
	return nil
}
