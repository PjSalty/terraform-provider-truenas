package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UpdateTrains represents the response from /update/get_trains — the map of
// available release trains plus the currently booted train and the train
// selected for update tracking. The `selected` field is what the provider
// reconciles against when the user sets `train` on truenas_system_update;
// `current` is what the box is actually booted on and is surfaced as a
// read-only computed attribute for drift visibility.
type UpdateTrains struct {
	Trains   map[string]UpdateTrainInfo `json:"trains"`
	Current  string                     `json:"current"`
	Selected string                     `json:"selected"`
}

// UpdateTrainInfo is the per-train metadata returned inside UpdateTrains.Trains.
type UpdateTrainInfo struct {
	Description string `json:"description"`
}

// UpdateCheckResult is the response from POST /update/check_available.
// Status values documented in the TrueNAS OpenAPI spec:
//   - AVAILABLE: an update is available
//   - UNAVAILABLE: no update available
//   - REBOOT_REQUIRED: an update has already been applied, waiting for reboot
//   - HA_UNAVAILABLE: HA is non-functional
//
// Changes is left as raw JSON because the exact shape is non-stable across
// TrueNAS releases and the resource does not expose it to users — it is
// only surfaced indirectly via the computed `available_version` field.
type UpdateCheckResult struct {
	Status  string          `json:"status"`
	Version string          `json:"version,omitempty"`
	Changes json.RawMessage `json:"changes,omitempty"`
	Notes   string          `json:"notes,omitempty"`
}

// GetUpdateAutoDownload returns whether auto-download of updates is enabled.
// The TrueNAS endpoint returns a bare JSON boolean, not an object.
func (c *Client) GetUpdateAutoDownload(ctx context.Context) (bool, error) {
	tflog.Trace(ctx, "GetUpdateAutoDownload start")

	resp, err := c.Get(ctx, "/update/get_auto_download")
	if err != nil {
		return false, fmt.Errorf("getting update auto_download: %w", err)
	}

	var enabled bool
	if err := json.Unmarshal(resp, &enabled); err != nil {
		return false, fmt.Errorf("parsing update auto_download response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateAutoDownload success")
	return enabled, nil
}

// SetUpdateAutoDownload sets the auto-download toggle. The TrueNAS endpoint
// expects the request body to be a bare JSON boolean.
func (c *Client) SetUpdateAutoDownload(ctx context.Context, enabled bool) error {
	tflog.Trace(ctx, "SetUpdateAutoDownload start")

	if _, err := c.Post(ctx, "/update/set_auto_download", enabled); err != nil {
		return fmt.Errorf("setting update auto_download: %w", err)
	}

	tflog.Trace(ctx, "SetUpdateAutoDownload success")
	return nil
}

// GetUpdateTrains returns the list of available release trains plus the
// currently booted and currently selected train names.
func (c *Client) GetUpdateTrains(ctx context.Context) (*UpdateTrains, error) {
	tflog.Trace(ctx, "GetUpdateTrains start")

	resp, err := c.Get(ctx, "/update/get_trains")
	if err != nil {
		return nil, fmt.Errorf("getting update trains: %w", err)
	}

	var trains UpdateTrains
	if err := json.Unmarshal(resp, &trains); err != nil {
		return nil, fmt.Errorf("parsing update trains response: %w", err)
	}

	tflog.Trace(ctx, "GetUpdateTrains success")
	return &trains, nil
}

// SetUpdateTrain sets the active release train. The TrueNAS endpoint expects
// the request body to be a bare JSON string.
func (c *Client) SetUpdateTrain(ctx context.Context, train string) error {
	tflog.Trace(ctx, "SetUpdateTrain start")

	if _, err := c.Post(ctx, "/update/set_train", train); err != nil {
		return fmt.Errorf("setting update train: %w", err)
	}

	tflog.Trace(ctx, "SetUpdateTrain success")
	return nil
}

// checkAvailableRequest is the POST body for /update/check_available. The
// train field is optional — when empty, TrueNAS checks the currently
// selected train. We always send an empty object because the provider's
// purpose is to observe the state of the currently pinned train, not to
// probe hypothetical others.
type checkAvailableRequest struct {
	Train string `json:"train,omitempty"`
}

// CheckUpdateAvailable queries the update server for a pending update. The
// response is shaped as UpdateCheckResult; status UNAVAILABLE is the common
// case and must not be treated as an error by callers.
func (c *Client) CheckUpdateAvailable(ctx context.Context) (*UpdateCheckResult, error) {
	tflog.Trace(ctx, "CheckUpdateAvailable start")

	resp, err := c.Post(ctx, "/update/check_available", checkAvailableRequest{})
	if err != nil {
		return nil, fmt.Errorf("checking update availability: %w", err)
	}

	var result UpdateCheckResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing update check_available response: %w", err)
	}

	tflog.Trace(ctx, "CheckUpdateAvailable success")
	return &result, nil
}
