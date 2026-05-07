package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// CloudSync, CloudSyncCreateRequest, CloudSyncUpdateRequest moved to
// internal/types/cloud_sync.go in the v2.0 transport-migration prep.
type (
	CloudSync              = types.CloudSync
	CloudSyncCreateRequest = types.CloudSyncCreateRequest
	CloudSyncUpdateRequest = types.CloudSyncUpdateRequest
)

// GetCloudSync retrieves a cloud sync task by ID.
func (c *Client) GetCloudSync(ctx context.Context, id int) (*types.CloudSync, error) {
	tflog.Trace(ctx, "GetCloudSync start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cloudsync/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cloud sync %d: %w", id, err)
	}

	var cs types.CloudSync
	if err := json.Unmarshal(resp, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudSync success")
	return &cs, nil
}

// CreateCloudSync creates a new cloud sync task.
func (c *Client) CreateCloudSync(ctx context.Context, req *types.CloudSyncCreateRequest) (*types.CloudSync, error) {
	tflog.Trace(ctx, "CreateCloudSync start")

	resp, err := c.Post(ctx, "/cloudsync", req)
	if err != nil {
		return nil, fmt.Errorf("creating cloud sync: %w", err)
	}

	var cs types.CloudSync
	if err := json.Unmarshal(resp, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudSync success")
	return &cs, nil
}

// UpdateCloudSync updates an existing cloud sync task.
func (c *Client) UpdateCloudSync(ctx context.Context, id int, req *types.CloudSyncUpdateRequest) (*types.CloudSync, error) {
	tflog.Trace(ctx, "UpdateCloudSync start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cloudsync/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cloud sync %d: %w", id, err)
	}

	var cs types.CloudSync
	if err := json.Unmarshal(resp, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudSync success")
	return &cs, nil
}

// DeleteCloudSync deletes a cloud sync task.
func (c *Client) DeleteCloudSync(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudSync start")

	_, err := c.Delete(ctx, fmt.Sprintf("/cloudsync/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting cloud sync %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteCloudSync success")
	return nil
}
