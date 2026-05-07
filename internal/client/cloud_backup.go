package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// CloudBackupSchedule, CloudBackup, CloudBackupCreateRequest,
// CloudBackupUpdateRequest moved to internal/types/cloud_backup.go in
// the v2.0 transport-migration prep.
type (
	CloudBackupSchedule      = types.CloudBackupSchedule
	CloudBackup              = types.CloudBackup
	CloudBackupCreateRequest = types.CloudBackupCreateRequest
	CloudBackupUpdateRequest = types.CloudBackupUpdateRequest
)

// GetCloudBackup retrieves a cloud backup task by ID.
func (c *Client) GetCloudBackup(ctx context.Context, id int) (*types.CloudBackup, error) {
	tflog.Trace(ctx, "GetCloudBackup start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cloud_backup/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cloud backup %d: %w", id, err)
	}

	var cb types.CloudBackup
	if err := json.Unmarshal(resp, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudBackup success")
	return &cb, nil
}

// CreateCloudBackup creates a cloud backup task.
func (c *Client) CreateCloudBackup(ctx context.Context, req *types.CloudBackupCreateRequest) (*types.CloudBackup, error) {
	tflog.Trace(ctx, "CreateCloudBackup start")

	resp, err := c.Post(ctx, "/cloud_backup", req)
	if err != nil {
		return nil, fmt.Errorf("creating cloud backup: %w", err)
	}

	var cb types.CloudBackup
	if err := json.Unmarshal(resp, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudBackup success")
	return &cb, nil
}

// UpdateCloudBackup updates an existing cloud backup task.
func (c *Client) UpdateCloudBackup(ctx context.Context, id int, req *types.CloudBackupUpdateRequest) (*types.CloudBackup, error) {
	tflog.Trace(ctx, "UpdateCloudBackup start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cloud_backup/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cloud backup %d: %w", id, err)
	}

	var cb types.CloudBackup
	if err := json.Unmarshal(resp, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudBackup success")
	return &cb, nil
}

// DeleteCloudBackup deletes a cloud backup task.
func (c *Client) DeleteCloudBackup(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudBackup start")

	_, err := c.Delete(ctx, fmt.Sprintf("/cloud_backup/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting cloud backup %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteCloudBackup success")
	return nil
}
