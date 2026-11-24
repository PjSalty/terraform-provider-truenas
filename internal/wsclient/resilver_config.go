package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for the resilver priority window:
// pool.resilver.{config,update}.

// GetResilverConfig retrieves the resilver priority singleton.
func (c *Client) GetResilverConfig(ctx context.Context) (*types.ResilverConfig, error) {
	tflog.Trace(ctx, "GetResilverConfig (ws) start")

	result, err := c.Call(ctx, "pool.resilver.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting resilver config: %w", err)
	}

	var cfg types.ResilverConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing resilver config response: %w", err)
	}

	tflog.Trace(ctx, "GetResilverConfig (ws) success")
	return &cfg, nil
}

// UpdateResilverConfig applies the resilver priority window and returns the
// stored entry. A change makes TrueNAS restart cron and write the ZFS
// resilver tunables for the current time.
func (c *Client) UpdateResilverConfig(ctx context.Context, req *types.ResilverConfigUpdateRequest) (*types.ResilverConfig, error) {
	tflog.Trace(ctx, "UpdateResilverConfig (ws) start")

	result, err := c.Call(ctx, "pool.resilver.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating resilver config: %w", err)
	}

	var cfg types.ResilverConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing resilver config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateResilverConfig (ws) success")
	return &cfg, nil
}
