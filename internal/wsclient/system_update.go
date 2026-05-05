package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method names mirror the REST paths verbatim with `/` -> `.`,
// which is the documented TrueNAS convention. So /update/get_auto_download
// becomes update.get_auto_download, etc.

// GetUpdateAutoDownload returns whether auto-download of updates is enabled.
// The server returns a bare bool which we unmarshal directly.
func (c *Client) GetUpdateAutoDownload(ctx context.Context) (bool, error) {
	tflog.Trace(ctx, "GetUpdateAutoDownload (ws) start")

	result, err := c.Call(ctx, "update.get_auto_download", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return false, fmt.Errorf("getting update auto_download: %w", err)
	}

	var enabled bool
	if err := json.Unmarshal(result, &enabled); err != nil {
		return false, fmt.Errorf("parsing update auto_download response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateAutoDownload (ws) success")
	return enabled, nil
}

// SetUpdateAutoDownload sets the auto-download toggle.
func (c *Client) SetUpdateAutoDownload(ctx context.Context, enabled bool) error {
	tflog.Trace(ctx, "SetUpdateAutoDownload (ws) start")

	if _, err := c.Call(ctx, "update.set_auto_download",
		[]interface{}{enabled}, CallOptions{}); err != nil {
		return fmt.Errorf("setting update auto_download: %w", err)
	}

	tflog.Trace(ctx, "SetUpdateAutoDownload (ws) success")
	return nil
}

// GetUpdateTrains returns the list of available release trains plus the
// currently booted and currently selected train names.
func (c *Client) GetUpdateTrains(ctx context.Context) (*types.UpdateTrains, error) {
	tflog.Trace(ctx, "GetUpdateTrains (ws) start")

	result, err := c.Call(ctx, "update.get_trains", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting update trains: %w", err)
	}

	var trains types.UpdateTrains
	if err := json.Unmarshal(result, &trains); err != nil {
		return nil, fmt.Errorf("parsing update trains response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateTrains (ws) success")
	return &trains, nil
}

// SetUpdateTrain sets the active release train.
func (c *Client) SetUpdateTrain(ctx context.Context, train string) error {
	tflog.Trace(ctx, "SetUpdateTrain (ws) start")

	if _, err := c.Call(ctx, "update.set_train",
		[]interface{}{train}, CallOptions{}); err != nil {
		return fmt.Errorf("setting update train: %w", err)
	}

	tflog.Trace(ctx, "SetUpdateTrain (ws) success")
	return nil
}

// CheckUpdateAvailable queries the update server for a pending update. The
// response is shaped as types.UpdateCheckResult; status UNAVAILABLE is the
// common case and must not be treated as an error by callers.
//
// JSON-RPC params shape mirrors the REST POST body: a single object with
// an optional `train` field. We send an empty object so the server checks
// the currently selected train.
func (c *Client) CheckUpdateAvailable(ctx context.Context) (*types.UpdateCheckResult, error) {
	tflog.Trace(ctx, "CheckUpdateAvailable (ws) start")

	result, err := c.Call(ctx, "update.check_available",
		[]interface{}{map[string]interface{}{}}, CallOptions{
			Read:       true,
			Idempotent: true,
		})
	if err != nil {
		return nil, fmt.Errorf("checking update availability: %w", err)
	}

	var out types.UpdateCheckResult
	if err := json.Unmarshal(result, &out); err != nil {
		return nil, fmt.Errorf("parsing update check_available response: %w", err)
	}

	tflog.Trace(ctx, "CheckUpdateAvailable (ws) success")
	return &out, nil
}
