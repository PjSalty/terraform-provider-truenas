package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UPSConfig represents the UPS configuration.
type UPSConfig struct {
	ID            int    `json:"id"`
	Mode          string `json:"mode"`
	Identifier    string `json:"identifier"`
	Driver        string `json:"driver"`
	Port          string `json:"port"`
	RemoteHost    string `json:"remotehost"`
	RemotePort    int    `json:"remoteport"`
	Shutdown      string `json:"shutdown"`
	ShutdownTimer int    `json:"shutdowntimer"`
	Description   string `json:"description"`
}

// UPSConfigUpdateRequest represents the request to update UPS configuration.
type UPSConfigUpdateRequest struct {
	Mode          *string `json:"mode,omitempty"`
	Identifier    *string `json:"identifier,omitempty"`
	Driver        *string `json:"driver,omitempty"`
	Port          *string `json:"port,omitempty"`
	RemoteHost    *string `json:"remotehost,omitempty"`
	RemotePort    *int    `json:"remoteport,omitempty"`
	Shutdown      *string `json:"shutdown,omitempty"`
	ShutdownTimer *int    `json:"shutdowntimer,omitempty"`
	Description   *string `json:"description,omitempty"`
}

// GetUPSConfig retrieves the UPS configuration.
func (c *Client) GetUPSConfig(ctx context.Context) (*UPSConfig, error) {
	tflog.Trace(ctx, "GetUPSConfig start")

	resp, err := c.Get(ctx, "/ups")
	if err != nil {
		return nil, fmt.Errorf("getting UPS config: %w", err)
	}

	var config UPSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing UPS config response: %w", err)
	}

	tflog.Trace(ctx, "GetUPSConfig success")
	return &config, nil
}

// UpdateUPSConfig updates the UPS configuration.
func (c *Client) UpdateUPSConfig(ctx context.Context, req *UPSConfigUpdateRequest) (*UPSConfig, error) {
	tflog.Trace(ctx, "UpdateUPSConfig start")

	resp, err := c.Put(ctx, "/ups", req)
	if err != nil {
		return nil, fmt.Errorf("updating UPS config: %w", err)
	}

	var config UPSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing UPS config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateUPSConfig success")
	return &config, nil
}
