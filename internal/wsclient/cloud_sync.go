package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for cloud sync tasks: cloudsync.{...}.

// GetCloudSync retrieves a cloud sync task by ID.
func (c *Client) GetCloudSync(ctx context.Context, id int) (*types.CloudSync, error) {
	tflog.Trace(ctx, "GetCloudSync (ws) start")

	result, err := c.Call(ctx, "cloudsync.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting cloud sync %d: %w", id, err)
	}

	var cs types.CloudSync
	if err := json.Unmarshal(result, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudSync (ws) success")
	return &cs, nil
}

// CreateCloudSync creates a new cloud sync task.
func (c *Client) CreateCloudSync(ctx context.Context, req *types.CloudSyncCreateRequest) (*types.CloudSync, error) {
	tflog.Trace(ctx, "CreateCloudSync (ws) start")

	result, err := c.Call(ctx, "cloudsync.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating cloud sync: %w", err)
	}

	var cs types.CloudSync
	if err := json.Unmarshal(result, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudSync (ws) success")
	return &cs, nil
}

// UpdateCloudSync updates an existing cloud sync task.
func (c *Client) UpdateCloudSync(ctx context.Context, id int, req *types.CloudSyncUpdateRequest) (*types.CloudSync, error) {
	tflog.Trace(ctx, "UpdateCloudSync (ws) start")

	result, err := c.Call(ctx, "cloudsync.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating cloud sync %d: %w", id, err)
	}

	var cs types.CloudSync
	if err := json.Unmarshal(result, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudSync (ws) success")
	return &cs, nil
}

// DeleteCloudSync deletes a cloud sync task.
func (c *Client) DeleteCloudSync(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudSync (ws) start")

	if _, err := c.Call(ctx, "cloudsync.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting cloud sync %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteCloudSync (ws) success")
	return nil
}
