package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Disk API (read-only) ---

// Disk represents a disk in TrueNAS.
type Disk struct {
	Identifier  string  `json:"identifier"`
	Name        string  `json:"name"`
	Subsystem   string  `json:"subsystem"`
	Number      int     `json:"number"`
	Serial      string  `json:"serial"`
	Size        int64   `json:"size"`
	Description string  `json:"description"`
	Model       string  `json:"model"`
	Type        string  `json:"type"`
	ZFSGuid     *string `json:"zfs_guid"`
	Bus         string  `json:"bus"`
	Devname     string  `json:"devname"`
	Pool        *string `json:"pool"`
}

// ListDisks retrieves all disks.
func (c *Client) ListDisks(ctx context.Context) ([]Disk, error) {
	tflog.Trace(ctx, "ListDisks start")

	resp, err := c.Get(ctx, "/disk")
	if err != nil {
		return nil, fmt.Errorf("listing disks: %w", err)
	}

	var disks []Disk
	if err := json.Unmarshal(resp, &disks); err != nil {
		return nil, fmt.Errorf("parsing disks list: %w", err)
	}

	tflog.Trace(ctx, "ListDisks success")
	return disks, nil
}

// GetDisk retrieves a disk by identifier.
func (c *Client) GetDisk(ctx context.Context, id string) (*Disk, error) {
	tflog.Trace(ctx, "GetDisk start")

	resp, err := c.Get(ctx, "/disk/id/"+url.PathEscape(id))
	if err != nil {
		return nil, fmt.Errorf("getting disk %q: %w", id, err)
	}

	var disk Disk
	if err := json.Unmarshal(resp, &disk); err != nil {
		return nil, fmt.Errorf("parsing disk response: %w", err)
	}

	tflog.Trace(ctx, "GetDisk success")
	return &disk, nil
}
