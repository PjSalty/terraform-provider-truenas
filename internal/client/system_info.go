package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- System Info API (read-only) ---

// SystemInfo moved to internal/types/system_update.go in the v2.0
// transport-migration prep. Aliased here for backward compatibility
// with existing imports.
type SystemInfo = types.SystemInfo

// GetSystemInfo retrieves system information.
func (c *Client) GetSystemInfo(ctx context.Context) (*types.SystemInfo, error) {
	tflog.Trace(ctx, "GetSystemInfo start")

	resp, err := c.Get(ctx, "/system/info")
	if err != nil {
		return nil, fmt.Errorf("getting system info: %w", err)
	}

	var info types.SystemInfo
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("parsing system info response: %w", err)
	}

	tflog.Trace(ctx, "GetSystemInfo success")
	return &info, nil
}
