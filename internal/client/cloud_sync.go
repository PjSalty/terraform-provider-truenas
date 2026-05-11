package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Cloud Sync API ---

// CloudSync represents a cloud sync task in TrueNAS.
type CloudSync struct {
	ID           int                    `json:"id"`
	Description  string                 `json:"description,omitempty"`
	Path         string                 `json:"path"`
	Credentials  int                    `json:"credentials"`
	Direction    string                 `json:"direction"`
	TransferMode string                 `json:"transfer_mode"`
	Schedule     Schedule               `json:"schedule"`
	Enabled      bool                   `json:"enabled"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

// CloudSyncCreateRequest represents the request to create a cloud sync task.
type CloudSyncCreateRequest struct {
	Description  string                 `json:"description,omitempty"`
	Path         string                 `json:"path"`
	Credentials  int                    `json:"credentials"`
	Direction    string                 `json:"direction"`
	TransferMode string                 `json:"transfer_mode"`
	Schedule     Schedule               `json:"schedule,omitempty"`
	Enabled      bool                   `json:"enabled"`
	Attributes   map[string]interface{} `json:"attributes"`
}

// CloudSyncUpdateRequest represents the request to update a cloud sync task.
type CloudSyncUpdateRequest struct {
	Description  string                 `json:"description,omitempty"`
	Path         string                 `json:"path,omitempty"`
	Credentials  int                    `json:"credentials,omitempty"`
	Direction    string                 `json:"direction,omitempty"`
	TransferMode string                 `json:"transfer_mode,omitempty"`
	Schedule     *Schedule              `json:"schedule,omitempty"`
	Enabled      *bool                  `json:"enabled,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

// GetCloudSync retrieves a cloud sync task by ID.
func (c *Client) GetCloudSync(ctx context.Context, id int) (*CloudSync, error) {
	tflog.Trace(ctx, "GetCloudSync start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cloudsync/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cloud sync %d: %w", id, err)
	}

	var cs CloudSync
	if err := json.Unmarshal(resp, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudSync success")
	return &cs, nil
}

// CreateCloudSync creates a new cloud sync task.
func (c *Client) CreateCloudSync(ctx context.Context, req *CloudSyncCreateRequest) (*CloudSync, error) {
	tflog.Trace(ctx, "CreateCloudSync start")

	resp, err := c.Post(ctx, "/cloudsync", req)
	if err != nil {
		return nil, fmt.Errorf("creating cloud sync: %w", err)
	}

	var cs CloudSync
	if err := json.Unmarshal(resp, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudSync success")
	return &cs, nil
}

// UpdateCloudSync updates an existing cloud sync task.
func (c *Client) UpdateCloudSync(ctx context.Context, id int, req *CloudSyncUpdateRequest) (*CloudSync, error) {
	tflog.Trace(ctx, "UpdateCloudSync start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cloudsync/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cloud sync %d: %w", id, err)
	}

	var cs CloudSync
	if err := json.Unmarshal(resp, &cs); err != nil {
		return nil, fmt.Errorf("parsing cloud sync update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudSync success")
	return &cs, nil
}

// DeleteCloudSync deletes a cloud sync task.
func (c *Client) DeleteCloudSync(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudSync start")

	_, err := c.Delete(ctx, fmt.Sprintf("/cloudsync/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting cloud sync %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteCloudSync success")
	return nil
}
