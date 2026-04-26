package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AlertClassEntry is the per-class configuration.
type AlertClassEntry struct {
	Level            string `json:"level,omitempty"`
	Policy           string `json:"policy,omitempty"`
	ProactiveSupport *bool  `json:"proactive_support,omitempty"`
}

// AlertClassesConfig is the singleton alert classes configuration.
type AlertClassesConfig struct {
	ID      int                        `json:"id"`
	Classes map[string]AlertClassEntry `json:"classes"`
}

// AlertClassesUpdateRequest is the update payload.
type AlertClassesUpdateRequest struct {
	Classes map[string]AlertClassEntry `json:"classes"`
}

// GetAlertClassesConfig retrieves the alert classes configuration.
func (c *Client) GetAlertClassesConfig(ctx context.Context) (*AlertClassesConfig, error) {
	tflog.Trace(ctx, "GetAlertClassesConfig start")

	resp, err := c.Get(ctx, "/alertclasses")
	if err != nil {
		return nil, fmt.Errorf("getting alert classes config: %w", err)
	}

	var cfg AlertClassesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing alert classes config response: %w", err)
	}

	if cfg.Classes == nil {
		cfg.Classes = map[string]AlertClassEntry{}
	}

	tflog.Trace(ctx, "GetAlertClassesConfig success")
	return &cfg, nil
}

// UpdateAlertClassesConfig updates the alert classes configuration via PUT.
func (c *Client) UpdateAlertClassesConfig(ctx context.Context, req *AlertClassesUpdateRequest) (*AlertClassesConfig, error) {
	tflog.Trace(ctx, "UpdateAlertClassesConfig start")

	resp, err := c.Put(ctx, "/alertclasses", req)
	if err != nil {
		return nil, fmt.Errorf("updating alert classes config: %w", err)
	}

	var cfg AlertClassesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing alert classes update response: %w", err)
	}

	if cfg.Classes == nil {
		cfg.Classes = map[string]AlertClassEntry{}
	}

	tflog.Trace(ctx, "UpdateAlertClassesConfig success")
	return &cfg, nil
}
