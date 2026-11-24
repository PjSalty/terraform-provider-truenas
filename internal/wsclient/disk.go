package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for disks: disk.{query, get_instance, update}.
//
// TrueNAS has no create or delete for a disk, only update of the stored
// settings. The provider never calls the other write methods, disk.wipe
// above all.

// ListDisks retrieves all disks.
func (c *Client) ListDisks(ctx context.Context) ([]types.Disk, error) {
	tflog.Trace(ctx, "ListDisks (ws) start")

	result, err := c.Call(ctx, "disk.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing disks: %w", err)
	}

	var disks []types.Disk
	if err := json.Unmarshal(result, &disks); err != nil {
		return nil, fmt.Errorf("parsing disks list: %w", err)
	}

	tflog.Trace(ctx, "ListDisks (ws) success")
	return disks, nil
}

// GetDisk retrieves a disk by identifier. It asks for the pool join: without
// extra.pools disk.query reports every disk's pool as null.
func (c *Client) GetDisk(ctx context.Context, id string) (*types.Disk, error) {
	tflog.Trace(ctx, "GetDisk (ws) start")

	result, err := c.Call(ctx, "disk.get_instance",
		[]interface{}{id, map[string]interface{}{
			"extra": map[string]interface{}{"pools": true},
		}},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting disk %q: %w", id, err)
	}

	var disk types.Disk
	if err := json.Unmarshal(result, &disk); err != nil {
		return nil, fmt.Errorf("parsing disk response: %w", err)
	}

	tflog.Trace(ctx, "GetDisk (ws) success")
	return &disk, nil
}

// UpdateDisk writes the settings in req to the disk with the given
// identifier. A change to the standby timer or the power management level
// makes TrueNAS run hdparm against the drive in the background, ignoring
// its exit status. The reply is not returned: disk.update answers without
// the pool join, so callers read the disk again with GetDisk.
func (c *Client) UpdateDisk(ctx context.Context, id string, req *types.DiskUpdateRequest) error {
	tflog.Trace(ctx, "UpdateDisk (ws) start")

	if _, err := c.Call(ctx, "disk.update", []interface{}{id, req}, CallOptions{}); err != nil {
		return fmt.Errorf("updating disk %q: %w", id, err)
	}

	tflog.Trace(ctx, "UpdateDisk (ws) success")
	return nil
}
