package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for the directory services singleton:
// directoryservices.{config, update, leave}.
//
// directoryservices.update is non-job for non-AD changes but can be a
// job for AD join (which talks to the domain controller). Mirroring
// the REST client, we attempt a regular Call and fall back to a
// re-fetch on shape uncertainty.

// GetDirectoryServicesConfig retrieves the directory services config.
func (c *Client) GetDirectoryServicesConfig(ctx context.Context) (*types.DirectoryServicesConfig, error) {
	tflog.Trace(ctx, "GetDirectoryServicesConfig (ws) start")

	result, err := c.Call(ctx, "directoryservices.config", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting directory services config: %w", err)
	}

	var cfg types.DirectoryServicesConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing directory services response: %w", err)
	}

	tflog.Trace(ctx, "GetDirectoryServicesConfig (ws) success")
	return &cfg, nil
}

// UpdateDirectoryServicesConfig updates the directory services
// singleton. The server may return either the placed config object or
// a job ID (for AD-join flows that fork off a background process); we
// attempt to decode the config directly, and on any decode trouble
// we re-fetch via directoryservices.config.
func (c *Client) UpdateDirectoryServicesConfig(ctx context.Context, req *types.DirectoryServicesUpdateRequest) (*types.DirectoryServicesConfig, error) {
	tflog.Trace(ctx, "UpdateDirectoryServicesConfig (ws) start")

	result, err := c.Call(ctx, "directoryservices.update",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating directory services config: %w", err)
	}

	// Try the config-object shape first.
	var cfg types.DirectoryServicesConfig
	if err := json.Unmarshal(result, &cfg); err == nil && cfg.ID != 0 {
		tflog.Trace(ctx, "UpdateDirectoryServicesConfig (ws) success (from update)")
		return &cfg, nil
	}

	// Fall back to a re-fetch (covers job-id-int responses and
	// polymorphic shapes the direct decode misses).
	tflog.Trace(ctx, "UpdateDirectoryServicesConfig (ws) success (refetch)")
	return c.GetDirectoryServicesConfig(ctx)
}

// LeaveDirectoryServices instructs TrueNAS to leave the currently-
// joined directory service. Returns the raw server result for the
// caller to handle (matches REST behavior).
func (c *Client) LeaveDirectoryServices(ctx context.Context, body map[string]interface{}) ([]byte, error) {
	tflog.Trace(ctx, "LeaveDirectoryServices (ws) start")

	result, err := c.Call(ctx, "directoryservices.leave",
		[]interface{}{body},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("leaving directory services: %w", err)
	}

	tflog.Trace(ctx, "LeaveDirectoryServices (ws) success")
	return []byte(result), nil
}
