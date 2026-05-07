package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for rsync tasks: rsynctask.{...}.

// GetRsyncTask retrieves an rsync task by ID.
func (c *Client) GetRsyncTask(ctx context.Context, id int) (*types.RsyncTask, error) {
	tflog.Trace(ctx, "GetRsyncTask (ws) start")

	result, err := c.Call(ctx, "rsynctask.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting rsync task %d: %w", id, err)
	}

	var task types.RsyncTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task response: %w", err)
	}

	tflog.Trace(ctx, "GetRsyncTask (ws) success")
	return &task, nil
}

// CreateRsyncTask creates a new rsync task.
func (c *Client) CreateRsyncTask(ctx context.Context, req *types.RsyncTaskCreateRequest) (*types.RsyncTask, error) {
	tflog.Trace(ctx, "CreateRsyncTask (ws) start")

	result, err := c.Call(ctx, "rsynctask.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating rsync task: %w", err)
	}

	var task types.RsyncTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateRsyncTask (ws) success")
	return &task, nil
}

// UpdateRsyncTask updates an existing rsync task.
func (c *Client) UpdateRsyncTask(ctx context.Context, id int, req *types.RsyncTaskUpdateRequest) (*types.RsyncTask, error) {
	tflog.Trace(ctx, "UpdateRsyncTask (ws) start")

	result, err := c.Call(ctx, "rsynctask.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating rsync task %d: %w", id, err)
	}

	var task types.RsyncTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateRsyncTask (ws) success")
	return &task, nil
}

// DeleteRsyncTask deletes an rsync task.
func (c *Client) DeleteRsyncTask(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteRsyncTask (ws) start")

	if _, err := c.Call(ctx, "rsynctask.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting rsync task %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteRsyncTask (ws) success")
	return nil
}
