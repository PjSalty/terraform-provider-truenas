package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// UpdateTrains, UpdateTrainInfo, UpdateCheckResult moved to
// internal/types/system_update.go in the v2.0 transport-migration prep.
// Re-exported here as aliases so existing test files and resources that
// import from internal/client/ still compile unchanged.
type (
	UpdateTrains      = types.UpdateTrains
	UpdateTrainInfo   = types.UpdateTrainInfo
	UpdateCheckResult = types.UpdateCheckResult
)

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
func (c *Client) GetUpdateTrains(ctx context.Context) (*types.UpdateTrains, error) {
	tflog.Trace(ctx, "GetUpdateTrains start")

	resp, err := c.Get(ctx, "/update/get_trains")
	if err != nil {
		return nil, fmt.Errorf("getting update trains: %w", err)
	}

	var trains types.UpdateTrains
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
// response is shaped as types.UpdateCheckResult; status UNAVAILABLE is the
// common case and must not be treated as an error by callers.
func (c *Client) CheckUpdateAvailable(ctx context.Context) (*types.UpdateCheckResult, error) {
	tflog.Trace(ctx, "CheckUpdateAvailable start")

	resp, err := c.Post(ctx, "/update/check_available", checkAvailableRequest{})
	if err != nil {
		return nil, fmt.Errorf("checking update availability: %w", err)
	}

	var result types.UpdateCheckResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing update check_available response: %w", err)
	}

	tflog.Trace(ctx, "CheckUpdateAvailable success")
	return &result, nil
}
