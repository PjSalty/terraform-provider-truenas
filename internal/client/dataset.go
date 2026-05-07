package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// Dataset, DatasetCreateRequest, DatasetUpdateRequest, and DatasetResponse
// moved to internal/types/dataset.go in the v2.0 transport-migration prep.
type (
	Dataset              = types.Dataset
	DatasetCreateRequest = types.DatasetCreateRequest
	DatasetUpdateRequest = types.DatasetUpdateRequest
	DatasetResponse      = types.DatasetResponse
)

// GetDataset retrieves a dataset by its ID (full path like "tank/dataset").
func (c *Client) GetDataset(ctx context.Context, id string) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "GetDataset start")

	encodedID := url.PathEscape(id)
	resp, err := c.Get(ctx, "/pool/dataset/id/"+encodedID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset %q: %w", id, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset response: %w", err)
	}

	tflog.Trace(ctx, "GetDataset success")
	return &dataset, nil
}

// ListDatasets retrieves all datasets. The optional pool filter limits
// results to datasets belonging to the named pool; the optional parent
// filter limits results to datasets whose ID has the given prefix followed
// by a '/'. Filtering is performed client-side.
func (c *Client) ListDatasets(ctx context.Context) ([]types.DatasetResponse, error) {
	tflog.Trace(ctx, "ListDatasets start")

	resp, err := c.Get(ctx, "/pool/dataset")
	if err != nil {
		return nil, fmt.Errorf("listing datasets: %w", err)
	}

	var datasets []types.DatasetResponse
	if err := json.Unmarshal(resp, &datasets); err != nil {
		return nil, fmt.Errorf("parsing datasets list response: %w", err)
	}
	tflog.Trace(ctx, "ListDatasets success")
	return datasets, nil
}

// CreateDataset creates a new ZFS dataset.
func (c *Client) CreateDataset(ctx context.Context, req *types.DatasetCreateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "CreateDataset start")

	resp, err := c.Post(ctx, "/pool/dataset", req)
	if err != nil {
		return nil, fmt.Errorf("creating dataset %q: %w", req.Name, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset create response: %w", err)
	}

	tflog.Trace(ctx, "CreateDataset success")
	return &dataset, nil
}

// UpdateDataset updates an existing ZFS dataset.
func (c *Client) UpdateDataset(ctx context.Context, id string, req *types.DatasetUpdateRequest) (*types.DatasetResponse, error) {
	tflog.Trace(ctx, "UpdateDataset start")

	encodedID := url.PathEscape(id)
	resp, err := c.Put(ctx, "/pool/dataset/id/"+encodedID, req)
	if err != nil {
		return nil, fmt.Errorf("updating dataset %q: %w", id, err)
	}

	var dataset types.DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateDataset success")
	return &dataset, nil
}

// DeleteDataset deletes a ZFS dataset.
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	tflog.Trace(ctx, "DeleteDataset start")

	encodedID := url.PathEscape(id)
	_, err := c.Delete(ctx, "/pool/dataset/id/"+encodedID)
	if err != nil {
		return fmt.Errorf("deleting dataset %q: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteDataset success")
	return nil
}
