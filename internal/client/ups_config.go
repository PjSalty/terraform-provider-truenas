package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// UPSConfig, UPSConfigUpdateRequest moved to internal/types/ups_config.go
// in the v2.0 transport-migration prep.
type (
	UPSConfig              = types.UPSConfig
	UPSConfigUpdateRequest = types.UPSConfigUpdateRequest
)

// GetUPSConfig retrieves the UPS configuration.
func (c *Client) GetUPSConfig(ctx context.Context) (*types.UPSConfig, error) {
	tflog.Trace(ctx, "GetUPSConfig start")

	resp, err := c.Get(ctx, "/ups")
	if err != nil {
		return nil, fmt.Errorf("getting UPS config: %w", err)
	}

	var config types.UPSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing UPS config response: %w", err)
	}

	tflog.Trace(ctx, "GetUPSConfig success")
	return &config, nil
}

// UpdateUPSConfig updates the UPS configuration.
func (c *Client) UpdateUPSConfig(ctx context.Context, req *types.UPSConfigUpdateRequest) (*types.UPSConfig, error) {
	tflog.Trace(ctx, "UpdateUPSConfig start")

	resp, err := c.Put(ctx, "/ups", req)
	if err != nil {
		return nil, fmt.Errorf("updating UPS config: %w", err)
	}

	var config types.UPSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing UPS config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateUPSConfig success")
	return &config, nil
}
