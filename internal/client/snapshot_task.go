package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Snapshot Task API ---

// SnapshotTask represents a periodic snapshot task.
type SnapshotTask struct {
	ID           int      `json:"id"`
	Dataset      string   `json:"dataset"`
	Recursive    bool     `json:"recursive"`
	Lifetime     int      `json:"lifetime_value"`
	LifetimeUnit string   `json:"lifetime_unit"`
	NamingSchema string   `json:"naming_schema"`
	Schedule     Schedule `json:"schedule"`
	Enabled      bool     `json:"enabled"`
	AllowEmpty   bool     `json:"allow_empty"`
	Exclude      []string `json:"exclude,omitempty"`
}

// SnapshotTaskCreateRequest represents the request to create a snapshot task.
type SnapshotTaskCreateRequest struct {
	Dataset      string   `json:"dataset"`
	Recursive    bool     `json:"recursive"`
	Lifetime     int      `json:"lifetime_value"`
	LifetimeUnit string   `json:"lifetime_unit"`
	NamingSchema string   `json:"naming_schema"`
	Schedule     Schedule `json:"schedule"`
	Enabled      bool     `json:"enabled"`
	AllowEmpty   bool     `json:"allow_empty"`
	Exclude      []string `json:"exclude,omitempty"`
}

// SnapshotTaskUpdateRequest represents the request to update a snapshot task.
type SnapshotTaskUpdateRequest struct {
	Dataset      string    `json:"dataset,omitempty"`
	Recursive    *bool     `json:"recursive,omitempty"`
	Lifetime     int       `json:"lifetime_value,omitempty"`
	LifetimeUnit string    `json:"lifetime_unit,omitempty"`
	NamingSchema string    `json:"naming_schema,omitempty"`
	Schedule     *Schedule `json:"schedule,omitempty"`
	Enabled      *bool     `json:"enabled,omitempty"`
	AllowEmpty   *bool     `json:"allow_empty,omitempty"`
	Exclude      []string  `json:"exclude,omitempty"`
}

// GetSnapshotTask retrieves a snapshot task by ID.
func (c *Client) GetSnapshotTask(ctx context.Context, id int) (*SnapshotTask, error) {
	tflog.Trace(ctx, "GetSnapshotTask start")

	resp, err := c.Get(ctx, fmt.Sprintf("/pool/snapshottask/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting snapshot task %d: %w", id, err)
	}

	var task SnapshotTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task response: %w", err)
	}

	tflog.Trace(ctx, "GetSnapshotTask success")
	return &task, nil
}

// CreateSnapshotTask creates a new snapshot task.
func (c *Client) CreateSnapshotTask(ctx context.Context, req *SnapshotTaskCreateRequest) (*SnapshotTask, error) {
	tflog.Trace(ctx, "CreateSnapshotTask start")

	resp, err := c.Post(ctx, "/pool/snapshottask", req)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot task: %w", err)
	}

	var task SnapshotTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing snapshot task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateSnapshotTask success")
	return &task, nil
}

// UpdateSnapshotTask updates an existing snapshot task.
func (c *Client) UpdateSnapshotTask(ctx context.Context, id int, req *SnapshotTaskUpdateRequest) (*SnapshotTask, error) {
	tflog.Trace(ctx, "UpdateSnapshotTask start")

	resp, err := c.Put(ctx, fmt.Sprintf("/pool/snapshottask/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating snapshot task %d: %w", id, err)
	}

	var task SnapshotTask
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
