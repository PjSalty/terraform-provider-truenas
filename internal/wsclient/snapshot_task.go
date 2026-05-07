package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for snapshot tasks: pool.snapshottask.{...}.

// GetSnapshotTask retrieves a snapshot task by ID.
func (c *Client) GetSnapshotTask(ctx context.Context, id int) (*types.SnapshotTask, error) {
	tflog.Trace(ctx, "GetSnapshotTask (ws) start")

	result, err := c.Call(ctx, "pool.snapshottask.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting snapshot task %d: %w", id, err)
	}

	var task types.SnapshotTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task response: %w", err)
	}

	tflog.Trace(ctx, "GetSnapshotTask (ws) success")
	return &task, nil
}

// CreateSnapshotTask creates a new snapshot task.
func (c *Client) CreateSnapshotTask(ctx context.Context, req *types.SnapshotTaskCreateRequest) (*types.SnapshotTask, error) {
	tflog.Trace(ctx, "CreateSnapshotTask (ws) start")

	result, err := c.Call(ctx, "pool.snapshottask.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating snapshot task: %w", err)
	}

	var task types.SnapshotTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateSnapshotTask (ws) success")
	return &task, nil
}

// UpdateSnapshotTask updates an existing snapshot task.
func (c *Client) UpdateSnapshotTask(ctx context.Context, id int, req *types.SnapshotTaskUpdateRequest) (*types.SnapshotTask, error) {
	tflog.Trace(ctx, "UpdateSnapshotTask (ws) start")

	result, err := c.Call(ctx, "pool.snapshottask.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating snapshot task %d: %w", id, err)
	}

	var task types.SnapshotTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSnapshotTask (ws) success")
	return &task, nil
}

// DeleteSnapshotTask deletes a snapshot task.
func (c *Client) DeleteSnapshotTask(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteSnapshotTask (ws) start")

	if _, err := c.Call(ctx, "pool.snapshottask.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting snapshot task %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteSnapshotTask (ws) success")
	return nil
}
