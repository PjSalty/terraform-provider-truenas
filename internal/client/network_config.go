package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// NetworkConfig and FullNetworkConfig (with their update requests) moved to
// internal/types/network_config.go in the v2.0 transport-migration prep.
type (
	NetworkConfig                  = types.NetworkConfig
	NetworkConfigUpdateRequest     = types.NetworkConfigUpdateRequest
	FullNetworkConfig              = types.FullNetworkConfig
	FullNetworkConfigUpdateRequest = types.FullNetworkConfigUpdateRequest
)

// GetNetworkConfig retrieves the network configuration.
func (c *Client) GetNetworkConfig(ctx context.Context) (*types.NetworkConfig, error) {
	tflog.Trace(ctx, "GetNetworkConfig start")

	resp, err := c.Get(ctx, "/network/configuration")
	if err != nil {
		return nil, fmt.Errorf("getting network configuration: %w", err)
	}

	var config types.NetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing network configuration response: %w", err)
	}

	tflog.Trace(ctx, "GetNetworkConfig success")
	return &config, nil
}

// UpdateNetworkConfig updates the network configuration.
func (c *Client) UpdateNetworkConfig(ctx context.Context, req *types.NetworkConfigUpdateRequest) (*types.NetworkConfig, error) {
	tflog.Trace(ctx, "UpdateNetworkConfig start")

	resp, err := c.Put(ctx, "/network/configuration", req)
	if err != nil {
		return nil, fmt.Errorf("updating network configuration: %w", err)
	}

	var config types.NetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing network configuration update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNetworkConfig success")
	return &config, nil
}

// GetFullNetworkConfig retrieves the full network configuration.
func (c *Client) GetFullNetworkConfig(ctx context.Context) (*types.FullNetworkConfig, error) {
	tflog.Trace(ctx, "GetFullNetworkConfig start")

	resp, err := c.Get(ctx, "/network/configuration")
	if err != nil {
		return nil, fmt.Errorf("getting full network configuration: %w", err)
	}

	var config types.FullNetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing full network configuration response: %w", err)
	}

	tflog.Trace(ctx, "GetFullNetworkConfig success")
	return &config, nil
}

// UpdateFullNetworkConfig updates the full network configuration.
func (c *Client) UpdateFullNetworkConfig(ctx context.Context, req *types.FullNetworkConfigUpdateRequest) (*types.FullNetworkConfig, error) {
	tflog.Trace(ctx, "UpdateFullNetworkConfig start")

	resp, err := c.Put(ctx, "/network/configuration", req)
	if err != nil {
		return nil, fmt.Errorf("updating full network configuration: %w", err)
	}

	var config types.FullNetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing full network configuration update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFullNetworkConfig success")
	return &config, nil
}
