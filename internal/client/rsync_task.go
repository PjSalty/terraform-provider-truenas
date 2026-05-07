package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// RsyncTask, RsyncTaskCreateRequest, RsyncTaskUpdateRequest moved to
// internal/types/rsync_task.go in the v2.0 transport-migration prep.
type (
	RsyncTask              = types.RsyncTask
	RsyncTaskCreateRequest = types.RsyncTaskCreateRequest
	RsyncTaskUpdateRequest = types.RsyncTaskUpdateRequest
)

// GetRsyncTask retrieves an rsync task by ID.
func (c *Client) GetRsyncTask(ctx context.Context, id int) (*types.RsyncTask, error) {
	tflog.Trace(ctx, "GetRsyncTask start")

	resp, err := c.Get(ctx, fmt.Sprintf("/rsynctask/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting rsync task %d: %w", id, err)
	}

	var task types.RsyncTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task response: %w", err)
	}

	tflog.Trace(ctx, "GetRsyncTask success")
	return &task, nil
}

// CreateRsyncTask creates a new rsync task.
func (c *Client) CreateRsyncTask(ctx context.Context, req *types.RsyncTaskCreateRequest) (*types.RsyncTask, error) {
	tflog.Trace(ctx, "CreateRsyncTask start")

	resp, err := c.Post(ctx, "/rsynctask", req)
	if err != nil {
		return nil, fmt.Errorf("creating rsync task: %w", err)
	}

	var task types.RsyncTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateRsyncTask success")
	return &task, nil
}

// UpdateRsyncTask updates an existing rsync task.
func (c *Client) UpdateRsyncTask(ctx context.Context, id int, req *types.RsyncTaskUpdateRequest) (*types.RsyncTask, error) {
	tflog.Trace(ctx, "UpdateRsyncTask start")

	resp, err := c.Put(ctx, fmt.Sprintf("/rsynctask/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating rsync task %d: %w", id, err)
	}

	var task types.RsyncTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateRsyncTask success")
	return &task, nil
}

// DeleteRsyncTask deletes an rsync task.
func (c *Client) DeleteRsyncTask(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteRsyncTask start")

	_, err := c.Delete(ctx, fmt.Sprintf("/rsynctask/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting rsync task %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteRsyncTask success")
	return nil
}
