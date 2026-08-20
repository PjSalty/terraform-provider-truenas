package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for the system update configuration:
// update.{config, update, profile_choices, status}.
//
// This file previously called update.{get_auto_download, set_auto_download,
// get_trains, set_train, check_available}. None of those exist. They went
// during the 25.10.0 migration of the update service to a config service, and
// they have no Args model in any versioned API directory, so @api_method never
// registered them. truenas_system_update was therefore unusable on every
// release the provider supports: Read called get_auto_download first, so a
// plain `terraform plan` died with -32601 before reaching anything else.
// Tracked as issue #32; found by scripts/api-drift.sh.
//
// The replacement surface is present on 25.10, 26.0 and 27.0 alike, so one
// code path covers every supported train.
//
// Beware when checking this by hand: a module-level `def get_trains` DOES
// exist in plugins/update_/trains.py, but as a plain helper rather than a
// service method, so grepping for the definition suggests it is alive.

// GetUpdateConfig retrieves the system update configuration.
func (c *Client) GetUpdateConfig(ctx context.Context) (*types.UpdateConfig, error) {
	tflog.Trace(ctx, "GetUpdateConfig (ws) start")

	result, err := c.Call(ctx, "update.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting update config: %w", err)
	}

	var cfg types.UpdateConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing update config response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateConfig (ws) success")
	return &cfg, nil
}

// SetUpdateConfig applies the system update configuration.
//
// update.update takes a single positional object and returns the full entry,
// which the caller maps straight into state.
func (c *Client) SetUpdateConfig(ctx context.Context, req *types.UpdateConfigUpdateRequest) (*types.UpdateConfig, error) {
	tflog.Trace(ctx, "SetUpdateConfig (ws) start")

	result, err := c.Call(ctx, "update.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating update config: %w", err)
	}

	var cfg types.UpdateConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing update config update response: %w", err)
	}

	tflog.Trace(ctx, "SetUpdateConfig (ws) success")
	return &cfg, nil
}

// GetUpdateProfileChoices lists the selectable update profiles.
//
// Used to validate `profile` client-side. Middleware rejects a profile that is
// not marked available, so checking here turns that into a diagnostic naming
// the valid choices instead of an opaque validation error.
func (c *Client) GetUpdateProfileChoices(ctx context.Context) (map[string]types.UpdateProfileChoice, error) {
	tflog.Trace(ctx, "GetUpdateProfileChoices (ws) start")

	result, err := c.Call(ctx, "update.profile_choices", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting update profile choices: %w", err)
	}

	var choices map[string]types.UpdateProfileChoice
	if err := json.Unmarshal(result, &choices); err != nil {
		return nil, fmt.Errorf("parsing update profile choices response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateProfileChoices (ws) success")
	return choices, nil
}

// GetUpdateStatus reports the running version and any available update.
//
// Replaces update.check_available. `status` and `error` are nullable, so the
// type uses pointers rather than bare structs.
func (c *Client) GetUpdateStatus(ctx context.Context) (*types.UpdateStatus, error) {
	tflog.Trace(ctx, "GetUpdateStatus (ws) start")

	result, err := c.Call(ctx, "update.status", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting update status: %w", err)
	}

	var st types.UpdateStatus
	if err := json.Unmarshal(result, &st); err != nil {
		return nil, fmt.Errorf("parsing update status response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateStatus (ws) success")
	return &st, nil
}
