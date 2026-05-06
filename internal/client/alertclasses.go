package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// AlertClassEntry, AlertClassesConfig, AlertClassesUpdateRequest moved to
// internal/types/alertclasses.go in the v2.0 transport-migration prep.
type (
	AlertClassEntry           = types.AlertClassEntry
	AlertClassesConfig        = types.AlertClassesConfig
	AlertClassesUpdateRequest = types.AlertClassesUpdateRequest
)

// GetAlertClassesConfig retrieves the alert classes configuration.
func (c *Client) GetAlertClassesConfig(ctx context.Context) (*types.AlertClassesConfig, error) {
	tflog.Trace(ctx, "GetAlertClassesConfig start")

	resp, err := c.Get(ctx, "/alertclasses")
	if err != nil {
		return nil, fmt.Errorf("getting alert classes config: %w", err)
	}

	var cfg types.AlertClassesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing alert classes config response: %w", err)
	}

	if cfg.Classes == nil {
		cfg.Classes = map[string]types.AlertClassEntry{}
	}

	tflog.Trace(ctx, "GetAlertClassesConfig success")
	return &cfg, nil
}

// UpdateAlertClassesConfig updates the alert classes configuration via PUT.
func (c *Client) UpdateAlertClassesConfig(ctx context.Context, req *types.AlertClassesUpdateRequest) (*types.AlertClassesConfig, error) {
	tflog.Trace(ctx, "UpdateAlertClassesConfig start")

	resp, err := c.Put(ctx, "/alertclasses", req)
	if err != nil {
		return nil, fmt.Errorf("updating alert classes config: %w", err)
	}

	var cfg types.AlertClassesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing alert classes update response: %w", err)
	}

	if cfg.Classes == nil {
		cfg.Classes = map[string]types.AlertClassEntry{}
	}

	tflog.Trace(ctx, "UpdateAlertClassesConfig success")
	return &cfg, nil
}
