package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for the System > Advanced settings:
// system.advanced.{config,update}.

// GetSystemAdvanced retrieves the System > Advanced singleton.
func (c *Client) GetSystemAdvanced(ctx context.Context) (*types.SystemAdvanced, error) {
	tflog.Trace(ctx, "GetSystemAdvanced (ws) start")

	result, err := c.Call(ctx, "system.advanced.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting system advanced config: %w", err)
	}

	var cfg types.SystemAdvanced
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing system advanced config response: %w", err)
	}

	tflog.Trace(ctx, "GetSystemAdvanced (ws) success")
	return &cfg, nil
}

// UpdateSystemAdvanced applies the settings and returns the stored entry.
// system.advanced.update is a plain call, not a job: TrueNAS does its work
// (restarting syslog, cron, powerd or ssh, regenerating the boot menu) before
// it returns, so one call is the whole operation. It returns an error when a
// service restart fails after the new values were already saved.
func (c *Client) UpdateSystemAdvanced(ctx context.Context, req *types.SystemAdvancedUpdateRequest) (*types.SystemAdvanced, error) {
	tflog.Trace(ctx, "UpdateSystemAdvanced (ws) start")

	result, err := c.Call(ctx, "system.advanced.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating system advanced config: %w", err)
	}

	var cfg types.SystemAdvanced
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing system advanced update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSystemAdvanced (ws) success")
	return &cfg, nil
}
