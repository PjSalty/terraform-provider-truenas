package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for pool scrub tasks: pool.scrub.{...}.

// GetScrubTask retrieves a scrub task by ID.
func (c *Client) GetScrubTask(ctx context.Context, id int) (*types.ScrubTask, error) {
	tflog.Trace(ctx, "GetScrubTask (ws) start")

	result, err := c.Call(ctx, "pool.scrub.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting scrub task %d: %w", id, err)
	}

	var task types.ScrubTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task response: %w", err)
	}

	tflog.Trace(ctx, "GetScrubTask (ws) success")
	return &task, nil
}

// CreateScrubTask creates a new scrub task.
func (c *Client) CreateScrubTask(ctx context.Context, req *types.ScrubTaskCreateRequest) (*types.ScrubTask, error) {
	tflog.Trace(ctx, "CreateScrubTask (ws) start")

	result, err := c.Call(ctx, "pool.scrub.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating scrub task: %w", err)
	}

	var task types.ScrubTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateScrubTask (ws) success")
	return &task, nil
}

// UpdateScrubTask updates an existing scrub task.
func (c *Client) UpdateScrubTask(ctx context.Context, id int, req *types.ScrubTaskUpdateRequest) (*types.ScrubTask, error) {
	tflog.Trace(ctx, "UpdateScrubTask (ws) start")

	result, err := c.Call(ctx, "pool.scrub.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating scrub task %d: %w", id, err)
	}

	var task types.ScrubTask
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateScrubTask (ws) success")
	return &task, nil
}

// DeleteScrubTask deletes a scrub task.
func (c *Client) DeleteScrubTask(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteScrubTask (ws) start")

	if _, err := c.Call(ctx, "pool.scrub.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting scrub task %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteScrubTask (ws) success")
	return nil
}
