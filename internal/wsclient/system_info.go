package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// GetSystemInfo retrieves system information via system.info JSON-RPC.
// Read-only and idempotent.
func (c *Client) GetSystemInfo(ctx context.Context) (*types.SystemInfo, error) {
	tflog.Trace(ctx, "GetSystemInfo (ws) start")

	result, err := c.Call(ctx, "system.info", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting system info: %w", err)
	}

	var info types.SystemInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("parsing system info response: %w", err)
	}

	tflog.Trace(ctx, "GetSystemInfo (ws) success")
	return &info, nil
}
