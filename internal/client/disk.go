package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- Disk API (read-only) ---

// Disk moved to internal/types/disk.go in the v2.0 transport-migration
// prep.
type Disk = types.Disk

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
