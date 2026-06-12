package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// alertclasses is a singleton config; methods follow the TrueNAS
// JSON-RPC convention for singletons: alertclasses.config / .update.

// GetAlertClassesConfig retrieves the alert classes configuration.
func (c *Client) GetAlertClassesConfig(ctx context.Context) (*types.AlertClassesConfig, error) {
	tflog.Trace(ctx, "GetAlertClassesConfig (ws) start")

	result, err := c.Call(ctx, "alertclasses.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting alert classes config: %w", err)
	}

	var cfg types.AlertClassesConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing alert classes config response: %w", err)
	}

	if cfg.Classes == nil {
		cfg.Classes = map[string]types.AlertClassEntry{}
	}

	tflog.Trace(ctx, "GetAlertClassesConfig (ws) success")
	return &cfg, nil
}

// UpdateAlertClassesConfig updates the alert classes configuration via
// alertclasses.update with a single object param.
func (c *Client) UpdateAlertClassesConfig(ctx context.Context, req *types.AlertClassesUpdateRequest) (*types.AlertClassesConfig, error) {
	tflog.Trace(ctx, "UpdateAlertClassesConfig (ws) start")

	result, err := c.Call(ctx, "alertclasses.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating alert classes config: %w", err)
	}

	var cfg types.AlertClassesConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing alert classes update response: %w", err)
	}

	if cfg.Classes == nil {
		cfg.Classes = map[string]types.AlertClassEntry{}
	}

	tflog.Trace(ctx, "UpdateAlertClassesConfig (ws) success")
	return &cfg, nil
}
