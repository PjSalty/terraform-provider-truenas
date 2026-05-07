package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for disk reads:
// disk.{query, get_instance}.
//
// Disks are read-only; TrueNAS does not expose mutate methods on this
// surface. The provider uses the Disk wsclient only from the disk
// data source.

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

// GetDisk retrieves a disk by identifier.
func (c *Client) GetDisk(ctx context.Context, id string) (*types.Disk, error) {
	tflog.Trace(ctx, "GetDisk (ws) start")

	result, err := c.Call(ctx, "disk.get_instance",
		[]interface{}{id},
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
