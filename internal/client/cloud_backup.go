package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CloudBackupSchedule represents a cron-style schedule for cloud backup tasks.
type CloudBackupSchedule struct {
	Minute string `json:"minute,omitempty"`
	Hour   string `json:"hour,omitempty"`
	Dom    string `json:"dom,omitempty"`
	Month  string `json:"month,omitempty"`
	Dow    string `json:"dow,omitempty"`
}

// CloudBackup represents a cloud backup task. Fields that are polymorphic
// (attributes, credentials) are kept as raw JSON so the resource layer can
// round-trip them without losing information.
type CloudBackup struct {
	ID              int                 `json:"id"`
	Description     string              `json:"description"`
	Path            string              `json:"path"`
	Credentials     json.RawMessage     `json:"credentials,omitempty"`
	Attributes      json.RawMessage     `json:"attributes,omitempty"`
	Schedule        CloudBackupSchedule `json:"schedule"`
	PreScript       string              `json:"pre_script"`
	PostScript      string              `json:"post_script"`
	Snapshot        bool                `json:"snapshot"`
	Include         []string            `json:"include"`
	Exclude         []string            `json:"exclude"`
	Args            string              `json:"args"`
	Enabled         bool                `json:"enabled"`
	Password        string              `json:"password"`
	KeepLast        int                 `json:"keep_last"`
	TransferSetting string              `json:"transfer_setting"`
}

// CloudBackupCreateRequest represents a request to create a cloud backup task.
type CloudBackupCreateRequest struct {
	Description     string               `json:"description,omitempty"`
	Path            string               `json:"path"`
	Credentials     int                  `json:"credentials"`
	Attributes      json.RawMessage      `json:"attributes"`
	Schedule        *CloudBackupSchedule `json:"schedule,omitempty"`
	PreScript       string               `json:"pre_script,omitempty"`
	PostScript      string               `json:"post_script,omitempty"`
	Snapshot        *bool                `json:"snapshot,omitempty"`
	Include         []string             `json:"include,omitempty"`
	Exclude         []string             `json:"exclude,omitempty"`
	Args            string               `json:"args,omitempty"`
	Enabled         *bool                `json:"enabled,omitempty"`
	Password        string               `json:"password"`
	KeepLast        int                  `json:"keep_last"`
	TransferSetting string               `json:"transfer_setting,omitempty"`
}

// CloudBackupUpdateRequest represents a request to update a cloud backup task.
type CloudBackupUpdateRequest struct {
	Description     *string              `json:"description,omitempty"`
	Path            *string              `json:"path,omitempty"`
	Credentials     *int                 `json:"credentials,omitempty"`
	Attributes      json.RawMessage      `json:"attributes,omitempty"`
	Schedule        *CloudBackupSchedule `json:"schedule,omitempty"`
	PreScript       *string              `json:"pre_script,omitempty"`
	PostScript      *string              `json:"post_script,omitempty"`
	Snapshot        *bool                `json:"snapshot,omitempty"`
	Include         *[]string            `json:"include,omitempty"`
	Exclude         *[]string            `json:"exclude,omitempty"`
	Args            *string              `json:"args,omitempty"`
	Enabled         *bool                `json:"enabled,omitempty"`
	Password        *string              `json:"password,omitempty"`
	KeepLast        *int                 `json:"keep_last,omitempty"`
	TransferSetting *string              `json:"transfer_setting,omitempty"`
}

// GetCloudBackup retrieves a cloud backup task by ID.
func (c *Client) GetCloudBackup(ctx context.Context, id int) (*CloudBackup, error) {
	tflog.Trace(ctx, "GetCloudBackup start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cloud_backup/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cloud backup %d: %w", id, err)
	}

	var cb CloudBackup
	if err := json.Unmarshal(resp, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudBackup success")
	return &cb, nil
}

// CreateCloudBackup creates a cloud backup task.
func (c *Client) CreateCloudBackup(ctx context.Context, req *CloudBackupCreateRequest) (*CloudBackup, error) {
	tflog.Trace(ctx, "CreateCloudBackup start")

	resp, err := c.Post(ctx, "/cloud_backup", req)
	if err != nil {
		return nil, fmt.Errorf("creating cloud backup: %w", err)
	}

	var cb CloudBackup
	if err := json.Unmarshal(resp, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudBackup success")
	return &cb, nil
}

// UpdateCloudBackup updates an existing cloud backup task.
func (c *Client) UpdateCloudBackup(ctx context.Context, id int, req *CloudBackupUpdateRequest) (*CloudBackup, error) {
	tflog.Trace(ctx, "UpdateCloudBackup start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cloud_backup/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cloud backup %d: %w", id, err)
	}

	var cb CloudBackup
	if err := json.Unmarshal(resp, &cb); err != nil {
		return nil, fmt.Errorf("parsing cloud backup update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudBackup success")
	return &cb, nil
}

// DeleteCloudBackup deletes a cloud backup task.
func (c *Client) DeleteCloudBackup(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudBackup start")

	_, err := c.Delete(ctx, fmt.Sprintf("/cloud_backup/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting cloud backup %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteCloudBackup success")
	return nil
}
