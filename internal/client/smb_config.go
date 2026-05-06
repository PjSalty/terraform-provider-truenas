package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// SMBConfig, SMBConfigUpdateRequest moved to internal/types/smb_config.go
// in the v2.0 transport-migration prep.
type (
	SMBConfig              = types.SMBConfig
	SMBConfigUpdateRequest = types.SMBConfigUpdateRequest
)

// GetSMBConfig retrieves the SMB service configuration.
func (c *Client) GetSMBConfig(ctx context.Context) (*types.SMBConfig, error) {
	tflog.Trace(ctx, "GetSMBConfig start")

	resp, err := c.Get(ctx, "/smb")
	if err != nil {
		return nil, fmt.Errorf("getting SMB config: %w", err)
	}

	var config types.SMBConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config response: %w", err)
	}

	tflog.Trace(ctx, "GetSMBConfig success")
	return &config, nil
}

// UpdateSMBConfig updates the SMB service configuration.
func (c *Client) UpdateSMBConfig(ctx context.Context, req *types.SMBConfigUpdateRequest) (*types.SMBConfig, error) {
	tflog.Trace(ctx, "UpdateSMBConfig start")

	resp, err := c.Put(ctx, "/smb", req)
	if err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}

	var config types.SMBConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSMBConfig success")
	return &config, nil
}
