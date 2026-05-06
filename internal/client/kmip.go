package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// KMIPConfig, KMIPUpdateRequest moved to internal/types/kmip.go in the
// v2.0 transport-migration prep.
type (
	KMIPConfig        = types.KMIPConfig
	KMIPUpdateRequest = types.KMIPUpdateRequest
)

// GetKMIPConfig retrieves the KMIP configuration.
func (c *Client) GetKMIPConfig(ctx context.Context) (*types.KMIPConfig, error) {
	tflog.Trace(ctx, "GetKMIPConfig start")

	resp, err := c.Get(ctx, "/kmip")
	if err != nil {
		return nil, fmt.Errorf("getting KMIP config: %w", err)
	}

	var cfg types.KMIPConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing KMIP config response: %w", err)
	}

	tflog.Trace(ctx, "GetKMIPConfig success")
	return &cfg, nil
}

// UpdateKMIPConfig updates the KMIP configuration via PUT.
func (c *Client) UpdateKMIPConfig(ctx context.Context, req *types.KMIPUpdateRequest) (*types.KMIPConfig, error) {
	tflog.Trace(ctx, "UpdateKMIPConfig start")

	resp, err := c.Put(ctx, "/kmip", req)
	if err != nil {
		return nil, fmt.Errorf("updating KMIP config: %w", err)
	}

	// TrueNAS may return either the updated object directly or a job ID
	// for async completion. Try decoding as the config first.
	var cfg types.KMIPConfig
	if err := json.Unmarshal(resp, &cfg); err == nil && cfg.ID != 0 {
		return &cfg, nil
	}

	// Re-fetch the canonical state.
	tflog.Trace(ctx, "UpdateKMIPConfig success")
	return c.GetKMIPConfig(ctx)
}
