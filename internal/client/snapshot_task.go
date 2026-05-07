package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// SnapshotTask, SnapshotTaskCreateRequest, SnapshotTaskUpdateRequest moved
// to internal/types/snapshot_task.go in the v2.0 transport-migration prep.
type (
	SnapshotTask              = types.SnapshotTask
	SnapshotTaskCreateRequest = types.SnapshotTaskCreateRequest
	SnapshotTaskUpdateRequest = types.SnapshotTaskUpdateRequest
)

// GetSnapshotTask retrieves a snapshot task by ID.
func (c *Client) GetSnapshotTask(ctx context.Context, id int) (*types.SnapshotTask, error) {
	tflog.Trace(ctx, "GetSnapshotTask start")

	resp, err := c.Get(ctx, fmt.Sprintf("/pool/snapshottask/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting snapshot task %d: %w", id, err)
	}

	var task types.SnapshotTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task response: %w", err)
	}

	tflog.Trace(ctx, "GetSnapshotTask success")
	return &task, nil
}

// CreateSnapshotTask creates a new snapshot task.
func (c *Client) CreateSnapshotTask(ctx context.Context, req *types.SnapshotTaskCreateRequest) (*types.SnapshotTask, error) {
	tflog.Trace(ctx, "CreateSnapshotTask start")

	resp, err := c.Post(ctx, "/pool/snapshottask", req)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot task: %w", err)
	}

	var task types.SnapshotTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateSnapshotTask success")
	return &task, nil
}

// UpdateSnapshotTask updates an existing snapshot task.
func (c *Client) UpdateSnapshotTask(ctx context.Context, id int, req *types.SnapshotTaskUpdateRequest) (*types.SnapshotTask, error) {
	tflog.Trace(ctx, "UpdateSnapshotTask start")

	resp, err := c.Put(ctx, fmt.Sprintf("/pool/snapshottask/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating snapshot task %d: %w", id, err)
	}

	var task types.SnapshotTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSnapshotTask success")
	return &task, nil
}

// DeleteSnapshotTask deletes a snapshot task.
func (c *Client) DeleteSnapshotTask(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteSnapshotTask start")

	_, err := c.Delete(ctx, fmt.Sprintf("/pool/snapshottask/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting snapshot task %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteSnapshotTask success")
	return nil
}
