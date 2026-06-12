package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for KMIP: kmip.{config,update}.

// GetKMIPConfig retrieves the KMIP configuration.
func (c *Client) GetKMIPConfig(ctx context.Context) (*types.KMIPConfig, error) {
	tflog.Trace(ctx, "GetKMIPConfig (ws) start")

	result, err := c.Call(ctx, "kmip.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting KMIP config: %w", err)
	}

	var cfg types.KMIPConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing KMIP config response: %w", err)
	}

	tflog.Trace(ctx, "GetKMIPConfig (ws) success")
	return &cfg, nil
}

// UpdateKMIPConfig updates the KMIP configuration. The TrueNAS server
// may return either the updated object directly or a job ID for async
// completion; if the immediate response doesn't decode as a config, we
// re-fetch the canonical state.
func (c *Client) UpdateKMIPConfig(ctx context.Context, req *types.KMIPUpdateRequest) (*types.KMIPConfig, error) {
	tflog.Trace(ctx, "UpdateKMIPConfig (ws) start")

	result, err := c.Call(ctx, "kmip.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating KMIP config: %w", err)
	}

	var cfg types.KMIPConfig
	if err := json.Unmarshal(result, &cfg); err == nil && cfg.ID != 0 {
		return &cfg, nil
	}

	tflog.Trace(ctx, "UpdateKMIPConfig (ws) success")
	return c.GetKMIPConfig(ctx)
}
