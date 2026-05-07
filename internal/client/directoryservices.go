package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// DirectoryServicesConfig, DirectoryServicesUpdateRequest moved to
// internal/types/directoryservices.go in the v2.0 transport-migration
// prep.
type (
	DirectoryServicesConfig        = types.DirectoryServicesConfig
	DirectoryServicesUpdateRequest = types.DirectoryServicesUpdateRequest
)

// GetDirectoryServicesConfig retrieves the directory services config.
func (c *Client) GetDirectoryServicesConfig(ctx context.Context) (*DirectoryServicesConfig, error) {
	tflog.Trace(ctx, "GetDirectoryServicesConfig start")

	resp, err := c.Get(ctx, "/directoryservices")
	if err != nil {
		return nil, fmt.Errorf("getting directory services config: %w", err)
	}

	var cfg DirectoryServicesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing directory services response: %w", err)
	}
	tflog.Trace(ctx, "GetDirectoryServicesConfig success")
	return &cfg, nil
}

// UpdateDirectoryServicesConfig updates the directory services singleton.
func (c *Client) UpdateDirectoryServicesConfig(ctx context.Context, req *DirectoryServicesUpdateRequest) (*DirectoryServicesConfig, error) {
	tflog.Trace(ctx, "UpdateDirectoryServicesConfig start")

	resp, err := c.Put(ctx, "/directoryservices", req)
	if err != nil {
		return nil, fmt.Errorf("updating directory services config: %w", err)
	}

	// PUT may return a job ID (int) for AD join, or the config object.
	// If it's a number we re-fetch the config.
	var asInt int
	if err := json.Unmarshal(resp, &asInt); err == nil {
		return c.GetDirectoryServicesConfig(ctx)
	}

	var cfg DirectoryServicesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		// Fallback: re-fetch if we can't decode (polymorphic shape may confuse us).
		return c.GetDirectoryServicesConfig(ctx)
	}
	tflog.Trace(ctx, "UpdateDirectoryServicesConfig success")
	return &cfg, nil
}

// LeaveDirectoryServices instructs TrueNAS to leave the currently-joined
// directory service (used during AD disable flows). Returns the raw job
// response. Errors from the API are propagated unchanged.
func (c *Client) LeaveDirectoryServices(ctx context.Context, body map[string]interface{}) ([]byte, error) {
	tflog.Trace(ctx, "LeaveDirectoryServices start")

	resp, err := c.Post(ctx, "/directoryservices/leave", body)
	if err != nil {
		return nil, fmt.Errorf("leaving directory services: %w", err)
	}
	tflog.Trace(ctx, "LeaveDirectoryServices success")
	return resp, nil
}
