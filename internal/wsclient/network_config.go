package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for network config: network.configuration.{config,update}.

// GetNetworkConfig retrieves the network configuration.
func (c *Client) GetNetworkConfig(ctx context.Context) (*types.NetworkConfig, error) {
	tflog.Trace(ctx, "GetNetworkConfig (ws) start")

	result, err := c.Call(ctx, "network.configuration.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting network configuration: %w", err)
	}

	var config types.NetworkConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing network configuration response: %w", err)
	}

	tflog.Trace(ctx, "GetNetworkConfig (ws) success")
	return &config, nil
}

// UpdateNetworkConfig updates the network configuration.
func (c *Client) UpdateNetworkConfig(ctx context.Context, req *types.NetworkConfigUpdateRequest) (*types.NetworkConfig, error) {
	tflog.Trace(ctx, "UpdateNetworkConfig (ws) start")

	result, err := c.Call(ctx, "network.configuration.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating network configuration: %w", err)
	}

	var config types.NetworkConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing network configuration update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNetworkConfig (ws) success")
	return &config, nil
}

// GetFullNetworkConfig retrieves the full network configuration.
func (c *Client) GetFullNetworkConfig(ctx context.Context) (*types.FullNetworkConfig, error) {
	tflog.Trace(ctx, "GetFullNetworkConfig (ws) start")

	result, err := c.Call(ctx, "network.configuration.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting full network configuration: %w", err)
	}

	var config types.FullNetworkConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing full network configuration response: %w", err)
	}

	tflog.Trace(ctx, "GetFullNetworkConfig (ws) success")
	return &config, nil
}

// UpdateFullNetworkConfig updates the full network configuration.
func (c *Client) UpdateFullNetworkConfig(ctx context.Context, req *types.FullNetworkConfigUpdateRequest) (*types.FullNetworkConfig, error) {
	tflog.Trace(ctx, "UpdateFullNetworkConfig (ws) start")

	result, err := c.Call(ctx, "network.configuration.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating full network configuration: %w", err)
	}

	var config types.FullNetworkConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing full network configuration update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFullNetworkConfig (ws) success")
	return &config, nil
}
