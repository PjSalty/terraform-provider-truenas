package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Rsync Task API ---

// RsyncTask represents an rsync task in TrueNAS.
type RsyncTask struct {
	ID           int      `json:"id"`
	Path         string   `json:"path"`
	Remotehost   string   `json:"remotehost,omitempty"`
	Remoteport   int      `json:"remoteport,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	Remotemodule string   `json:"remotemodule,omitempty"`
	Remotepath   string   `json:"remotepath,omitempty"`
	Direction    string   `json:"direction,omitempty"`
	Schedule     Schedule `json:"schedule"`
	User         string   `json:"user"`
	Enabled      bool     `json:"enabled"`
	Desc         string   `json:"desc,omitempty"`
}

// RsyncTaskCreateRequest represents the request to create an rsync task.
type RsyncTaskCreateRequest struct {
	Path         string   `json:"path"`
	Remotehost   string   `json:"remotehost,omitempty"`
	Remoteport   int      `json:"remoteport,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	Remotemodule string   `json:"remotemodule,omitempty"`
	Remotepath   string   `json:"remotepath,omitempty"`
	Direction    string   `json:"direction,omitempty"`
	Schedule     Schedule `json:"schedule,omitempty"`
	User         string   `json:"user"`
	Enabled      bool     `json:"enabled"`
	Desc         string   `json:"desc,omitempty"`
}

// RsyncTaskUpdateRequest represents the request to update an rsync task.
type RsyncTaskUpdateRequest struct {
	Path         string    `json:"path,omitempty"`
	Remotehost   string    `json:"remotehost,omitempty"`
	Remoteport   int       `json:"remoteport,omitempty"`
	Mode         string    `json:"mode,omitempty"`
	Remotemodule string    `json:"remotemodule,omitempty"`
	Remotepath   string    `json:"remotepath,omitempty"`
	Direction    string    `json:"direction,omitempty"`
	Schedule     *Schedule `json:"schedule,omitempty"`
	User         string    `json:"user,omitempty"`
	Enabled      *bool     `json:"enabled,omitempty"`
	Desc         string    `json:"desc,omitempty"`
}

// GetRsyncTask retrieves an rsync task by ID.
func (c *Client) GetRsyncTask(ctx context.Context, id int) (*RsyncTask, error) {
	tflog.Trace(ctx, "GetRsyncTask start")

	resp, err := c.Get(ctx, fmt.Sprintf("/rsynctask/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting rsync task %d: %w", id, err)
	}

	var task RsyncTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task response: %w", err)
	}

	tflog.Trace(ctx, "GetRsyncTask success")
	return &task, nil
}

// CreateRsyncTask creates a new rsync task.
func (c *Client) CreateRsyncTask(ctx context.Context, req *RsyncTaskCreateRequest) (*RsyncTask, error) {
	tflog.Trace(ctx, "CreateRsyncTask start")

	resp, err := c.Post(ctx, "/rsynctask", req)
	if err != nil {
		return nil, fmt.Errorf("creating rsync task: %w", err)
	}

	var task RsyncTask
	if err := json.Unmarshal(resp, &task); err != nil {
		return nil, fmt.Errorf("parsing rsync task create response: %w", err)
	}

	tflog.Trace(ctx, "CreateRsyncTask success")
	return &task, nil
}

// UpdateRsyncTask updates an existing rsync task.
func (c *Client) UpdateRsyncTask(ctx context.Context, id int, req *RsyncTaskUpdateRequest) (*RsyncTask, error) {
	tflog.Trace(ctx, "UpdateRsyncTask start")

	resp, err := c.Put(ctx, fmt.Sprintf("/rsynctask/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating rsync task %d: %w", id, err)
	}

	var task RsyncTask
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
