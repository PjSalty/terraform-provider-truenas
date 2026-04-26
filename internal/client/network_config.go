package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Network Configuration API ---

// NetworkConfig represents the network configuration in TrueNAS.
type NetworkConfig struct {
	ID          int    `json:"id"`
	Hostname    string `json:"hostname"`
	Domain      string `json:"domain"`
	Nameserver1 string `json:"nameserver1"`
	Nameserver2 string `json:"nameserver2"`
	Nameserver3 string `json:"nameserver3"`
	IPv4Gateway string `json:"ipv4gateway"`
	HTTPProxy   string `json:"httpproxy"`
}

// NetworkConfigUpdateRequest represents the request to update network configuration.
type NetworkConfigUpdateRequest struct {
	Nameserver1 *string `json:"nameserver1,omitempty"`
	Nameserver2 *string `json:"nameserver2,omitempty"`
	Nameserver3 *string `json:"nameserver3,omitempty"`
}

// GetNetworkConfig retrieves the network configuration.
func (c *Client) GetNetworkConfig(ctx context.Context) (*NetworkConfig, error) {
	tflog.Trace(ctx, "GetNetworkConfig start")

	resp, err := c.Get(ctx, "/network/configuration")
	if err != nil {
		return nil, fmt.Errorf("getting network configuration: %w", err)
	}

	var config NetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing network configuration response: %w", err)
	}

	tflog.Trace(ctx, "GetNetworkConfig success")
	return &config, nil
}

// UpdateNetworkConfig updates the network configuration.
func (c *Client) UpdateNetworkConfig(ctx context.Context, req *NetworkConfigUpdateRequest) (*NetworkConfig, error) {
	tflog.Trace(ctx, "UpdateNetworkConfig start")

	resp, err := c.Put(ctx, "/network/configuration", req)
	if err != nil {
		return nil, fmt.Errorf("updating network configuration: %w", err)
	}

	var config NetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing network configuration update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNetworkConfig success")
	return &config, nil
}

// --- Full Network Configuration Resource API ---

// FullNetworkConfig represents the full network configuration in TrueNAS.
type FullNetworkConfig struct {
	ID          int      `json:"id"`
	Hostname    string   `json:"hostname"`
	Domain      string   `json:"domain"`
	IPv4Gateway string   `json:"ipv4gateway"`
	IPv6Gateway string   `json:"ipv6gateway"`
	Nameserver1 string   `json:"nameserver1"`
	Nameserver2 string   `json:"nameserver2"`
	Nameserver3 string   `json:"nameserver3"`
	HTTPProxy   string   `json:"httpproxy"`
	Hosts       []string `json:"hosts"`
}

// FullNetworkConfigUpdateRequest represents the request to update the full network configuration.
type FullNetworkConfigUpdateRequest struct {
	Hostname    *string  `json:"hostname,omitempty"`
	Domain      *string  `json:"domain,omitempty"`
	IPv4Gateway *string  `json:"ipv4gateway,omitempty"`
	IPv6Gateway *string  `json:"ipv6gateway,omitempty"`
	Nameserver1 *string  `json:"nameserver1,omitempty"`
	Nameserver2 *string  `json:"nameserver2,omitempty"`
	Nameserver3 *string  `json:"nameserver3,omitempty"`
	HTTPProxy   *string  `json:"httpproxy,omitempty"`
	Hosts       []string `json:"hosts,omitempty"`
}

// GetFullNetworkConfig retrieves the full network configuration.
func (c *Client) GetFullNetworkConfig(ctx context.Context) (*FullNetworkConfig, error) {
	tflog.Trace(ctx, "GetFullNetworkConfig start")

	resp, err := c.Get(ctx, "/network/configuration")
	if err != nil {
		return nil, fmt.Errorf("getting full network configuration: %w", err)
	}

	var config FullNetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing full network configuration response: %w", err)
	}

	tflog.Trace(ctx, "GetFullNetworkConfig success")
	return &config, nil
}

// UpdateFullNetworkConfig updates the full network configuration.
func (c *Client) UpdateFullNetworkConfig(ctx context.Context, req *FullNetworkConfigUpdateRequest) (*FullNetworkConfig, error) {
	tflog.Trace(ctx, "UpdateFullNetworkConfig start")

	resp, err := c.Put(ctx, "/network/configuration", req)
	if err != nil {
		return nil, fmt.Errorf("updating full network configuration: %w", err)
	}

	var config FullNetworkConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing full network configuration update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFullNetworkConfig success")
	return &config, nil
}
