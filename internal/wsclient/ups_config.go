package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for UPS service: ups.{config,update}.

// GetUPSConfig retrieves the UPS service configuration.
func (c *Client) GetUPSConfig(ctx context.Context) (*types.UPSConfig, error) {
	tflog.Trace(ctx, "GetUPSConfig (ws) start")

	result, err := c.Call(ctx, "ups.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting UPS config: %w", err)
	}

	var config types.UPSConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing UPS config response: %w", err)
	}

	tflog.Trace(ctx, "GetUPSConfig (ws) success")
	return &config, nil
}

// UpdateUPSConfig updates the UPS service configuration.
func (c *Client) UpdateUPSConfig(ctx context.Context, req *types.UPSConfigUpdateRequest) (*types.UPSConfig, error) {
	tflog.Trace(ctx, "UpdateUPSConfig (ws) start")

	result, err := c.Call(ctx, "ups.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating UPS config: %w", err)
	}

	var config types.UPSConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing UPS config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateUPSConfig (ws) success")
	return &config, nil
}
