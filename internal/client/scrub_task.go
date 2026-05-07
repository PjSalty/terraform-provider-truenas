package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ScrubTask, ScrubTaskCreateRequest, ScrubTaskUpdateRequest moved to
// internal/types/scrub_task.go in the v2.0 transport-migration prep.
type (
	ScrubTask              = types.ScrubTask
	ScrubTaskCreateRequest = types.ScrubTaskCreateRequest
	ScrubTaskUpdateRequest = types.ScrubTaskUpdateRequest
)

// GetScrubTask retrieves a scrub task by ID.
func (c *Client) GetScrubTask(ctx context.Context, id int) (*types.ScrubTask, error) {
	tflog.Trace(ctx, "GetScrubTask start")

	resp, err := c.Get(ctx, fmt.Sprintf("/pool/scrub/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting scrub task %d: %w", id, err)
	}

	var task types.ScrubTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task response: %w", err)
	}

	tflog.Trace(ctx, "GetScrubTask success")
	return &task, nil
}

// CreateScrubTask creates a new scrub task.
func (c *Client) CreateScrubTask(ctx context.Context, req *types.ScrubTaskCreateRequest) (*types.ScrubTask, error) {
	tflog.Trace(ctx, "CreateScrubTask start")

	resp, err := c.Post(ctx, "/pool/scrub", req)
	if err != nil {
		return nil, fmt.Errorf("creating scrub task: %w", err)
	}

	var task types.ScrubTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateScrubTask success")
	return &task, nil
}

// UpdateScrubTask updates an existing scrub task.
func (c *Client) UpdateScrubTask(ctx context.Context, id int, req *types.ScrubTaskUpdateRequest) (*types.ScrubTask, error) {
	tflog.Trace(ctx, "UpdateScrubTask start")

	resp, err := c.Put(ctx, fmt.Sprintf("/pool/scrub/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating scrub task %d: %w", id, err)
	}

	var task types.ScrubTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateScrubTask success")
	return &task, nil
}

// DeleteScrubTask deletes a scrub task.
func (c *Client) DeleteScrubTask(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteScrubTask start")

	_, err := c.Delete(ctx, fmt.Sprintf("/pool/scrub/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting scrub task %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteScrubTask success")
	return nil
}
