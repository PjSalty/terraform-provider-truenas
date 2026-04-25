package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Pool Scrub Task API ---

// ScrubTask represents a ZFS pool scrub task.
type ScrubTask struct {
	ID          int      `json:"id"`
	Pool        int      `json:"pool"`
	PoolName    string   `json:"pool_name"`
	Threshold   int      `json:"threshold"`
	Description string   `json:"description"`
	Schedule    Schedule `json:"schedule"`
	Enabled     bool     `json:"enabled"`
}

// ScrubTaskCreateRequest represents the request to create a scrub task.
type ScrubTaskCreateRequest struct {
	Pool        int      `json:"pool"`
	Threshold   int      `json:"threshold"`
	Description string   `json:"description,omitempty"`
	Schedule    Schedule `json:"schedule"`
	Enabled     bool     `json:"enabled"`
}

// ScrubTaskUpdateRequest represents the request to update a scrub task.
type ScrubTaskUpdateRequest struct {
	Pool        int       `json:"pool,omitempty"`
	Threshold   int       `json:"threshold,omitempty"`
	Description string    `json:"description,omitempty"`
	Schedule    *Schedule `json:"schedule,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
}

// GetScrubTask retrieves a scrub task by ID.
func (c *Client) GetScrubTask(ctx context.Context, id int) (*ScrubTask, error) {
	tflog.Trace(ctx, "GetScrubTask start")

	resp, err := c.Get(ctx, fmt.Sprintf("/pool/scrub/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting scrub task %d: %w", id, err)
	}

	var task ScrubTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task response: %w", err)
	}

	tflog.Trace(ctx, "GetScrubTask success")
	return &task, nil
}

// CreateScrubTask creates a new scrub task.
func (c *Client) CreateScrubTask(ctx context.Context, req *ScrubTaskCreateRequest) (*ScrubTask, error) {
	tflog.Trace(ctx, "CreateScrubTask start")

	resp, err := c.Post(ctx, "/pool/scrub", req)
	if err != nil {
		return nil, fmt.Errorf("creating scrub task: %w", err)
	}

	var task ScrubTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing scrub task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateScrubTask success")
	return &task, nil
}

// UpdateScrubTask updates an existing scrub task.
func (c *Client) UpdateScrubTask(ctx context.Context, id int, req *ScrubTaskUpdateRequest) (*ScrubTask, error) {
	tflog.Trace(ctx, "UpdateScrubTask start")

	resp, err := c.Put(ctx, fmt.Sprintf("/pool/scrub/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating scrub task %d: %w", id, err)
	}

	var task ScrubTask
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
