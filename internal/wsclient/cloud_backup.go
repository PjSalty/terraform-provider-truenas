package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for cloud backup tasks: cloud_backup.{...}.

// GetCloudBackup retrieves a cloud backup task by ID.
func (c *Client) GetCloudBackup(ctx context.Context, id int) (*types.CloudBackup, error) {
	tflog.Trace(ctx, "GetCloudBackup (ws) start")

	result, err := c.Call(ctx, "cloud_backup.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting cloud backup %d: %w", id, err)
	}

	var cb types.CloudBackup
	if err := json.Unmarshal(result, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudBackup (ws) success")
	return &cb, nil
}

// CreateCloudBackup creates a cloud backup task.
func (c *Client) CreateCloudBackup(ctx context.Context, req *types.CloudBackupCreateRequest) (*types.CloudBackup, error) {
	tflog.Trace(ctx, "CreateCloudBackup (ws) start")

	result, err := c.Call(ctx, "cloud_backup.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating cloud backup: %w", err)
	}

	var cb types.CloudBackup
	if err := json.Unmarshal(result, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudBackup (ws) success")
	return &cb, nil
}

// UpdateCloudBackup updates an existing cloud backup task.
func (c *Client) UpdateCloudBackup(ctx context.Context, id int, req *types.CloudBackupUpdateRequest) (*types.CloudBackup, error) {
	tflog.Trace(ctx, "UpdateCloudBackup (ws) start")

	result, err := c.Call(ctx, "cloud_backup.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating cloud backup %d: %w", id, err)
	}

	var cb types.CloudBackup
	if err := json.Unmarshal(result, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudBackup (ws) success")
	return &cb, nil
}

// DeleteCloudBackup deletes a cloud backup task.
func (c *Client) DeleteCloudBackup(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudBackup (ws) start")

	if _, err := c.Call(ctx, "cloud_backup.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting cloud backup %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteCloudBackup (ws) success")
	return nil
}
